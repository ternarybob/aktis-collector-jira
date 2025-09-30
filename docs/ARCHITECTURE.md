# Aktis Collector Jira - Architecture Documentation

## Overview

Aktis Collector Jira is a Chrome extension-based web scraping system designed to collect Jira project and ticket data. This application serves as a template for future collectors that will scrape other types of information from various web applications.

## System Architecture

### Components

```
┌─────────────────────────────────────────────────────────────────┐
│                      Chrome Extension                            │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────────┐   │
│  │  Content     │  │  Background  │  │  Side Panel UI      │   │
│  │  Script      │  │  Service     │  │  (User Interface)   │   │
│  └──────────────┘  └──────────────┘  └─────────────────────┘   │
└────────────────┬────────────────────────────────────────────────┘
                 │
                 │ HTTP/REST API
                 │
┌────────────────▼────────────────────────────────────────────────┐
│                    Go HTTP Server                                │
│  ┌──────────────────┐  ┌─────────────────┐  ┌───────────────┐  │
│  │  API Handlers    │  │  HTML Parser    │  │  Storage      │  │
│  │  (/receiver)     │  │  (Jira Parser)  │  │  Service      │  │
│  └──────────────────┘  └─────────────────┘  └───────────────┘  │
└─────────────────────────────────┬───────────────────────────────┘
                                  │
                                  │
                          ┌───────▼────────┐
                          │  BoltDB        │
                          │  (Local Store) │
                          └────────────────┘
```

### Technology Stack

- **Frontend**: Chrome Extension (Manifest V3)
  - Background Service Worker (background.js)
  - Content Script (content.js)
  - Side Panel UI (sidepanel.html/js)

- **Backend**: Go HTTP Server
  - HTTP Server (port 8084)
  - REST API handlers
  - HTML parsing engine
  - BoltDB storage layer

- **Storage**: BoltDB (embedded key-value database)
  - Projects bucket
  - Tickets bucket (per project)

## Data Flow

### Phase 1: Initial Setup & Project Discovery

```
1. User configures extension
   ↓
2. User navigates to Jira projects page
   ↓
3. Extension detects page type (projectsList)
   ↓
4. User clicks "Collect Current Page"
   ↓
5. Extension sends HTML to server
   ↓
6. Server parses HTML and extracts projects
   ↓
7. Server stores projects in database
   ↓
8. Server returns project list to extension
```

### Phase 2: Ticket Collection

```
1. User marks projects for collection
   ↓
2. User clicks "Collect" button
   ↓
3. Extension navigates through project pages
   ↓
4. For each page:
   a. Extract all visible tickets/items
   b. Send raw HTML to server
   c. Server parses HTML
   d. Server extracts ticket details
   e. Server stores in database
   ↓
5. Repeat until all tickets collected
```

### Phase 3: Refresh & Updates

```
1. User requests refresh (full or incremental)
   ↓
2. Extension re-navigates project pages
   ↓
3. Server compares with existing data
   ↓
4. Server updates changed tickets
   ↓
5. Database reflects current state
```

## Process Workflow

### 1. Extension Configuration

**User Action**: Opens extension settings tab

