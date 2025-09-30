# Aktis Collector - Jira

A production-ready Jira ticket collector built with the Aktis Plugin SDK, featuring multiple collection methods (API and browser-based scraping) and an integrated web dashboard.

## 🎯 System Architecture

This system implements a complete Jira ticket collection and analytics solution with flexible data collection methods and an integrated web interface for monitoring and analytics.

### **Collection Methods**

**API Collection** (Primary)
- **Purpose**: Direct REST API access to Jira
- **Requirements**: Jira username and API token
- **Key Features**:
  - Fast, reliable batch processing
  - JQL query support for filtering
  - Incremental updates based on timestamps
  - Configurable batch sizes and rate limiting
  - Full ticket history support

**Browser-Based Scraping** (Alternative)
- **Purpose**: Scrape data from Jira web pages when API access is limited
- **Technology**: Chrome/Chromium automation with remote debugging
- **Key Features**:
  - Works with OAuth2/SSO authentication
  - Uses existing browser session (no credentials stored)
  - Supports headless or visible browser modes
  - Configurable wait times and page parsing
  - Useful when API access is restricted

**Chrome Extension** (Supplemental)
- **Purpose**: Manual or automatic page data capture as you browse
- **Technology**: Chrome Extension (Manifest V3) with content scripts and side panel
- **Key Features**:
  - Side panel interface for in-browser monitoring
  - Manual collection via popup
  - Auto-collect on page load (configurable)
  - Sends data to local server via POST to /receiver
  - No API credentials required

### **Server Mode** (Web Interface + Data Storage)
- **Purpose**: Integrated web server with dashboard and data management
- **Technology**: HTMX-based dynamic UI with RESTful API
- **Storage**: BBolt embedded database with automatic backups
- **Key Features**:
  - Real-time statistics and metrics dashboard
  - Collection activity monitoring
  - Database viewer and management
  - Configuration display
  - RESTful API endpoints
  - Responsive design optimized for all devices
  - Structured logging with arbor

## 🏗️ Project Structure

```
aktis-collector-jira/
├── cmd/
│   ├── aktis-collector-jira/    # Main application entry point
│   │   └── main.go               # Server mode: Config → Logger → Banner → Start
│   └── aktis-chrome-extension/  # Chrome extension for supplemental data collection
│       ├── manifest.json         # Extension manifest (Manifest V3)
│       ├── background.js         # Service worker (data forwarding)
│       ├── content.js            # Content script (page data extraction)
│       ├── popup.html/js         # Extension popup UI
│       ├── sidepanel.html/js     # Side panel interface
│       ├── icons/                # Extension icons (16x16, 48x48, 128x128)
│       ├── create-icons.ps1      # PowerShell script to generate icons
│       └── README.md             # Extension documentation
├── internal/
│   ├── common/                   # Infrastructure layer (aktis-receiver template)
│   │   ├── banner.go             # Startup banner display
│   │   ├── config.go             # Configuration management (TOML)
│   │   ├── errors.go             # Structured error handling
│   │   ├── logging.go            # Arbor logger integration
│   │   └── version.go            # Version management
│   ├── interfaces/               # Service interfaces
│   │   └── jira.go               # Jira client and service interfaces
│   ├── services/                 # Service implementations
│   │   ├── page_assessor.go      # Jira page analysis and validation
│   │   ├── storage.go            # BBolt database persistence
│   │   └── webserver.go          # Integrated web server
│   └── handlers/                 # HTTP handlers
│       ├── api.go                # API endpoint handlers
│       ├── ui.go                 # UI/HTMX handlers
│       ├── jira_parser.go        # Jira data parsing
│       └── jira_parser_details.go # Detailed Jira field extraction
├── pages/                        # Web UI templates
│   └── index.html                # HTMX-based dashboard interface
├── deployments/
│   ├── aktis-collector-jira.toml # Configuration file (TOML format)
│   ├── docker/                   # Docker deployment
│   │   ├── Dockerfile            # Multi-stage build
│   │   ├── docker-compose.yml    # Service definition
│   │   └── .env.example          # Environment variables
│   └── local/                    # Local deployment configs
├── scripts/
│   ├── build.ps1                 # Windows build with versioning
│   └── build.sh                  # Linux/Mac build script
├── data/                         # Runtime data directory
│   └── aktis-collector-jira.db   # BBolt database (created at runtime)
├── backups/                      # Database backups (created at runtime)
├── .github/workflows/
│   └── ci-cd.yml                 # GitHub Actions pipeline
├── go.mod                        # Go module definition
├── .version                      # Auto-increment version tracking
├── README.md                     # This file - complete system documentation
└── CLAUDE.md                     # Developer documentation
```

