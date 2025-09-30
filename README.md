# Aktis Collector - Jira

A production-ready Jira ticket collector built with the Aktis Plugin SDK, featuring Chrome extension-based data collection and an integrated web dashboard.

## 🎯 System Architecture

This system implements a complete Jira ticket collection and analytics solution with Chrome extension-based data capture and an integrated web interface.

### **Chrome Extension Collection**
- **Purpose**: Browser-based data collection as you browse Jira
- **Technology**: Chrome Extension (Manifest V3) with content scripts
- **Architecture**: Extension sends data to local server via POST to /receiver
- **Key Features**:
  - Automatic data collection from Jira pages (issues, boards, search results)
  - Manual collection via extension popup
  - Auto-collect on page load (configurable)
  - Support for multiple Jira page types
  - Future: Follow links to chase project items
  - No API credentials required (uses your existing browser session)

### **Server Mode** (Web Interface + Data Receiver)
- **Purpose**: Integrated web server with dashboard and extension data receiver
- **Technology**: HTMX-based dynamic UI with RESTful API
- **Storage**: BBolt database with automatic backups
- **Key Features**:
  - Receives data from Chrome extension via /receiver endpoint
  - Real-time statistics and metrics dashboard
  - Interactive data visualization
  - Ticket filtering and search capabilities
  - Project overview and analytics
  - Responsive design optimized for all devices
  - Structured logging with arbor

## 🏗️ Project Structure

```
aktis-collector-jira/
├── cmd/
│   ├── aktis-collector-jira/    # Main application entry point
│   │   └── main.go               # Server mode: Config → Logger → Banner
│   └── aktis-chrome-extension/  # Chrome extension for data collection
│       ├── manifest.json         # Extension manifest (Manifest V3)
│       ├── background.js         # Service worker (data forwarding)
│       ├── content.js            # Content script (page data extraction)
│       ├── popup.html/js         # Extension UI
│       ├── icons/                # Extension icons (16x16, 48x48, 128x128)
│       └── README.md             # Extension documentation
├── internal/
│   ├── common/                   # Infrastructure layer (aktis-receiver template)
│   │   ├── banner.go             # Startup banner display
│   │   ├── config.go             # Configuration management (TOML)
│   │   ├── errors.go             # Structured error handling
│   │   ├── logging.go            # Arbor logger integration
│   │   └── version.go            # Version management
│   ├── interfaces/               # Service interfaces
│   │   └── interfaces.go         # Interface definitions
│   ├── services/                 # Service implementations
│   │   ├── collector.go          # Main collection orchestration
│   │   ├── jira_client.go        # Jira API integration
│   │   ├── storage.go            # BBolt database persistence
│   │   └── webserver.go          # Integrated web server
│   └── handlers/                 # HTTP handlers
│       ├── api/                  # API handlers
│       └── ui/                   # UI handlers
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
├── web-interface/                # Legacy dashboard (deprecated)
│   ├── index.html                # Legacy dashboard interface
│   ├── app.js                    # Legacy dashboard logic
│   └── server.go                 # Legacy API server
├── .github/workflows/
│   └── ci-cd.yml                 # GitHub Actions pipeline
├── go.mod                        # Go module definition
├── .version                      # Auto-increment version tracking
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
web_port = 8080   # Port for web interface in server mode

[jira]
base_url = "https://your-company.atlassian.net"
username = "your-email@company.com"
api_token = "your-jira-api-token"
timeout_seconds = 30

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
backup_dir = "./backups"
retention_days = 90
```

**Configuration Priority:** Defaults → TOML file → Environment variables → Command line flags

