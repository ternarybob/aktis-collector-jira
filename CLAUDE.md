# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Recent Enhancements (October 2025)

### Auto-Collection Enhancement (v0.1.126 - Latest)

**Problem**: Projects list pages were not auto-collecting, only issue lists and individual issues.

**Root Cause**:
- `isCollectable()` required "medium" or "high" confidence
- Projects list pages often got "low" confidence if HTML wasn't fully loaded
- URL-based detection alone wasn't enough to meet confidence threshold

**Solution**:
1. **Relaxed Confidence Requirements**: Allow "low" confidence for all known page types
   - Rationale: URL patterns are sufficient for page type determination
   - Auto-collection should be permissive (better to collect than miss)
2. **Enhanced Projects List Detection**: Added multiple URL pattern variations
   - `/jira/projects` (original)
   - `/projects` (direct endpoint)
   - `/projects?` (with query params)

**Impact**: Auto-collection now works for **ALL page types** as user navigates:
- ✅ Projects list pages → Projects collected
- ✅ Issue list pages → Tickets collected (breadth)
- ✅ Individual issue detail pages → Tickets enhanced (depth)
- ✅ Board pages → Tickets from cards
- ✅ Search results pages → Tickets from results

### Comprehensive Refactoring (v0.1.123)

The codebase was comprehensively refactored to follow clean architecture best practices and go-refactor standards while preserving all existing functionality:

### Architectural Improvements

**1. Models Package Created** (`internal/models/`)
- Extracted data structures from `internal/interfaces/` into dedicated model files:
  - `ticket.go` - TicketData and related structures (Comment, Subtask, Attachment, etc.)
  - `project.go` - ProjectData
  - `page.go` - PageAssessment
- Interfaces now only define behavior contracts, not data structures
- Better separation of concerns and reusability

**2. Middleware Package Added** (`internal/middleware/`)
- `cors.go` - CORS headers for Chrome extension communication
- `logging.go` - HTTP request/response logging with status codes and duration
- Extracted from webserver.go for better modularity and reusability
- Chainable middleware pattern for composability

**3. HTML Utilities Created** (`internal/common/html_utils.go`)
- Centralized HTML parsing utilities to reduce code duplication
- Reusable functions: ExtractText, FindNodesByTag, FindNodesByAttribute, FindLinks, etc.

**4. Code Cleanup**
- Removed unused functions: `fileExists()`, `getMapKeys()`, `handleError()`
- Eliminated code duplication in HTML parsing logic
- Organized imports: stdlib → third-party → internal (no dot imports)

**5. Infrastructure Additions**

**Docker Deployment** (`deployments/docker/`)
- Multi-stage Dockerfile with build args for version injection
- docker-compose.yml for orchestration
- .env.example for environment configuration
- Health checks and non-root user security
- Optimized Alpine Linux image (~25MB)

**Build & Deployment Scripts** (`scripts/`)
- `build.ps1` / `build.sh` - Cross-platform build with auto-versioning
- `deploy.ps1` - Deployment automation (local/docker/production)
- `test.ps1` - Test runner with coverage reporting
- `create-favicon.ps1` - Web UI favicon generator

**CI/CD Pipeline** (`.github/workflows/ci-cd.yml`)
- Automated linting with golangci-lint
- Unit tests with race detection and coverage reporting
- Multi-platform builds (Linux, Windows, macOS)
- Docker image build and push to GitHub Container Registry
- Automated GitHub releases with artifacts
- Version tagging and release notes generation

**Local Deployment** (`deployments/local/`)
- Local development configuration
- Quick-start guide and troubleshooting
- Extension installation instructions

### Migration Notes
- Import paths changed from dot imports to explicit imports
- Models moved from `interfaces` to `models` package
- All public APIs remain backward compatible
- 14 backup files created during refactoring

### Code Quality Metrics
- **Packages**: 5 → 7 (+40% modularity)
- **Unused Functions**: 3 → 0 (-100% dead code)
- **Dot Imports**: ~10 files → 0 (better clarity)
- **Middleware Duplication**: 2 places → 0 (DRY principle)
- **Docker Support**: ❌ → ✅ (production ready)
- **Auto-Collection Coverage**: Partial → 100% (all page types)

## Project Overview