## 🚀 Getting Started

### Prerequisites
- Go 1.24 or higher
- Jira account with API token access
- Modern web browser for dashboard

### Build from Source

**Using Build Scripts:**
```bash
# Windows
.\scripts\build.ps1

# Linux/Mac
./scripts/build.sh
```

**Manual Build:**
```bash
go mod download
go build -o bin/aktis-collector-jira ./cmd/aktis-collector-jira
```

### Configuration

Create your configuration file based on the template:

```bash
cp deployments/aktis-collector-jira.toml deployments/config.toml
```

Edit `deployments/config.toml`:

```toml
[collector]
name = "aktis-collector-jira"
environment = "development"
send_limit = 100  # Maximum payloads per run (for aktis-collector scheduling)
port = 8080       # Port for web interface in server mode

[jira]
# Collection methods: ["api"] or ["scraper"] or ["api", "scraper"]
# - "api": Direct REST API access (requires username and api_token)
# - "scraper": Browser-based scraping (useful when OAuth2/SSO is required)
# First method in list is used as primary
method = ["api"]

base_url = "https://your-company.atlassian.net"
timeout_seconds = 30

[jira.api]
# API authentication settings (required when method includes "api")
username = "your-email@company.com"
api_token = "your-jira-api-token"

[jira.scraper]
# Scraper-specific settings (only used when method includes "scraper")

# Use an existing authenticated browser session
# Set to true if you need to login via OAuth2/Microsoft Authenticator
# IMPORTANT: When true, you must start Chrome with remote debugging enabled:
#   Windows: chrome.exe --remote-debugging-port=9222
#   Linux: google-chrome --remote-debugging-port=9222
#   macOS: /Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --remote-debugging-port=9222
use_existing_browser = false

# Remote debugging port (default: 9222)
remote_debug_port = 9222

# Path to Chrome/Edge executable (only used when use_existing_browser = false)
browser_path = ""

# User data directory for the browser profile
user_data_dir = ""

# Run browser in headless mode (no visible window)
headless = true

# Wait time in milliseconds before starting scrape
wait_before_scrape_ms = 1000

[projects]
# List of project keys to collect from
projects = ["dev", "proj"]

[dev]
name = "Development Project"
issue_types = ["Bug", "Story", "Task", "Epic", "Sub-task"]
statuses = ["To Do", "In Progress", "In Review", "Done", "Closed"]
max_results = 1000
include_history = true

[proj]
name = "Product Project"
issue_types = ["Bug", "Feature", "Improvement", "Task"]
statuses = ["Open", "In Progress", "Resolved", "Closed"]
max_results = 500
include_history = false

[storage]
# BBolt database file location - defaults to {executable_location}/data/{exec_name}.db
database_path = "./data/aktis-collector-jira.db"
# Backup directory for database backups
backup_dir = ""
# Data retention in days (0 = keep forever)
retention_days = 90
```

**Configuration Priority:** Defaults → TOML file → Environment variables → Command line flags

