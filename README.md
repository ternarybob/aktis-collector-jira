# Aktis Collector - Jira

A production-ready Jira ticket collector built with the Aktis Plugin SDK, following aktis-receiver template standards.

## 🎯 System Architecture

This system implements a complete Jira ticket collection and analytics solution. It consists of three main components:

### 1. **Jira Collector** (`cmd/aktis-collector-jira/`)
- **Purpose**: Collects Jira tickets from configured projects in batches
- **Technology**: Go 1.24+ with Aktis Plugin SDK
- **Architecture**: Clean architecture following aktis-receiver standards
- **Key Features**:
  - Batch processing for efficient data collection
  - Incremental updates (only fetch changed tickets)
  - Local JSON dataset storage with automatic backups
  - Multi-project support
  - Configurable filtering by issue types and statuses
  - Structured logging with arbor
  - Version management and build flags

### 2. **Web Dashboard** (`web-interface/`)
- **Purpose**: Visual interface for monitoring and analyzing collected data
- **Technology**: HTML/CSS/JavaScript with Plotly.js
- **Key Features**:
  - Real-time statistics and metrics
  - Interactive charts and graphs
  - Ticket filtering and search
  - Project overview dashboard
  - Responsive design with modern UI

### 3. **Dashboard Server** (`web-interface/server.go`)
- **Purpose**: Serves the web interface and provides API endpoints
- **Technology**: Go HTTP server
- **Key Features**:
  - RESTful API for dashboard data
  - Static file serving
  - Data aggregation and statistics
  - Cross-origin support for API calls

## 🏗️ Project Structure

```
aktis-collector-jira/
├── cmd/
│   └── aktis-collector-jira/    # Main application entry point
│       └── main.go               # Startup sequence: Config → Logger → Banner
├── internal/
│   ├── common/                   # Infrastructure layer (aktis-receiver template)
│   │   ├── banner.go             # Startup banner display
│   │   ├── config.go             # Configuration management
│   │   ├── errors.go             # Structured error handling
│   │   ├── logging.go            # Arbor logger integration
│   │   └── version.go            # Version management
│   └── collector/                # Business logic
│       ├── collector.go          # Main collection orchestration
│       ├── config.go             # Collection configuration
│       ├── jira_client.go        # Jira API integration
│       └── storage.go            # Data persistence
├── deployments/
│   ├── config.example.json       # Configuration template
│   ├── docker/                   # Docker deployment
│   │   ├── Dockerfile            # Multi-stage build
│   │   ├── docker-compose.yml    # Service definition
│   │   └── .env.example          # Environment variables
│   └── local/                    # Local deployment configs
├── scripts/
│   ├── build.ps1                 # Windows build with versioning
│   └── build.sh                  # Linux/Mac build script
├── web-interface/                # Dashboard UI and server
│   ├── index.html                # Dashboard interface
│   ├── app.js                    # Dashboard logic
│   └── server.go                 # API server
├── .github/workflows/
│   └── ci-cd.yml                 # GitHub Actions pipeline
├── go.mod                        # Go module definition
├── .version                      # Version tracking
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
cp configs/config.example.toml configs/config.toml
```

Edit `configs/config.toml`:

```toml
[collector]
name = "aktis-collector-jira"
environment = "development"
send_limit = 100  # Maximum payloads per run (for aktis-collector scheduling)

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
# Database file location - defaults to {executable_location}/data/{exec_name}.db
database_path = "./data/aktis-collector-jira.db"
backup_dir = "./backups"
retention_days = 90
```

**Configuration Priority:** Defaults → TOML file → Environment variables → Command line flags

#### Get Your Jira API Token
1. Go to [https://id.atlassian.com/manage-profile/security/api-tokens](https://id.atlassian.com/manage-profile/security/api-tokens)
2. Click "Create API token"
3. Copy the generated token to your config file

### Running the Collector

**Basic Collection:**
```bash
./bin/aktis-collector-jira -config configs/config.toml
```

**Command Line Options:**
- `-version`: Show version information
- `-help`: Show help message
- `-config <path>`: Configuration file path (default: `./config.json`)
- `-mode <env>`: Environment mode: dev/development/prod/production (default: dev)
- `-quiet`: Suppress banner output (for aktis-collector integration)
- `-update`: Run in update mode (incremental - fetch only latest changes)
- `-batch-size <n>`: Number of tickets to process per batch (default: 50)

**Examples:**
```bash
# Full collection in development mode
./bin/aktis-collector-jira -config configs/config.toml

# Incremental update
./bin/aktis-collector-jira -config configs/config.toml -update

# Production mode with custom batch size
./bin/aktis-collector-jira -config configs/config.toml -mode prod -batch-size 100

# For Aktis platform integration (called by aktis-collector)
./bin/aktis-collector-jira -config configs/config.toml -mode prod -quiet
```

### Running the Dashboard

**Start the Dashboard Server:**
```bash
cd web-interface
go run server.go
```

**Access the Dashboard:**
Open http://localhost:8080 in your browser

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
✅ **Local Storage**: JSON-based dataset with automatic backups
✅ **Data Retention**: Configurable cleanup of old data
✅ **Structured Logging**: Arbor logger with file and console output
✅ **Version Management**: Build flags inject version/build/commit info
✅ **Error Handling**: Comprehensive error handling and logging
✅ **Aktis Integration**: Full compliance with Aktis Plugin SDK

### Dashboard Features
✅ **Real-time Statistics**: Live metrics and counters
✅ **Interactive Charts**: Plotly.js powered visualizations
✅ **Status Distribution**: Pie charts showing ticket status breakdown
✅ **Priority Analysis**: Bar charts for priority distribution
✅ **Project Overview**: Per-project statistics and metrics
✅ **Ticket Filtering**: Filter by project, status, priority
✅ **Recent Activity**: Latest updated tickets display
✅ **Responsive Design**: Works on desktop and mobile

## 📈 Data Flow

```
Jira Cloud API
     |
     | [REST API Calls with Basic Auth]
     v
Jira Collector (cmd/aktis-collector-jira)
     |
     | [Batch processing + JQL queries]
     v
Storage Layer (internal/collector/storage.go)
     |
     | [JSON Files: {project}_tickets.json]
     v
Local Dataset (./data/)
     |
     | [HTTP API]
     v
Dashboard Server (web-interface/server.go)
     |
     | [JSON Responses]
     v
Web Dashboard (web-interface/index.html)
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
├── dev_tickets.json      # Development project tickets
├── proj_tickets.json     # Product project tickets
└── ...

./backups/
├── dev_tickets.json.20250926_103000.bak
├── proj_tickets.json.20250926_103015.bak
└── ...
```

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
1. **New Issue Fields**: Modify `ExtractIssueData()` in `internal/collector/jira_client.go`
2. **Custom Processing**: Add methods to `JiraCollector` struct in `internal/collector/collector.go`
3. **Additional Storage**: Extend `Storage` in `internal/collector/storage.go`
4. **New API Endpoints**: Extend `JiraClient` in `internal/collector/jira_client.go`

**To the Dashboard:**
1. **New Charts**: Add Plotly.js configurations in `web-interface/app.js`
2. **Additional Filters**: Extend filtering logic in dashboard
3. **New Metrics**: Add calculation functions in `web-interface/server.go`
4. **Custom Views**: Create new HTML sections in `web-interface/index.html`

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
./bin/aktis-collector-jira -config deployments/config.json -mode prod -quiet
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
- **API Documentation**: See dashboard server endpoints above
- **Configuration**: See `configs/config.example.toml` for all options

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