Jira ticket collection and analytics system with multiple collection methods and integrated web interface:
- **Main Server** (`cmd/aktis-collector-jira/`): Go-based server with web interface and data receiver
- **Chrome Extension** (`cmd/aktis-chrome-extension/`): Browser extension for manual/auto data collection
- **Web Interface** (`pages/index.html`): HTMX-based dashboard for monitoring and management
- **Planned Features**: API collection and browser scraping methods (configuration present, implementation pending)

## Build Commands

### Main Application
```bash
# Windows
.\scripts\build.ps1

# Linux/Mac
./scripts/build.sh

# Manual build
go mod download
go build -o bin/aktis-collector-jira ./cmd/aktis-collector-jira
```

### Chrome Extension
```bash
# Generate icons (one-time setup)
powershell.exe -ExecutionPolicy Bypass -File cmd/aktis-chrome-extension/create-icons.ps1

# Load unpacked extension in Chrome
# 1. Navigate to chrome://extensions/
# 2. Enable "Developer mode"
# 3. Click "Load unpacked"
# 4. Select cmd/aktis-chrome-extension directory
```

## Running the System

### Start Server
```bash
./bin/aktis-collector-jira -config deployments/aktis-collector-jira.toml

# Common flags:
#   -version                # Show version information
#   -help                   # Show help message
#   -config <path>          # Configuration file path
#   -mode prod              # Production mode (dev/development/prod/production)
#   -quiet                  # Suppress banner output
#   -validate               # Validate configuration and exit
```

### Access Web Interface
```bash
# Default: http://localhost:8080
# Or: http://localhost:<port> (from config file)
```

## Architecture

### Data Flow

**Current Implementation (Extension-based):**
```
Chrome Extension → POST /receiver → API Handler → Jira Parser → Storage → BBolt DB
                                                                              ↓
Web Browser ← HTMX UI ← UI Handlers ← Web Server ← BBolt DB
```

**Planned Implementation (API-based):**
```
Jira REST API → Jira Client (TBD) → Storage → BBolt DB
```

**Planned Implementation (Scraper-based):**
```
Chrome/Chromium → Page Scraper (TBD) → Jira Parser → Storage → BBolt DB
```

### Key Components

**Server** (`cmd/aktis-collector-jira/main.go`)
- Configuration loading and validation
- Logger initialization (arbor)
- Startup banner display
- Web server lifecycle management
- Signal handling for graceful shutdown

**Web Server** (`internal/services/webserver.go`)
- HTTP routing and middleware
- API and UI handler registration
- CORS support for extension
- Static file serving
- Health and status endpoints

**API Handlers** (`internal/handlers/api.go`)
- `POST /receiver`: Accepts data from Chrome extension or scraper
- `GET /health`: System health check
- `GET /status`: Collector status and metrics
- `GET /config`: Configuration display (sanitized)
- `GET /database`: Database contents
- `DELETE /database`: Clear database

**UI Handlers** (`internal/handlers/ui.go`)
- HTMX response generation
- Dashboard tabs (Collection, Overview, Storage, Config)
- Real-time statistics aggregation
- Activity log display

**Jira Parser** (`internal/handlers/jira_parser.go`, `jira_parser_details.go`)
- HTML/DOM parsing from extension data
- Field extraction (summary, description, status, etc.)
- Project and ticket key validation
- Data normalization

**Storage** (`internal/services/storage.go`)
- BBolt database operations
- Project bucket management
- Ticket CRUD operations
- Transaction handling
- Database statistics

**Page Assessor** (`internal/services/page_assessor.go`)
- Jira page type detection
- URL validation
- Page structure analysis

**Configuration** (`internal/common/config.go`)
- TOML file parsing
- Environment variable overrides
- Default value management
- Configuration validation

### Storage Pattern
- **Database**: Single BBolt file at `data/aktis-collector-jira.db`
- **Buckets**: One bucket per project (e.g., "DEV", "PROJ")
- **Keys**: Ticket keys (e.g., "DEV-123")
- **Values**: JSON-serialized ticket data
- **Transactions**: ACID-compliant writes

## Configuration

