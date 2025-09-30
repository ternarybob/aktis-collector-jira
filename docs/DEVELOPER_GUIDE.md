# Aktis Collector Jira - Developer Guide

## Overview

This guide helps developers understand the codebase and adapt it as a template for building collectors for other web applications.

## Project Structure

```
aktis-collector-jira/
├── cmd/
│   └── aktis-chrome-extension/      # Chrome extension source
│       ├── manifest.json             # Extension manifest
│       ├── background.js             # Service worker
│       ├── content.js                # Content script (runs on Jira pages)
│       ├── sidepanel.html            # UI layout
│       ├── sidepanel.js              # UI logic
│       └── icons/                    # Extension icons
│
├── internal/                         # Go server source
│   ├── common/                       # Shared utilities
│   │   ├── config.go                 # Configuration management
│   │   └── logger.go                 # Logging setup
│   │
│   ├── handlers/                     # HTTP handlers
│   │   ├── api.go                    # API endpoints (/receiver, /health, etc.)
│   │   ├── ui.go                     # Web UI handlers
│   │   └── jira_parser.go            # HTML parsing logic
│   │
│   ├── interfaces/                   # Data structures and interfaces
│   │   └── types.go                  # TicketData, ProjectData, etc.
│   │
│   └── services/                     # Business logic
│       ├── storage.go                # BoltDB operations
│       └── webserver.go              # HTTP server setup
│
├── cmd/aktis-collector-jira/
│   └── main.go                       # Application entry point
│
├── scripts/
│   └── build.ps1                     # Build script
│
├── docs/                             # Documentation
│   ├── ARCHITECTURE.md               # System architecture
│   ├── USER_GUIDE.md                 # User documentation
│   └── DEVELOPER_GUIDE.md            # This file
│
└── deployments/
    └── aktis-collector-jira.toml     # Configuration file
```

## Core Components

### 1. Chrome Extension

#### manifest.json
Defines extension metadata, permissions, and components.

```json
{
  "manifest_version": 3,
  "permissions": ["activeTab", "storage", "scripting", "sidePanel"],
  "host_permissions": ["https://*.atlassian.net/*"],
  "background": {
    "service_worker": "background.js"
  },
  "content_scripts": [{
    "matches": ["https://*.atlassian.net/*"],
    "js": ["content.js"]
  }],
  "side_panel": {
    "default_path": "sidepanel.html"
  }
}
```

**Key permissions:**
- `activeTab`: Access current tab when user clicks extension
- `storage`: Store configuration (server URL, settings)
- `scripting`: Execute scripts to collect HTML
- `sidePanel`: Display side panel UI

#### background.js (Service Worker)
Manages extension lifecycle and communication.

```javascript
// Default configuration
const DEFAULT_CONFIG = {
  serverUrl: 'http://localhost:8084',
  autoCollect: false,
  followLinks: false
};

// Handle messages from content script
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  if (request.type === 'PAGE_DATA') {
    handlePageData(request.data, sender.tab)
      .then(response => sendResponse({ success: true, response }))
      .catch(error => sendResponse({ success: false, error: error.message }));
    return true;
  }
});

// Send data to server
async function handlePageData(pageData, tab) {
  const serverUrl = `${config.serverUrl}/receiver`;

  const payload = {
    timestamp: new Date().toISOString(),
    url: tab.url,
    title: tab.title,
    data: pageData,
    collector: {
      name: 'aktis-jira-collector-extension',
      version: chrome.runtime.getManifest().version
    }
  };

  const response = await fetch(serverUrl, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });

  return await response.json();
}
```

#### content.js (Content Script)
Runs on target pages (Jira), detects page type, and collects HTML.