**Extension Behavior**:
- Displays current configuration
- Server URL (default: http://localhost:8084)
- Auto-collect on page load toggle
- Auto-navigate to follow links toggle

**Server Role**: None (client-side only)

### 2. Project Discovery

**User Action**: Navigates to Jira projects page (e.g., `https://company.atlassian.net/jira/projects`)

**Extension Behavior**:
- Content script detects page type as `projectsList`
- Side panel shows "Online" server status
- Side panel shows page type: "projectsList"

**User Action**: Clicks "Collect Current Page" button

**Extension Behavior**:
- Collects full HTML: `document.documentElement.outerHTML`
- Sends POST request to `/receiver` endpoint:

```json
{
  "timestamp": "2025-09-30T14:30:00Z",
  "url": "https://company.atlassian.net/jira/projects",
  "title": "Projects - Jira",
  "data": {
    "pageType": "projectsList",
    "url": "https://company.atlassian.net/jira/projects",
    "title": "Projects - Jira",
    "html": "<html>...</html>"
  },
  "collector": {
    "name": "aktis-jira-collector-extension",
    "version": "0.1.0"
  }
}
```

**Server Behavior**:
1. Receives payload at `/receiver` endpoint
2. Extracts `pageType` from payload
3. Routes to `JiraParser.ParseHTML()` with type `projectsList`
4. Parser extracts project data:
   - Project Key (e.g., "PROJ", "DEV")
   - Project Name
   - Project Type (Software/Service)
   - Project URL
   - Description (if available)
5. Stores projects in BoltDB `projects` bucket
6. Returns success response:

```json
{
  "success": true,
  "message": "Successfully received and stored projectsList page data",
  "timestamp": "2025-09-30T14:30:01Z"
}
```

### 3. Project Selection & Ticket Collection

**User Action**: Views Buffer tab to see collected projects

**Extension Behavior**:
- Fetches `/database/data` endpoint
- Displays projects in expandable list
- Shows project count

**User Action**: Selects projects to collect tickets from

**Extension Behavior** (Planned):
- Allows marking specific projects
- Provides "Collect Selected Projects" button

**User Action**: Clicks collect for a project

**Extension Behavior**:
- Navigates to project issue list page
- Example: `https://company.atlassian.net/jira/software/c/projects/PROJ/issues`
- Detects page type as `issueList`
- Collects HTML and sends to server

**Server Behavior**:
1. Receives HTML with `pageType: issueList`
2. Parser extracts issue keys from HTML:
   - Searches for issue key patterns: `[A-Z]+-\d+`
   - Looks in table rows with `data-issue-key` attributes
   - Searches links: `<a href="/browse/PROJ-123">`
3. For each issue found:
   - Extracts: key, summary, status, type, priority
   - Creates `TicketData` structure
4. Groups tickets by project key
5. Stores in BoltDB under project-specific bucket
6. Returns success with ticket count

### 4. Individual Ticket Detail Collection

**Extension Behavior** (For detailed data):
- Navigates to individual ticket pages
- Example: `https://company.atlassian.net/browse/PROJ-123`
- Detects page type as `issue`
- Sends HTML to server

**Server Behavior**:
1. Receives HTML with `pageType: issue`
2. Parser extracts detailed fields:
   - Summary
   - Description (full text)
   - Issue Type
   - Status
   - Priority
   - Reporter
   - Assignee
   - Created/Updated dates
   - Comments
   - Custom fields (via data-testid attributes)
3. Merges with existing ticket or creates new
4. Updates database

### 5. Data Storage

**BoltDB Structure**:

```
aktis-collector-jira.db
├── projects (bucket)
│   └── [project_key] → ProjectData JSON
│       ├── key: "PROJ"
│       ├── name: "Product Project"
│       ├── type: "Software"
│       ├── url: "https://..."
│       └── updated: "2025-09-30T14:30:00Z"
│
└── tickets (bucket)
    └── [project_key]_tickets (nested bucket)
        └── [ticket_key] → TicketData JSON
            ├── key: "PROJ-123"
            ├── summary: "Fix login bug"
            ├── description: "Users cannot..."
            ├── issue_type: "Bug"
            ├── status: "In Progress"
            ├── priority: "High"
            └── updated: "2025-09-30T14:30:00Z"
```

### 6. Refresh Operations

**Full Refresh**:
- Re-collect all project pages
- Re-collect all ticket lists
- Update all existing tickets
- Add new tickets

**Incremental Refresh**:
- Collect only modified tickets
- Use "updated" timestamp to filter
- Merge changes with existing data

**User Action**: Clicks "Refresh" button

**Extension Behavior**:
- Re-navigates previously collected projects
- Sends HTML to server for each page

**Server Behavior**:
- Compares incoming tickets with stored tickets
- Updates only changed records
- Adds new tickets
- Maintains historical data

## API Endpoints

### `/receiver` (POST)
**Purpose**: Receive and process HTML from extension

**Request**:
```json
{
  "timestamp": "ISO-8601",
  "url": "string",
  "title": "string",
  "data": {
    "pageType": "projectsList | issue | issueList | board | search | generic",
    "html": "string"
  },
  "collector": {
    "name": "string",
    "version": "string"
  }
}
```

**Response**:
```json
{
  "success": true,
  "message": "string",
  "timestamp": "ISO-8601"
}
```

### `/health` (GET)
**Purpose**: Check server status

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "ISO-8601",
  "version": "string",
  "build": "string",
  "uptime_seconds": 123.45,
  "services": {
    "database": true,
    "jira": true
  }
}
```

### `/status` (GET)
**Purpose**: Get collector statistics

**Response**:
```json
{
  "collector": {
    "running": true,
    "uptime": 123.45,
    "error_count": 0
  },
  "projects": [],
  "stats": {
    "total_tickets": 0,
    "last_collection": "Never",
    "database_size": "N/A"
  }
}
```

### `/database/data` (GET)
**Purpose**: Retrieve all stored tickets grouped by project

**Response**:
```json
{
  "PROJ": {
    "PROJ-1": {
      "key": "PROJ-1",
      "summary": "...",
      "updated": "..."
    }
  }
}
```

### `/database` (DELETE)
**Purpose**: Clear all stored tickets

**Response**:
```json
{
  "success": true,
  "message": "All tickets cleared from database",
  "count": 0
}
```

## Page Type Detection (Enhanced October 2025)

### Multi-Strategy Detection

The system now uses **content-based detection** in addition to URL patterns, making it more robust for modern Jira Cloud's SPA architecture.

**Detection Priority**:
1. **Content Analysis**: Counts issue links, issue rows, project links in HTML
2. **URL Pattern Matching**: Traditional URL-based patterns
3. **Combined Classification**: Merges both strategies with confidence scoring

| Page Type | Detection Methods | Description |
|-----------|-------------------|-------------|
| `projectsList` | URL: `/jira/projects?page=*`<br>Content: 3+ project links | Projects directory page |
| `issue` | URL: `/browse/[KEY-123]`<br>Content: issue detail layout | Individual ticket detail page |
| `issueList` | URL: `/jira/software/c/projects/[KEY]/issues`<br>Content: 3+ issue links or rows | Project ticket list |
| `board` | URL: `/board/*`<br>Content: board layout | Kanban/Scrum board |
| `search` | URL: `/issues/?jql=*`<br>Content: 3+ issue links | Search results page |
| `generic` | URL: Jira domain<br>Content: insufficient indicators | Fallback extraction |

**Key Enhancement**: Detection no longer relies solely on URL. Modern Jira Cloud uses virtual scrolling and SPA navigation with inconsistent URLs, so content analysis is now prioritized.

## HTML Parsing Strategy (Enhanced October 2025)

### Issue List Page - Multi-Strategy Parsing

Modern Jira Cloud uses different HTML structures than old Jira Server. The parser now uses **4 strategies** to find issue rows:

**Strategy 1: Explicit Attributes (Old Jira Server)**
```html
<tr data-issue-key="DATA-35">...</tr>
```

**Strategy 2: Test ID Patterns (New Jira Cloud)**
```html
<div data-testid="issue.container.123">...</div>
<div data-testid="issue-row-456">...</div>
```

**Strategy 3: Link-Based Detection (Modern Jira Cloud)**
- Finds `<div>`, `<li>`, `<article>` elements containing `/browse/` links
- Validates element has 2+ child elements (not just wrapper)
- Depth-limited to avoid false positives
- **This is the key enhancement** for modern Jira Cloud support

**Strategy 4: Semantic Attributes**
```html
<div role="row" class="issue-row">DATA-35</div>
```

### Issue Key Extraction - 3 Fallback Strategies

1. **Data Attributes**: `data-issue-key` or other `data-*` attributes
2. **Links**: `<a href="/browse/DATA-35">` patterns
3. **Text Content**: Regex `\b([A-Z]+-\d+)\b` on all text

### Projects List Page
1. Find all `<tr>` table rows
2. Look for cells containing project keys (2-10 uppercase chars)
3. Extract adjacent cells for name, type, URL
4. Store as `ProjectData`

### Issue Detail Page
1. Search for elements with `data-testid` attributes:
   - `*summary*` → summary field
   - `*description*` → description field
   - `*issue-type*` → issue type
   - `*status*` → status
   - `*priority*` → priority
2. Extract text content from matched elements
3. Parse structured data into `TicketData`

### Accumulative Collection

**List Pages** provide breadth (many tickets, basic fields):
- Key, project_id, summary, status, priority, type, assignee

**Detail Pages** provide depth (one ticket, all fields):
- All list fields PLUS: description, comments, subtasks, attachments, links, work logs

**Data accumulates**: Visiting list page first, then detail pages builds complete dataset.

## Extension Template Pattern

This implementation serves as a template for future collectors:

### Reusable Components
1. **Extension Structure**:
   - Background service worker
   - Content script injection
   - Side panel UI
   - Settings management

2. **Server Architecture**:
   - HTTP server with CORS support
   - `/receiver` endpoint for data ingestion
   - HTML parsing pipeline
   - BoltDB storage layer

3. **Data Flow Pattern**:
   - User navigates to target page
   - Extension detects page type
   - User triggers collection
   - Extension sends raw HTML
   - Server parses and stores data
   - User views buffer and manages data

### Customization Points

To adapt this template for other websites:

1. **Page Type Detection** (`content.js`):
   - Update `detectPageType()` with new URL patterns
   - Add page-specific detection logic

2. **HTML Parser** (`jira_parser.go`):
   - Create new parser struct
   - Implement page-specific extraction logic
   - Define target data structures

3. **Data Models** (`interfaces/types.go`):
   - Define new data structures
   - Add database buckets for new entity types

4. **UI Updates** (`sidepanel.html/js`):
   - Update page type display
   - Customize buffer visualization
   - Add domain-specific controls

## Security Considerations

1. **Extension Permissions**:
   - `activeTab`: Access current tab only when user clicks
   - `storage`: Store configuration
   - `scripting`: Execute scripts to collect HTML
   - `sidePanel`: Display side panel UI

2. **Host Permissions**:
   - Limited to `*.atlassian.net` and `*.jira.com`
   - Localhost access for server communication

3. **Data Privacy**:
   - All data stored locally in BoltDB
   - No external transmission except to local server
   - User controls all collection actions

4. **CORS Configuration**:
   - Server allows all origins (`*`) for local development
   - Should be restricted for production deployments

## Future Enhancements

1. **Automated Navigation**:
   - Follow pagination links automatically
   - Crawl all project pages sequentially
   - Rate limiting to avoid detection

2. **Incremental Updates**:
   - Track collection timestamps
   - Only collect modified tickets
   - Differential updates

3. **Export Functionality**:
   - Export to JSON
   - Export to CSV
   - Send to external systems (aktis-receiver)

4. **Progress Tracking**:
   - Show collection progress
   - Estimate time remaining
   - Pause/resume capability

5. **Error Handling**:
   - Retry failed collections
   - Log parsing errors
   - Alert on missing data

## Troubleshooting

### "Receiving end does not exist" Error
**Cause**: Content script not loaded on current page
**Solution**: Extension now includes fallback to direct collection

### Server Connection Failed
**Cause**: Server not running or wrong port
**Fix**:
1. Check server is running: `netstat -ano | findstr "8084"`
2. Verify extension config matches server port
3. Check CORS headers in server response

### No Data Collected
**Cause**: HTML structure doesn't match parser expectations
**Solution**:
1. Check console logs for parsing errors
2. Review HTML structure in browser DevTools
3. Update parser regex/selectors in `jira_parser.go`

### Database Locked
**Cause**: Multiple processes accessing BoltDB
**Solution**: Ensure only one server instance running