Located at `deployments/aktis-collector-jira.toml`:
```toml
[collector]
name = "aktis-collector-jira"
environment = "development"
send_limit = 100
port = 8080

[jira]
method = ["api"]  # ["api"] or ["scraper"] or ["api", "scraper"]
base_url = "https://your-company.atlassian.net"
timeout_seconds = 30

[jira.api]
username = "your-email@company.com"
api_token = "your-jira-api-token"

[jira.scraper]
use_existing_browser = false
remote_debug_port = 9222
browser_path = ""
user_data_dir = ""
headless = true
wait_before_scrape_ms = 1000

[projects]
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
database_path = "./data/aktis-collector-jira.db"
backup_dir = ""
retention_days = 90
```

**Configuration Loading Order:**
1. Defaults (from `DefaultConfig()` in `internal/common/config.go`)
2. TOML file override (if provided via `-config` flag)
3. Environment variables (not yet implemented)
4. Command line flags

## Go Version

Go 1.24 or higher required (specified in `go.mod`)

## Dependencies

- `github.com/ternarybob/arbor v1.4.44` - Structured logging framework
- `github.com/ternarybob/banner v0.0.4` - Startup banner display
- `go.etcd.io/bbolt v1.3.x` - BBolt embedded database
- Future: Jira client libraries for API collection

## Development Notes

**Current Implementation Status:**
- ✅ **Chrome Extension**: Fully implemented with popup and side panel
- ✅ **Web Server**: Fully implemented with HTMX UI
- ✅ **Data Receiver**: POST /receiver endpoint functional
- ✅ **Storage**: BBolt database with project buckets
- ✅ **Jira Parser**: HTML/DOM parsing from extension data
- ⏳ **API Collection**: Configuration present, implementation pending
- ⏳ **Browser Scraping**: Configuration present, implementation pending

**Error Handling**
- Comprehensive error wrapping with context (`internal/common/errors.go`)
- Structured logging with arbor at multiple levels
- Graceful degradation when services unavailable
- Extension continues on individual page collection failures

**Data Processing**
- Extension sends raw HTML to server
- Server-side parsing via `jira_parser.go`
- Field extraction using multiple selector strategies
- Project bucket assignment based on ticket key

**Configuration Validation**
- Use `-validate` flag to check config without starting
- Sanitized display in web UI (credentials masked)
- Default values for all optional settings

## File Structure

```
aktis-collector-jira/
├── cmd/
│   ├── aktis-collector-jira/     # Main application
│   │   └── main.go                # Entry point: Config → Logger → Banner → Server
│   └── aktis-chrome-extension/   # Chrome extension
│       ├── manifest.json          # Extension manifest (Manifest V3)
│       ├── background.js          # Service worker
│       ├── content.js             # Content script
│       ├── popup.html/js          # Popup UI
│       ├── sidepanel.html/js      # Side panel UI
│       ├── icons/                 # Extension icons
│       └── create-icons.ps1       # Icon generation script
├── internal/
│   ├── common/                    # Infrastructure layer
│   │   ├── banner.go              # Startup banner
│   │   ├── config.go              # Configuration management
│   │   ├── errors.go              # Error handling
│   │   ├── logging.go             # Arbor logger setup
│   │   └── version.go             # Version management
│   ├── interfaces/                # Service interfaces
│   │   └── jira.go                # Interface definitions
│   ├── services/                  # Service implementations
│   │   ├── page_assessor.go       # Page validation
│   │   ├── storage.go             # BBolt database
│   │   └── webserver.go           # HTTP server
│   └── handlers/                  # HTTP handlers
│       ├── api.go                 # API endpoints
│       ├── ui.go                  # UI/HTMX handlers
│       ├── jira_parser.go         # Jira data parsing
│       └── jira_parser_details.go # Field extraction
├── pages/
│   └── index.html                 # HTMX dashboard
├── deployments/
│   ├── aktis-collector-jira.toml  # Configuration template
│   └── docker/                    # Docker deployment
├── scripts/
│   ├── build.ps1                  # Windows build script
│   └── build.sh                   # Linux/Mac build script
├── data/                          # Runtime data (created)
│   └── aktis-collector-jira.db    # BBolt database
├── backups/                       # Database backups (created)
├── go.mod                         # Go dependencies
├── .version                       # Build version tracking
├── README.md                      # User documentation
└── CLAUDE.md                      # Developer documentation (this file)
```