```javascript
// Detect what type of page we're on
function detectPageType() {
  const url = window.location.href;

  if (url.includes('/jira/projects') && !url.includes('/projects/')) {
    return 'projectsList';
  }
  if (url.includes('/browse/')) {
    return 'issue';
  }
  if (url.includes('/jira/software/c/projects/') && url.includes('/issues')) {
    return 'issueList';
  }

  return 'generic';
}

// Collect page data
async function collectCurrentPage() {
  const pageType = detectPageType();

  let pageData = {
    pageType: pageType,
    url: window.location.href,
    title: document.title,
    timestamp: new Date().toISOString(),
    html: document.documentElement.outerHTML
  };

  // Send to background script
  await chrome.runtime.sendMessage({
    type: 'PAGE_DATA',
    data: pageData
  });
}
```

#### sidepanel.js (UI Logic)
Provides user interface for collection control and viewing buffer.

```javascript
// Collect current page button handler
document.getElementById('collect-btn').addEventListener('click', async () => {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });

  try {
    // Try content script first
    const response = await chrome.tabs.sendMessage(tab.id, { type: 'COLLECT_PAGE' });

    if (response && response.success) {
      showMessage('Page collected successfully!');
    }
  } catch (error) {
    // Fallback: collect directly
    const pageData = await collectPageDirectly(tab);
    await sendToServer(pageData);
  }
});

// Check server status
async function checkServerStatus() {
  const response = await fetch(`${config.serverUrl}/health`);
  if (response.ok) {
    statusEl.textContent = 'Online';
  }
}

// Load buffer data
async function loadBufferData() {
  const response = await fetch(`${config.serverUrl}/database/data`);
  const data = await response.json();
  displayBufferData(data);
}
```

### 2. Go HTTP Server

#### main.go (Entry Point)
Initializes services and starts server.

```go
package main

import (
    "context"
    "flag"
    "os"
    "os/signal"
    "syscall"

    "aktis-collector-jira/internal/common"
    "aktis-collector-jira/internal/services"
)

func main() {
    // Load configuration
    cfg, err := common.LoadConfig("")
    if err != nil {
        log.Fatal(err)
    }

    // Initialize logger
    logger := common.CreateLogger(cfg)

    // Create storage service
    storage, err := services.NewStorage(cfg, logger)
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to initialize storage")
    }

    // Create web server
    webServer, err := services.NewWebServer(cfg, storage, logger)
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to create web server")
    }

    // Start server
    ctx := context.Background()
    if err := webServer.Start(ctx); err != nil {
        logger.Fatal().Err(err).Msg("Failed to start web server")
    }

    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan

    // Graceful shutdown
    webServer.Stop()
    storage.Close()
}
```

#### services/webserver.go (HTTP Server)
Sets up routes and middleware.

```go
package services

import (
    "net/http"

    "aktis-collector-jira/internal/handlers"
)

func NewWebServer(cfg *Config, storage Storage, logger Logger) (*webServer, error) {
    mux := http.NewServeMux()

    apiHandlers := handlers.NewAPIHandlers(cfg, storage, logger)
    uiHandlers := handlers.NewUIHandlers(cfg, storage, logger, "pages")

    // Register API endpoints
    mux.HandleFunc("/health", loggingMiddleware(apiHandlers.HealthHandler))
    mux.HandleFunc("/status", loggingMiddleware(apiHandlers.StatusHandler))
    mux.HandleFunc("/receiver", loggingMiddleware(apiHandlers.ReceiverHandler))
    mux.HandleFunc("/database", loggingMiddleware(apiHandlers.DatabaseHandler))

    // Register UI endpoints
    mux.HandleFunc("/", loggingMiddleware(uiHandlers.IndexHandler))
    mux.HandleFunc("/database/data", loggingMiddleware(uiHandlers.BufferDataHandler))

    server := &http.Server{
        Addr:    fmt.Sprintf(":%d", cfg.Collector.Port),
        Handler: mux,
    }

    return &webServer{server: server}, nil
}

// Logging middleware with CORS headers
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Add CORS headers for Chrome extension
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        next(w, r)
    }
}
```

#### handlers/api.go (API Handlers)
Processes incoming requests from extension.

