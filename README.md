# Aktis Collector - Jira

A production-ready Jira ticket collector built with the Aktis Plugin SDK, following aktis-receiver template standards.

## 🎯 System Architecture

This system implements a complete Jira ticket collection and analytics solution with an integrated web interface. The application operates in two modes:

### **Collection Mode** (Default)
- **Purpose**: Collects Jira tickets from configured projects in batches
- **Technology**: Go 1.24+ with Aktis Plugin SDK
- **Architecture**: Clean architecture following aktis-receiver standards
- **Storage**: BBolt database with automatic backups
- **Key Features**:
  - Batch processing for efficient data collection
  - Incremental updates (only fetch changed tickets)
  - Multi-project support with flexible configuration
  - Configurable filtering by issue types and statuses
  - Structured logging with arbor
  - Version management and build flags

### **Server Mode** (Web Interface)
- **Purpose**: Integrated web server with dashboard for monitoring and analytics
- **Technology**: HTMX-based dynamic UI with modern web standards
- **Key Features**:
  - Real-time statistics and metrics dashboard
  - Interactive data visualization
  - Ticket filtering and search capabilities
  - Project overview and analytics
  - Responsive design optimized for all devices
  - RESTful API endpoints for data access

## 🏗️ Project Structure

```
aktis-collector-jira/
├── cmd/
│   └── aktis-collector-jira/    # Main application entry point
│       └── main.go               # Startup sequence: Config → Logger → Banner
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

**Collection Mode (Default):**
```bash
./bin/aktis-collector-jira -config deployments/config.toml
```

**Server Mode (Web Interface):**
```bash
./bin/aktis-collector-jira -server -config deployments/config.toml
```

**Command Line Options:**
- `-version`: Show version information
- `-help`: Show help message
- `-config <path>`: Configuration file path (default: `./config.toml`)
- `-mode <env>`: Environment mode: dev/development/prod/production (default: dev)
- `-quiet`: Suppress banner output (for aktis-collector integration)
- `-update`: Run in update mode (incremental - fetch only latest changes)
- `-batch-size <n>`: Number of tickets to process per batch (default: 50)
- `-server`: Run in server mode with web interface
- `-validate`: Validate configuration file and exit

**Collection Examples:**
```bash
# Full collection in development mode
./bin/aktis-collector-jira -config deployments/config.toml

# Incremental update
./bin/aktis-collector-jira -config deployments/config.toml -update

# Production mode with custom batch size
./bin/aktis-collector-jira -config deployments/config.toml -mode prod -batch-size 100

# For Aktis platform integration (called by aktis-collector)
./bin/aktis-collector-jira -config deployments/config.toml -mode prod -quiet
```

**Server Mode Examples:**
```bash
# Start web interface on default port (8080)
./bin/aktis-collector-jira -server -config deployments/config.toml

# Start web interface in production mode
./bin/aktis-collector-jira -server -config deployments/config.toml -mode prod
```

### Web Interface

**Access the Dashboard:**
Open http://localhost:8080 in your browser (or the port configured in `web_port`)

**Features:**
- HTMX-based dynamic UI with real-time updates
- Interactive data visualization and analytics
- Project overview and ticket filtering
- Responsive design for desktop and mobile

**Dashboard API Endpoints:**
- `GET /api/dashboard` - Complete dashboard data
- `GET /api/projects` - List of projects
- `GET /api/tickets` - All tickets (supports filters)
- `POST /api/refresh` - Refresh data

## 📊 Key Features

### Collector Features
✅ **Batch Processing**: Configurable batch size for efficient collection
✅ **Incremental Updates**: Only fetch tickets updated since last run
✅ **Multi-Project Support**: Configure multiple Jira projects
✅ **Flexible Filtering**: Filter by issue types, statuses, custom fields
✅ **BBolt Database**: Embedded database with automatic backups and transactions
✅ **Data Retention**: Configurable cleanup of old data
✅ **Structured Logging**: Arbor logger with file and console output
✅ **Version Management**: Auto-increment build versioning with timestamps
✅ **Error Handling**: Comprehensive error handling and logging
✅ **Aktis Integration**: Full compliance with Aktis Plugin SDK

### Web Interface Features
✅ **Integrated Server**: Built-in web server with single-binary deployment
✅ **HTMX-Based UI**: Modern dynamic interface without complex JavaScript frameworks
✅ **Real-time Updates**: Live metrics and data refresh capabilities
✅ **Interactive Visualizations**: Charts and graphs for data analysis
✅ **Project Analytics**: Per-project statistics and trending
✅ **Advanced Filtering**: Multi-dimensional ticket filtering and search
✅ **Responsive Design**: Optimized for desktop, tablet, and mobile devices
✅ **RESTful API**: Clean API endpoints for external integrations

## 📈 Data Flow

### Collection Mode
```
Jira Cloud API
     |
     | [REST API Calls with Basic Auth + JQL queries]
     v