#### Get Your Jira API Token
1. Go to [https://id.atlassian.com/manage-profile/security/api-tokens](https://id.atlassian.com/manage-profile/security/api-tokens)
2. Click "Create API token"
3. Copy the generated token to your config file

### Running the Application

**Server Mode (Required for Extension):**
```bash
./bin/aktis-collector-jira -config deployments/config.toml
```

The application now runs as a server by default, listening for data from the Chrome extension.

**Command Line Options:**
- `-version`: Show version information
- `-help`: Show help message
- `-config <path>`: Configuration file path (default: `./config.toml`)
- `-mode <env>`: Environment mode: dev/development/prod/production (default: dev)
- `-validate`: Validate configuration file and exit

**Server Examples:**
```bash
# Start server on default port (8080)
./bin/aktis-collector-jira -config deployments/config.toml

# Start server in production mode
./bin/aktis-collector-jira -config deployments/config.toml -mode prod
```

### Chrome Extension Setup

**1. Create Extension Icons (One-time Setup):**
```bash
# Run the icon creation script
powershell.exe -ExecutionPolicy Bypass -File cmd/aktis-chrome-extension/create-icons.ps1
```

This creates the required icon files (16x16, 48x48, 128x128) in the `cmd/aktis-chrome-extension/icons/` directory.

**2. Install Extension in Chrome:**

1. Start the server (required for extension to work):
   ```bash
   ./bin/aktis-collector-jira -config deployments/config.toml
   ```

2. Open Chrome and navigate to `chrome://extensions/`

3. Enable "Developer mode" (toggle in top right corner)

4. Click "Load unpacked"

5. Select the directory: `cmd/aktis-chrome-extension`

6. The Aktis Jira Collector extension icon should appear in your toolbar

**3. Configure Extension:**

1. Click the extension icon in Chrome toolbar

2. Configure settings:
   - **Server URL**: `http://localhost:8080` (or your server address)
   - **Auto-collect**: Enable to automatically collect data when Jira pages load
   - **Follow links**: (Future feature) Enable to automatically follow linked items

3. Click "Save Settings"

**4. Collect Data:**

**Manual Collection:**
- Navigate to any Jira page (issue, board, search results)
- Click the extension icon
- Click "Collect Current Page"
- Data is sent to the server and stored in the database

**Automatic Collection:**
- Enable "Auto-collect" in extension settings
- Data is automatically collected whenever you visit a Jira page
- Activity is logged in the web dashboard

**Supported Jira Pages:**
- Issue pages: `/browse/PROJECT-123`
- Board/Backlog: `/board/`, `/secure/RapidBoard`
- Search results: `/issues/`
- Project pages: `/projects/`

### Web Interface

**Access the Dashboard:**
Open http://localhost:8080 in your browser (or the port configured in `web_port`)

**Features:**
- **Collection Tab**: View extension activity log and installation instructions
- **Overview Tab**: Real-time statistics and metrics dashboard
- **Storage Tab**: View database contents and manage stored data
- **Config Tab**: System configuration display
- HTMX-based dynamic UI with real-time updates
- Interactive data visualization and analytics
- Responsive design for desktop and mobile

**API Endpoints:**
- `POST /receiver` - Receives data from Chrome extension
- `GET /health` - System health check
- `GET /status` - Collector status and metrics
- `GET /config` - System configuration
- `GET /database` - Database contents
- `DELETE /database` - Clear database

## 📊 Key Features

### Chrome Extension Features
✅ **Browser-Based Collection**: Collect data as you browse Jira (no API tokens needed)
✅ **Automatic Collection**: Optional auto-collect on page load
✅ **Manual Collection**: Click to collect current page on demand
✅ **Multi-Page Support**: Issues, boards, search results, project pages
✅ **Structured Data Extraction**: Parses issue fields, status, priority, labels, etc.
✅ **Configurable Server URL**: Point to any Aktis Collector server
✅ **Real-time Feedback**: In-page notifications on collection success/failure
✅ **Future-Ready**: Framework for link following and project chasing

### Server Features
✅ **Extension Data Receiver**: POST /receiver endpoint accepts data from extension
✅ **BBolt Database**: Embedded database with automatic backups and transactions
✅ **Multi-Project Support**: Automatically organizes data by project
✅ **Data Retention**: Configurable cleanup of old data
✅ **Structured Logging**: Arbor logger with file and console output
✅ **Version Management**: Auto-increment build versioning with timestamps
✅ **Error Handling**: Comprehensive error handling and logging
✅ **CORS Support**: Cross-origin requests enabled for extension communication

### Web Interface Features
✅ **Integrated Server**: Built-in web server with single-binary deployment
✅ **HTMX-Based UI**: Modern dynamic interface without complex JavaScript frameworks
✅ **Collection Activity Log**: Real-time view of extension data collection
✅ **Interactive Visualizations**: Charts and graphs for data analysis
✅ **Project Analytics**: Per-project statistics and trending
✅ **Database Management**: View and clear stored data
✅ **Responsive Design**: Optimized for desktop, tablet, and mobile devices
✅ **RESTful API**: Clean API endpoints for external integrations

## 📈 Data Flow

### Chrome Extension Collection Flow
```
User browses Jira in Chrome
     |
     | [Jira page load]
     v
Chrome Extension Content Script (content.js)
     |
     | [DOM scraping + data extraction]
     v
Extension Background Worker (background.js)
     |
     | [POST JSON to /receiver endpoint]
     v
Server Receiver Handler (internal/handlers/api.go)
     |
     | [Data validation + parsing]
     v
Storage Layer (internal/services/storage.go)
     |
     | [BBolt Database transactions]
     v
Local Database (./data/aktis-collector-jira.db)
```

### Web Dashboard Flow
```
BBolt Database (./data/aktis-collector-jira.db)
     |
     | [Read stored ticket data]
     v
Web Server (internal/services/webserver.go)
     |
     | [HTTP API + HTMX responses]
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
├── aktis-collector-jira.db.20250926_103000.bak
├── aktis-collector-jira.db.20250926_103015.bak
└── ...
```

### BBolt Database Organization
- **Buckets**: Each project gets its own bucket (e.g., "dev", "proj")
- **Keys**: Ticket keys (e.g., "DEV-123", "PROJ-456")
- **Values**: JSON-serialized ticket data
- **Transactions**: ACID compliance for data integrity
- **Backup**: Automatic periodic backups with configurable retention

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
    github.com/ternarybob/aktis-plugin-sdk v0.1.2  // Aktis Plugin SDK
    github.com/go-resty/resty/v2 v2.16.2           // HTTP client for Jira API
    github.com/ternarybob/arbor v1.4.44            // Structured logging
    github.com/ternarybob/banner v0.0.4            // Startup banners
)
```

### Adding Features

**To the Chrome Extension:**
1. **New Page Types**: Add detection logic in `detectPageType()` in `cmd/aktis-chrome-extension/content.js`
2. **Additional Data Fields**: Extend extraction functions (`extractIssueData()`, `extractBoardData()`, etc.)
3. **New Selectors**: Add selector arrays for different Jira versions
4. **Custom Processing**: Modify `collectCurrentPage()` to handle new data structures

**To the Server:**
1. **Enhanced Data Storage**: Extend `storeExtensionData()` in `internal/handlers/api.go`
2. **New API Endpoints**: Add handlers in `internal/handlers/` directory
3. **Additional Processing**: Extend receiver logic to handle new data types
4. **Storage Extensions**: Extend storage interface in `internal/interfaces/interfaces.go`

**To the Web Interface:**
1. **New UI Components**: Extend HTMX templates in `pages/index.html`
2. **Additional Visualizations**: Add chart functions in JavaScript section
3. **Custom Analytics**: Add calculation functions in `internal/services/webserver.go`
4. **Activity Log Enhancements**: Add real-time updates via polling or SSE

## 🚀 Production Deployment

### Deployment Checklist
- [ ] Build and deploy server binary
- [ ] Configure production server URL
- [ ] Set up HTTPS for web interface (if exposed externally)
- [ ] Configure firewall rules (allow port 8080 or configured port)
- [ ] Set up log rotation
- [ ] Configure data retention policies
- [ ] Set up monitoring and alerting
- [ ] Configure backup strategies for BBolt database
- [ ] Install Chrome extension on user machines
- [ ] Configure extension with production server URL
- [ ] Train users on manual vs auto-collect modes

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