```go
package handlers

// ReceiverHandler accepts data from Chrome extension
func (h *APIHandlers) ReceiverHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var payload ExtensionDataPayload
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        h.logger.Error().Err(err).Msg("Failed to decode extension data")
        sendError(w, "Invalid payload format", http.StatusBadRequest)
        return
    }

    h.logger.Info().
        Str("url", payload.URL).
        Str("page_type", payload.Data["pageType"].(string)).
        Msg("Received data from Chrome extension")

    // Store the received data
    if err := h.storeExtensionData(payload); err != nil {
        h.logger.Error().Err(err).Msg("Failed to store extension data")
        sendError(w, "Failed to store data", http.StatusInternalServerError)
        return
    }

    response := ReceiverResponse{
        Success:   true,
        Message:   "Successfully received and stored data",
        Timestamp: time.Now(),
    }

    json.NewEncoder(w).Encode(response)
}

// storeExtensionData processes and stores extension data
func (h *APIHandlers) storeExtensionData(payload ExtensionDataPayload) error {
    pageType := payload.Data["pageType"].(string)
    htmlContent := payload.Data["html"].(string)

    // Parse HTML on server side
    parser := NewJiraParser()
    results, err := parser.ParseHTML(htmlContent, pageType, payload.URL)
    if err != nil {
        return fmt.Errorf("failed to parse HTML: %w", err)
    }

    // Handle based on page type
    if pageType == "projectsList" {
        return h.storeProjects(results, payload.Timestamp)
    }

    return h.storeTickets(results, payload.Timestamp)
}
```

#### handlers/jira_parser.go (HTML Parser)
Extracts structured data from HTML.

```go
package handlers

import (
    "regexp"
    "golang.org/x/net/html"
)

type JiraParser struct{}

// ParseHTML parses Jira HTML and extracts data based on page type
func (p *JiraParser) ParseHTML(htmlContent, pageType, url string) ([]map[string]interface{}, error) {
    doc, err := html.Parse(strings.NewReader(htmlContent))
    if err != nil {
        return nil, err
    }

    switch pageType {
    case "projectsList":
        return p.parseProjectsListPage(doc, url)
    case "issue":
        return p.parseIssuePage(doc, url)
    case "issueList":
        return p.parseIssueListPage(doc, url)
    default:
        return p.parseGenericPage(doc, url)
    }
}

// parseProjectsListPage extracts projects from HTML
func (p *JiraParser) parseProjectsListPage(doc *html.Node, url string) ([]map[string]interface{}, error) {
    projects := []map[string]interface{}{}

    // Find all project rows
    projectRows := p.findProjectRows(doc)

    for _, row := range projectRows {
        project := p.extractProjectFromRow(row)
        if project != nil && project["key"] != nil {
            projects = append(projects, project)
        }
    }

    return projects, nil
}

// parseIssueListPage extracts tickets from list page
func (p *JiraParser) parseIssueListPage(doc *html.Node, url string) ([]map[string]interface{}, error) {
    issues := []map[string]interface{}{}
    projectKey := p.extractProjectKeyFromURL(url)

    // Find issue rows
    issueRows := p.findIssueRows(doc)

    for _, row := range issueRows {
        issue := p.extractIssueFromRow(row, projectKey)
        if issue != nil && issue["key"] != nil {
            issues = append(issues, issue)
        }
    }

    return issues, nil
}

// extractIssueKeys finds all Jira issue keys in HTML
func (p *JiraParser) extractIssueKeys(node *html.Node) map[string]bool {
    keys := make(map[string]bool)
    keyRegex := regexp.MustCompile(`\b([A-Z]+-\d+)\b`)

    var traverse func(*html.Node)
    traverse = func(n *html.Node) {
        if n.Type == html.TextNode {
            matches := keyRegex.FindAllString(n.Data, -1)
            for _, match := range matches {
                keys[match] = true
            }
        }

        for c := n.FirstChild; c != nil; c = c.NextSibling {
            traverse(c)
        }
    }

    traverse(node)
    return keys
}
```