Jira Collector (internal/services/collector.go)
     |
     | [Batch processing + data transformation]
     v
Storage Layer (internal/services/storage.go)
     |
     | [BBolt Database: aktis-collector-jira.db]
     v
Local Database (./data/)
```

### Server Mode
```
BBolt Database (./data/aktis-collector-jira.db)
     |
     | [Direct database queries]
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

### Stored Ticket Format
```json
{
  "key": "DEV-123",
  "project": "DEV",
  "data": {
    "summary": "Fix login bug",
    "status": "In Progress",
    "priority": "High",
    "assignee": "John Doe",
    "created": "2025-09-25T09:00:00Z",
    "updated": "2025-09-26T10:00:00Z",
    "issue_type": "Bug",
    "description": "Detailed description..."
  },
  "hash": "a1b2c3d4",
  "collected": "2025-09-26T10:30:00Z",
  "updated": "2025-09-26T10:30:00Z",
  "version": 1
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

**To the Collector:**
1. **New Issue Fields**: Modify `ExtractIssueData()` in `internal/services/jira_client.go`
2. **Custom Processing**: Add methods to collector service in `internal/services/collector.go`
3. **Additional Storage**: Extend storage interface in `internal/interfaces/interfaces.go` and implement in `internal/services/storage.go`
4. **New API Endpoints**: Extend Jira client interface and implementation

**To the Web Interface:**
1. **New API Endpoints**: Add handlers in `internal/handlers/api/` directory
2. **Additional UI Components**: Extend HTMX templates in `pages/index.html`
3. **New Views**: Create new handler functions in `internal/handlers/ui/`
4. **Custom Analytics**: Add calculation functions in `internal/services/webserver.go`

## 🚀 Production Deployment

### Deployment Checklist
- [ ] Configure production Jira credentials
- [ ] Set up secure API token storage
- [ ] Configure appropriate batch sizes
- [ ] Set up automated collection schedules (cron/Task Scheduler)
- [ ] Configure data retention policies
- [ ] Set up monitoring and alerting
- [ ] Configure backup strategies
- [ ] Set up HTTPS for web interface
- [ ] Configure firewall rules
- [ ] Set up log rotation

### Integration with Aktis Platform

For use with the Aktis data collection platform:

```bash
./bin/aktis-collector-jira -config deployments/aktis-collector-jira.toml -mode prod -quiet
```

The `-quiet` flag outputs JSON payloads compatible with `aktis-collector`.

## 🔐 Security Considerations

- **API Token Security**: Never commit config files with real credentials
- **Data Privacy**: Local storage keeps sensitive data in-house
- **Access Control**: Dashboard server should be secured with authentication in production
- **Backup Security**: Backup files contain sensitive project data
- **Network Security**: Use HTTPS for production deployments

## 📊 Performance

### Collector Optimizations
- **Batch Processing**: Reduces API calls and memory usage
- **Incremental Updates**: Minimizes data transfer with JQL filters
- **Efficient Hashing**: SHA256 for change detection
- **Memory Management**: Streaming data processing
- **Error Recovery**: Graceful handling of network issues

### Dashboard Optimizations
- **Client-side Processing**: Reduces server load
- **Efficient Charts**: Optimized Plotly.js configurations
- **Lazy Loading**: Load data on demand
- **Caching**: Browser caching for static assets

## 📖 Documentation

- **CLAUDE.md**: Developer documentation with build commands and architecture
- **API Documentation**: See web interface endpoints above
- **Configuration**: See `deployments/aktis-collector-jira.toml` for all options

## 🎯 Use Cases

### Development Teams
- Sprint planning with ticket distribution analysis
- Bug tracking and resolution trends
- Resource planning and assignee workloads
- Real-time project status monitoring

### Project Management
- Executive dashboards with high-level metrics
- Team performance and productivity insights
- Quality metrics: bug rates and resolution times
- Capacity planning with workload distribution

### DevOps & Operations
- Automated data collection via cron jobs
- Integration with monitoring infrastructure
- Automated report generation
- Compliance and audit trail

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