#### Get Your Jira API Token
1. Go to [https://id.atlassian.com/manage-profile/security/api-tokens](https://id.atlassian.com/manage-profile/security/api-tokens)
2. Click "Create API token"
3. Copy the generated token to your config file

### Running the Application

**Server Mode:**
```bash
./bin/aktis-collector-jira -config deployments/config.toml
```

The application runs as a server, providing a web interface for monitoring and management. It supports multiple data collection methods configured in the TOML file.

**Command Line Options:**
- `-version`: Show version information
- `-help`: Show help message
- `-config <path>`: Configuration file path (default: `./config.toml`)
- `-mode <env>`: Environment mode: dev/development/prod/production (default: dev)
- `-quiet`: Suppress banner output
- `-validate`: Validate configuration file and exit

**Examples:**
```bash
# Start server with default settings
./bin/aktis-collector-jira -config deployments/config.toml

# Start in production mode with quiet output
./bin/aktis-collector-jira -config deployments/config.toml -mode prod -quiet

# Validate configuration without starting
./bin/aktis-collector-jira -config deployments/config.toml -validate
```

### Chrome Extension Setup (Optional)

The Chrome extension provides supplemental manual data collection as you browse Jira. This is optional - the main server can collect data via API or browser scraping methods.

**1. Create Extension Icons (One-time Setup):**
```bash
# Run the icon creation script
powershell.exe -ExecutionPolicy Bypass -File cmd/aktis-chrome-extension/create-icons.ps1
```

This creates the required icon files (16x16, 48x48, 128x128) in the [cmd/aktis-chrome-extension/icons](cmd/aktis-chrome-extension/icons) directory.

**2. Install Extension in Chrome:**

1. Start the server (required for extension to work):
   ```bash
   ./bin/aktis-collector-jira -config deployments/config.toml
   ```

2. Open Chrome and navigate to `chrome://extensions/`

3. Enable "Developer mode" (toggle in top right corner)

4. Click "Load unpacked"

5. Select the directory: [cmd/aktis-chrome-extension](cmd/aktis-chrome-extension)

6. The Aktis Jira Collector extension icon should appear in your toolbar

**3. Configure Extension:**

1. Click the extension icon in Chrome toolbar to open the popup

2. Configure settings:
   - **Server URL**: `http://localhost:8080` (or your server address)
   - **Auto-collect**: Enable to automatically collect data when Jira pages load

3. Click "Save Settings"

**4. Using the Extension:**

**Side Panel Interface:**
- Click the extension icon and select "Open Side Panel"
- View real-time collection activity
- Monitor server connection status
- See collected ticket statistics

**Manual Collection:**
- Navigate to any Jira page (issue, board, search results)
- Click the extension icon popup
- Click "Collect Current Page"
- Data is sent to the server via POST /receiver

**Automatic Collection:**
- Enable "Auto-collect" in extension settings
- Data is automatically collected on Jira page loads
- Activity is logged in the side panel and web dashboard

**Supported Jira Pages:**
- Issue detail pages: `/browse/PROJECT-123`
- Board/Backlog: `/board/`, `/secure/RapidBoard`
- Search results: `/issues/`
- Project pages: `/projects/`

### Web Interface

**Access the Dashboard:**
Open http://localhost:8080 in your browser (or the port configured in `port` setting)

**Features:**
- **Collection Tab**: View extension activity log and installation instructions
- **Overview Tab**: Real-time statistics and metrics dashboard
- **Storage Tab**: View database contents and manage stored data
- **Config Tab**: System configuration display
- HTMX-based dynamic UI with server-side rendering
- Interactive data visualization and analytics
- Responsive design for desktop and mobile

**API Endpoints:**
- `POST /receiver` - Receives data from Chrome extension or scraper
- `GET /health` - System health check and service status
- `GET /status` - Collector status and metrics
- `GET /config` - System configuration (sanitized)
- `GET /database` - Database contents and statistics
- `DELETE /database` - Clear database (requires confirmation)

## 📊 Key Features

### Collection Method Features

**API Collection:**
✅ **Direct REST API Access**: Fast, reliable batch processing via Jira REST API
✅ **JQL Query Support**: Advanced filtering with Jira Query Language
✅ **Incremental Updates**: Timestamp-based updates for efficiency
✅ **Batch Processing**: Configurable batch sizes and rate limiting
✅ **Full History**: Optional ticket history and changelog collection
✅ **Multi-Project**: Collect from multiple projects in parallel

**Browser Scraping:**
✅ **OAuth2/SSO Support**: Works when direct API access is restricted
✅ **Existing Session**: Leverage authenticated browser sessions
✅ **Headless Mode**: Run without visible browser window
✅ **Remote Debugging**: Connect to existing Chrome instances
✅ **Configurable Wait Times**: Adjust for page load performance

**Chrome Extension:**
✅ **Manual Collection**: Click-to-collect from any Jira page
✅ **Auto-collect**: Optional automatic collection on page load
✅ **Side Panel Interface**: In-browser monitoring and statistics
✅ **Multi-Page Support**: Issues, boards, search results, project pages
✅ **No Credentials**: Uses your existing browser session
✅ **Real-time Feedback**: Immediate collection status notifications

### Server Features
✅ **Multiple Collection Methods**: API, scraper, and extension support
✅ **BBolt Database**: Embedded database with ACID transactions
✅ **Multi-Project Support**: Automatic data organization by project
✅ **Data Retention**: Configurable cleanup policies
✅ **Structured Logging**: Arbor logger with file and console output
✅ **Version Management**: Auto-increment build versioning
✅ **Comprehensive Error Handling**: Detailed error context and recovery
✅ **CORS Support**: Cross-origin requests for extension communication

### Web Interface Features
✅ **Integrated Server**: Built-in web server with single-binary deployment
✅ **HTMX-Based UI**: Dynamic server-side rendering without heavy JavaScript
✅ **Collection Activity Log**: Real-time collection monitoring
✅ **Database Viewer**: Inspect stored tickets and statistics
✅ **Configuration Display**: View sanitized system configuration
✅ **Health Monitoring**: Service status and uptime tracking
✅ **Responsive Design**: Optimized for all device sizes
✅ **RESTful API**: Clean endpoints for external integrations

## 📈 Data Flow

### API Collection Flow
```
Configuration (deployments/config.toml)
     |
     | [Load jira.api settings]
     v
Jira Client (internal/services/jira_client.go) [NOT YET IMPLEMENTED]
     |
     | [REST API calls with JQL queries]
     v
Jira REST API (https://your-company.atlassian.net/rest/api/3)
     |
     | [JSON response batches]
     v
Storage Layer (internal/services/storage.go)
     |
     | [BBolt database transactions]
     v
Local Database (./data/aktis-collector-jira.db)
```

### Browser Scraping Flow
```
Configuration (deployments/config.toml)
     |
     | [Load jira.scraper settings]
     v
Page Scraper (internal/services/scraper.go) [NOT YET IMPLEMENTED]
     |
     | [Chrome DevTools Protocol / Remote debugging]
     v
Chrome Browser (existing session or headless)
     |
     | [Load Jira pages, extract DOM data]
     v
Storage Layer (internal/services/storage.go)
     |
     | [BBolt database transactions]
     v
Local Database (./data/aktis-collector-jira.db)
```

### Chrome Extension Collection Flow
```
User browses Jira in Chrome
     |
     | [Jira page load or manual trigger]
     v
Extension Content Script (cmd/aktis-chrome-extension/content.js)
     |
     | [DOM scraping + data extraction]
     v
Extension Background Worker (cmd/aktis-chrome-extension/background.js)
     |
     | [POST JSON to /receiver endpoint]
     v
Server API Handler (internal/handlers/api.go)
     |
     | [Validation + parsing via jira_parser.go]
     v
Storage Layer (internal/services/storage.go)
     |
     | [BBolt database transactions]
     v
Local Database (./data/aktis-collector-jira.db)
```

### Web Dashboard Flow
```
Local Database (./data/aktis-collector-jira.db)
     |
     | [Read stored ticket data]
     v
Web Server (internal/services/webserver.go)
     |
     | [Route requests to handlers]
     v
UI Handlers (internal/handlers/ui.go)
     |
     | [Generate HTMX responses]
     v
Web Interface (pages/index.html)
     |
     | [Dynamic UI updates via HTMX]
     v
User Browser
```

## 📈 Data Structures

### Extension Data Payload Format
```json
{
  "timestamp": "2025-09-30T10:54:00Z",
  "url": "https://company.atlassian.net/browse/PROJ-123",
  "title": "PROJ-123: Fix login bug",
  "data": {
    "pageType": "issue",
    "html": "...",
    "issue": {
      "key": "PROJ-123",
      "summary": "Fix login bug",
      "description": "Detailed description...",
      "issueType": "Bug",
      "status": "In Progress",
      "priority": "High",
      "assignee": "John Doe",
      "labels": ["backend", "security"],
      "components": ["Auth"]
    }
  },
  "collector": {
    "name": "aktis-jira-collector-extension",
    "version": "0.1.0"
  }
}
```

### Stored Ticket Format
```json
{
  "key": "PROJ-123",
  "summary": "Fix login bug",
  "description": "Detailed description...",
  "type": "Bug",
  "status": "In Progress",
  "priority": "High",
  "source": "extension",
  "updated": "2025-09-30T10:54:00Z"
}
```

### Storage Structure
```
./data/
├── aktis-collector-jira.db    # BBolt database (all project data)
└── ...

./backups/
├── (Backup functionality currently not implemented)
└── ...
```

### BBolt Database Organization
- **Buckets**: Each project gets its own bucket (e.g., "dev", "proj")
- **Keys**: Ticket keys (e.g., "DEV-123", "PROJ-456")
- **Values**: JSON-serialized ticket data
- **Transactions**: ACID compliance for data integrity
- **Embedded**: Single-file database, no external dependencies
- **Concurrent Access**: Safe for multiple readers, single writer

## 🐳 Docker Deployment

Build and run with Docker:

```bash
# Build Docker image
docker build -f deployments/docker/Dockerfile -t aktis-collector-jira .

# Run with docker-compose
cd deployments/docker
docker-compose up -d
```

## 🔧 Development

### Project Standards

This project follows the **aktis-receiver template standards**:

- **Startup Sequence**: Config → Logger → Banner → Info logging
- **Logging**: Uses `github.com/ternarybob/arbor` for all logging
- **Banner**: Uses `github.com/ternarybob/banner` for startup display
- **Configuration**: Hierarchical config system (defaults → env → flags)
- **Architecture**: Clean `/cmd` and `/internal` structure

### Dependencies

```go
require (
    github.com/ternarybob/arbor v1.4.44            // Structured logging
    github.com/ternarybob/banner v0.0.4            // Startup banners
    go.etcd.io/bbolt v1.3.x                        // BBolt embedded database
    // Future: Jira client libraries for API collection
)
```

### Adding Features

**To the Chrome Extension:**
1. **New Page Types**: Add detection logic in `detectPageType()` in [cmd/aktis-chrome-extension/content.js](cmd/aktis-chrome-extension/content.js)
2. **Additional Data Fields**: Extend extraction functions (`extractIssueData()`, `extractBoardData()`, etc.)
3. **New Selectors**: Add selector arrays for different Jira versions
4. **Side Panel Features**: Enhance monitoring in [cmd/aktis-chrome-extension/sidepanel.js](cmd/aktis-chrome-extension/sidepanel.js)

**To the Server:**
1. **API Collection**: Implement Jira API client in `internal/services/jira_client.go` (currently planned)
2. **Browser Scraping**: Implement page scraper in `internal/services/scraper.go` (currently planned)
3. **Enhanced Parsing**: Extend [internal/handlers/jira_parser.go](internal/handlers/jira_parser.go) for new field types
4. **New API Endpoints**: Add handlers in [internal/handlers/api.go](internal/handlers/api.go)
5. **Storage Extensions**: Extend storage interface in [internal/interfaces/jira.go](internal/interfaces/jira.go)

**To the Web Interface:**
1. **New UI Components**: Extend HTMX templates in [pages/index.html](pages/index.html)
2. **Additional Tabs**: Add new sections to the dashboard
3. **Custom Analytics**: Add calculation functions in [internal/handlers/ui.go](internal/handlers/ui.go)
4. **Real-time Updates**: Enhance polling or implement SSE for live data

## 🚀 Production Deployment

### Deployment Checklist
- [ ] Build and deploy server binary using [scripts/build.ps1](scripts/build.ps1) or [scripts/build.sh](scripts/build.sh)
- [ ] Configure production settings in [deployments/aktis-collector-jira.toml](deployments/aktis-collector-jira.toml)
- [ ] Set collection method: API, scraper, or extension-only
- [ ] Set up HTTPS reverse proxy if exposing web interface externally
- [ ] Configure firewall rules (allow configured port, default: 8080)
- [ ] Set up log rotation for application logs
- [ ] Configure data retention policies in storage settings
- [ ] Set up monitoring and alerting for service health
- [ ] Plan backup strategy for BBolt database file
- [ ] (Optional) Install Chrome extension on user machines
- [ ] (Optional) Configure extension with production server URL
- [ ] (Optional) Train users on extension usage

### Chrome Extension Distribution

**For Team Distribution:**

1. **Create Extension Package:**
   - Zip the `cmd/aktis-chrome-extension` directory
   - Distribute to team members

2. **Users Install Locally:**
   - Extract zip file
   - Load unpacked in Chrome (`chrome://extensions/`)
   - Configure server URL to production server

3. **Corporate Distribution (Optional):**
   - Package extension as `.crx` file
   - Distribute via Chrome Enterprise policies
   - Pre-configure server URL via extension policies

**Future: Chrome Web Store Publication:**
- Package extension for Chrome Web Store
- Submit for review
- Publish for public/private installation

## 🔐 Security Considerations

- **Extension Permissions**: Extension requires access to Jira domains and localhost
- **Data Privacy**: Data collected via extension stays local (sent to your server only)
- **Browser Session**: Extension uses your existing Jira browser session (no credentials stored)
- **Server Security**: Dashboard server should be secured with authentication in production
- **CORS Configuration**: Receiver endpoint allows cross-origin requests for extension
- **Backup Security**: Backup files contain sensitive project data
- **Network Security**: Use HTTPS for production server deployments
- **Extension Distribution**: Only install extension from trusted sources

## 📊 Performance

### Extension Optimizations
- **Selective Extraction**: Only collects data from supported page types
- **Efficient DOM Parsing**: Multiple selector strategies minimize search time
- **Async Processing**: Non-blocking collection and transmission
- **Minimal Memory**: Cleans up after each collection
- **Smart Auto-collect**: Only triggers on actual page loads

### Server Optimizations
- **Concurrent Handling**: Handles multiple extension submissions simultaneously
- **Efficient Storage**: BBolt transactions minimize disk I/O
- **Project-based Bucketing**: Fast data organization by project key
- **Logging Control**: Configurable log levels for production

### Dashboard Optimizations
- **HTMX Updates**: Minimal JavaScript for fast page interactions
- **Lazy Loading**: Load data on demand
- **Caching**: Browser caching for static assets
- **Polling Control**: Activity checks only when collection tab is active

## 📖 Documentation

- **README.md** (this file): Complete system documentation
- **cmd/aktis-chrome-extension/README.md**: Chrome extension documentation
- **CLAUDE.md**: Developer documentation with build commands and architecture
- **API Documentation**: See web interface endpoints section above
- **Configuration**: See `deployments/aktis-collector-jira.toml` for all options

## 🎯 Use Cases

### Development Teams
- **Ad-hoc Data Collection**: Collect tickets as you work without scheduled jobs
- **Sprint Planning**: Gather tickets from board/backlog views for analysis
- **Bug Tracking**: Collect bug tickets while triaging or reviewing
- **Context-Aware Collection**: Collect only the data you're actively viewing

### Project Management
- **Meeting Preparation**: Collect specific project tickets before standups/reviews
- **Status Reports**: Gather tickets from search results for reporting
- **Cross-Project Analysis**: Collect tickets from multiple projects as needed
- **Real-time Data**: Always get current data since it uses your live browser session

### Individual Contributors
- **Personal Tracking**: Collect your assigned tickets for time tracking
- **Documentation**: Capture ticket details while writing documentation
- **Knowledge Base**: Build local archive of project tickets
- **Offline Reference**: Store ticket data for offline access

## 📝 License

See LICENSE file for details.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Follow the coding standards (see CLAUDE.md)
5. Submit a pull request

## 🆘 Support

For issues and questions:
- Check the CLAUDE.md documentation
- Review the example configuration
- Open an issue in the repository