#### services/storage.go (BoltDB Operations)
Manages database persistence.

```go
package services

import (
    bolt "go.etcd.io/bbolt"
)

type Storage interface {
    SaveProjects([]*ProjectData) error
    SaveTickets(projectKey string, map[string]*TicketData) error
    LoadTickets(projectKey string) (map[string]*TicketData, error)
    LoadAllTickets() (map[string]*TicketData, error)
    ClearAllTickets() error
}

type storage struct {
    db     *bolt.DB
    logger Logger
}

func NewStorage(cfg *Config, logger Logger) (*storage, error) {
    db, err := bolt.Open(cfg.Storage.DatabasePath, 0600, nil)
    if err != nil {
        return nil, err
    }

    // Create buckets
    err = db.Update(func(tx *bolt.Tx) error {
        if _, err := tx.CreateBucketIfNotExists([]byte("projects")); err != nil {
            return err
        }
        if _, err := tx.CreateBucketIfNotExists([]byte("tickets")); err != nil {
            return err
        }
        return nil
    })

    return &storage{db: db, logger: logger}, err
}

func (s *storage) SaveTickets(projectKey string, tickets map[string]*TicketData) error {
    return s.db.Update(func(tx *bolt.Tx) error {
        bucket := tx.Bucket([]byte("tickets"))

        // Create project-specific bucket
        projectBucket, err := bucket.CreateBucketIfNotExists([]byte(projectKey + "_tickets"))
        if err != nil {
            return err
        }

        // Store each ticket
        for key, ticket := range tickets {
            data, err := json.Marshal(ticket)
            if err != nil {
                return err
            }
            if err := projectBucket.Put([]byte(key), data); err != nil {
                return err
            }
        }

        return nil
    })
}

func (s *storage) LoadAllTickets() (map[string]*TicketData, error) {
    allTickets := make(map[string]*TicketData)

    err := s.db.View(func(tx *bolt.Tx) error {
        bucket := tx.Bucket([]byte("tickets"))

        return bucket.ForEach(func(k, v []byte) error {
            projectBucket := bucket.Bucket(k)
            if projectBucket == nil {
                return nil
            }

            return projectBucket.ForEach(func(ticketKey, ticketData []byte) error {
                var ticket TicketData
                if err := json.Unmarshal(ticketData, &ticket); err != nil {
                    return err
                }
                allTickets[string(ticketKey)] = &ticket
                return nil
            })
        })
    })

    return allTickets, err
}
```

## Adapting as a Template

### Step 1: Identify Target Website

Choose the website you want to collect data from:
- Confluence
- Trello
- GitHub
- Linear
- Notion
- Custom web app

### Step 2: Update Extension Manifest

Edit `cmd/aktis-chrome-extension/manifest.json`:

```json
{
  "name": "Aktis Collector [YOUR_APP]",
  "description": "Collects data from [YOUR_APP]",
  "host_permissions": [
    "https://*.yourapp.com/*",
    "http://localhost/*"
  ],
  "content_scripts": [{
    "matches": ["https://*.yourapp.com/*"],
    "js": ["content.js"]
  }]
}
```

### Step 3: Update Page Type Detection

Edit `cmd/aktis-chrome-extension/content.js`:

```javascript
function detectPageType() {
  const url = window.location.href;
  const path = window.location.pathname;

  // Add your app's URL patterns
  if (url.includes('/workspaces')) {
    return 'workspaceList';
  }
  if (url.includes('/documents/')) {
    return 'document';
  }
  if (path.startsWith('/board/')) {
    return 'board';
  }

  return 'generic';
}
```

### Step 4: Create Custom Parser

Create `internal/handlers/your_app_parser.go`:

```go
package handlers

type YourAppParser struct{}

func NewYourAppParser() *YourAppParser {
    return &YourAppParser{}
}

func (p *YourAppParser) ParseHTML(htmlContent, pageType, url string) ([]map[string]interface{}, error) {
    doc, err := html.Parse(strings.NewReader(htmlContent))
    if err != nil {
        return nil, err
    }

    switch pageType {
    case "workspaceList":
        return p.parseWorkspaces(doc, url)
    case "document":
        return p.parseDocument(doc, url)
    default:
        return []map[string]interface{}{}, nil
    }
}

func (p *YourAppParser) parseWorkspaces(doc *html.Node, url string) ([]map[string]interface{}, error) {
    // Your custom extraction logic
    // Look for specific HTML elements, data attributes, etc.

    workspaces := []map[string]interface{}{}

    // Example: find elements with specific class
    var traverse func(*html.Node)
    traverse = func(n *html.Node) {
        if n.Type == html.ElementNode && n.Data == "div" {
            for _, attr := range n.Attr {
                if attr.Key == "class" && strings.Contains(attr.Val, "workspace-item") {
                    workspace := extractWorkspaceData(n)
                    workspaces = append(workspaces, workspace)
                }
            }
        }

        for c := n.FirstChild; c != nil; c = c.NextSibling {
            traverse(c)
        }
    }

    traverse(doc)
    return workspaces, nil
}
```

### Step 5: Define Data Structures

Edit `internal/interfaces/types.go`:

```go
package interfaces

// WorkspaceData represents a workspace
type WorkspaceData struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Owner       string    `json:"owner"`
    Created     string    `json:"created"`
    Updated     string    `json:"updated"`
}

// DocumentData represents a document
type DocumentData struct {
    ID       string `json:"id"`
    Title    string `json:"title"`
    Content  string `json:"content"`
    Author   string `json:"author"`
    Created  string `json:"created"`
    Updated  string `json:"updated"`
}
```

### Step 6: Update API Handler

Edit `internal/handlers/api.go`:

```go
func (h *APIHandlers) storeExtensionData(payload ExtensionDataPayload) error {
    pageType := payload.Data["pageType"].(string)
    htmlContent := payload.Data["html"].(string)

    // Use your custom parser
    parser := NewYourAppParser()
    results, err := parser.ParseHTML(htmlContent, pageType, payload.URL)
    if err != nil {
        return fmt.Errorf("failed to parse HTML: %w", err)
    }

    // Store based on page type
    switch pageType {
    case "workspaceList":
        return h.storeWorkspaces(results, payload.Timestamp)
    case "document":
        return h.storeDocuments(results, payload.Timestamp)
    default:
        return nil
    }
}
```

### Step 7: Update Storage Layer

Add new storage methods in `internal/services/storage.go`:

```go
func (s *storage) SaveWorkspaces(workspaces []*WorkspaceData) error {
    return s.db.Update(func(tx *bolt.Tx) error {
        bucket := tx.Bucket([]byte("workspaces"))

        for _, workspace := range workspaces {
            data, err := json.Marshal(workspace)
            if err != nil {
                return err
            }
            if err := bucket.Put([]byte(workspace.ID), data); err != nil {
                return err
            }
        }

        return nil
    })
}
```

### Step 8: Update UI

Edit `cmd/aktis-chrome-extension/sidepanel.html` and `sidepanel.js` to reflect your data types:

```html
<div class="workspace-list">
  <!-- Your custom UI for displaying workspaces -->
</div>
```

```javascript
function displayBufferData(data) {
  // Custom rendering for your data types
  for (const workspace in data.workspaces) {
    // Render workspace item
  }
}
```

## Testing

### Extension Testing

1. **Load unpacked extension** in Chrome
2. **Open DevTools** (F12) on extension pages
3. **Monitor Console** for errors
4. **Use debugger**:
   ```javascript
   debugger; // Add breakpoints in code
   ```

### Server Testing

1. **Unit tests** for parser:
```go
func TestParseWorkspaces(t *testing.T) {
    parser := NewYourAppParser()

    html := `<div class="workspace-item">...</div>`
    results, err := parser.ParseHTML(html, "workspaceList", "https://test.com")

    assert.NoError(t, err)
    assert.Equal(t, 1, len(results))
}
```

2. **Integration tests** for API:
```go
func TestReceiverEndpoint(t *testing.T) {
    payload := ExtensionDataPayload{
        URL: "https://test.com",
        Data: map[string]interface{}{
            "pageType": "workspaceList",
            "html": "<html>...</html>",
        },
    }

    response := testServer.POST("/receiver", payload)

    assert.Equal(t, 200, response.StatusCode)
}
```

3. **Manual testing** with curl:
```bash
curl -X POST http://localhost:8084/receiver \
  -H "Content-Type: application/json" \
  -d @test-payload.json
```

## Build & Package

### Development Build

```bash
# Build server
cd internal
go build -o ../bin/aktis-collector-jira.exe ./cmd/aktis-collector-jira

# No build needed for extension (load unpacked)
```

### Production Build

```bash
# Use build script
.\scripts\build.ps1

# Output:
# bin/aktis-collector-jira.exe
# extension packaged as .zip
```

### Extension Packaging

```bash
# Create zip for Chrome Web Store
cd cmd/aktis-chrome-extension
zip -r ../../dist/extension.zip . -x "*.git*" "*.DS_Store"
```

## Debugging Tips

### Extension Debugging

1. **Background script logs**:
   - Go to `chrome://extensions/`
   - Click "Inspect views: service worker"
   - View console logs

2. **Content script logs**:
   - Open DevTools (F12) on Jira page
   - Console shows content script output

3. **Side panel logs**:
   - Right-click side panel
   - "Inspect"
   - View console

### Server Debugging

1. **Enable debug logging**:
```toml
[logging]
level = "debug"
```

2. **View logs**:
```bash
tail -f logs/aktis-collector-jira.*.log
```

3. **Use debugger**:
```go
import "github.com/davecgh/go-spew/spew"

spew.Dump(parsedData) // Pretty print data structures
```

## Performance Optimization

### Extension Performance

1. **Minimize HTML size**:
   - Remove scripts/styles before sending
   - Only send visible content

2. **Debounce auto-collection**:
   - Wait for page to fully load
   - Use `document.readyState === 'complete'`

3. **Batch operations**:
   - Collect multiple items before sending

### Server Performance

1. **Parser optimization**:
   - Use efficient HTML traversal
   - Compile regex once, reuse

2. **Database optimization**:
   - Batch writes in transactions
   - Use indexes for lookups

3. **Memory management**:
   - Stream large HTML documents
   - Clear parsed DOM after extraction

## Security Best Practices

1. **Extension permissions**:
   - Request minimum necessary permissions
   - Use `activeTab` instead of `<all_urls>`

2. **Input validation**:
   - Sanitize HTML before parsing
   - Validate all user inputs

3. **CORS configuration**:
   - Restrict origins in production
   - Use specific origins, not `*`

4. **Data privacy**:
   - Don't log sensitive data
   - Encrypt database at rest (optional)

## Common Pitfalls

1. **Content Security Policy (CSP)**:
   - Can't use inline scripts in extension
   - Use external JS files

2. **Async/Await in extension**:
   - Remember to return `true` in message listeners
   - Keeps channel open for async response

3. **BoltDB file locking**:
   - Only one process can access DB
   - Close DB on shutdown

4. **HTML structure changes**:
   - Target sites update frequently
   - Make parser resilient with fallbacks

## Resources

- **Chrome Extension Docs**: https://developer.chrome.com/docs/extensions/
- **Manifest V3 Migration**: https://developer.chrome.com/docs/extensions/mv3/intro/
- **BoltDB Documentation**: https://github.com/etcd-io/bbolt
- **Go HTML Parsing**: https://pkg.go.dev/golang.org/x/net/html
- **Regex Testing**: https://regex101.com/

## Contributing

When contributing to this template:

1. Maintain clean separation of concerns
2. Add tests for new parsers
3. Update documentation
4. Follow Go and JavaScript style guides
5. Use conventional commits (feat:, fix:, docs:, etc.)