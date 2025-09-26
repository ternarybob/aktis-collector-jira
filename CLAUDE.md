# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Jira ticket collection and analytics system with three main components:
- **Jira Collector Plugin** (`internal/`): Go-based collector using Aktis Plugin SDK
- **Web Dashboard Server** (`web-interface/server.go`): Go HTTP server providing REST API
- **Web Dashboard UI** (`web-interface/`): HTML/JS interface with Plotly.js visualizations

## Build Commands

### Collector
```bash
cd internal
./build.sh
# OR manually:
go mod download
go build -o jira-collector main.go
```

### Web Server
```bash
cd web-interface
go build -o dashboard-server server.go
```

### Full Deployment
```bash
./scripts/deploy.sh
```

## Running the System

### Start Web Dashboard
```bash
cd web-interface
./dashboard-server
# Access at http://localhost:8080
```

### Run Collector
```bash
cd internal
./jira-collector -config config.example.json

# Common flags:
#   -mode prod              # Production mode
#   -quiet                  # JSON output for Aktis integration
#   -update                 # Incremental update (only changed tickets)
#   -batch-size 100         # Tickets per batch (default: 50)
```

## Architecture

### Data Flow
```
Jira API → Collector (batch processing) → JSON Storage (data/*.json)
                                              ↓
                                    Web Server (aggregation)
                                              ↓
                                    Web Dashboard (visualization)
```

### Key Components

**Collector** (`internal/collector/`)
- `collector.go`: Main collection orchestration, batch processing
- `jira_client.go`: Jira REST API integration, JQL query building
- `config.go`: Configuration management
- `storage.go`: JSON file persistence, backup management

**Storage Pattern**
- Format: `data/{project_key}_tickets.json`
- Backups: `backups/{project_key}_tickets.json.{timestamp}.bak`
- Structure: `ProjectDataset` contains `map[string]*TicketData`

**Web Server** (`web-interface/server.go`)
- Endpoints: `/api/dashboard`, `/api/projects`, `/api/tickets`, `/api/refresh`
- Aggregates statistics from JSON files on each request
- No database - reads directly from collector's JSON output

### Integration Points

**Aktis Plugin SDK**
- Collector implements Aktis plugin interface
- Use `-quiet` flag for JSON output consumed by `aktis-collector`
- Payloads typed as `jira_{issue_type}` (e.g., `jira_bug`, `jira_story`)

**JQL Query Building**
- `BuildJQL()` constructs queries from project config
- Supports filtering by issue types, statuses, and update timestamps
- Used for both full collection and incremental updates

## Configuration

Located at `internal/config.example.json`:
```json
{
  "jira": {
    "base_url": "https://company.atlassian.net",
    "username": "email@company.com",
    "api_token": "token",
    "timeout_seconds": 30
  },
  "projects": [
    {
      "key": "DEV",
      "name": "Development Project",
      "issue_types": ["Bug", "Story", "Task"],
      "statuses": ["To Do", "In Progress", "Done"],
      "max_results": 1000
    }
  ],
  "storage": {
    "data_dir": "./data",
    "backup_dir": "./backups",
    "retention_days": 90
  }
}
```

## Go Version

Go 1.24 or higher required (specified in `internal/go.mod`)

## Dependencies

- `github.com/ternarybob/aktis-plugin-sdk v0.1.2` - Plugin framework
- `github.com/go-resty/resty/v2 v2.16.2` - HTTP client for Jira API
- Plotly.js (web interface) - Chart visualization

## Development Notes

**Error Handling**
- Collector continues on individual ticket failures
- Comprehensive error wrapping with context
- Storage operations include automatic backup before updates

**Batch Processing**
- Default batch size: 50 tickets
- Configurable via `-batch-size` flag
- Pagination handled automatically via `startAt` offset

**Incremental Updates**
- `UpdateTickets()` checks `last_update` timestamp per project
- Falls back to full collection if no previous data exists
- JQL includes `updated >= {timestamp}` filter

**Data Hashing**
- Each ticket has SHA256 hash (first 8 chars) for change detection
- Generated from JSON-serialized ticket data
- Stored in `TicketData.Hash` field

## File Structure

```
internal/              # Collector plugin
├── collector/         # Core collection logic
├── main.go           # Entry point
├── config.example.json
├── go.mod
└── build.sh

web-interface/        # Dashboard
├── server.go         # Go HTTP server
├── index.html        # Main UI
├── app.js           # Dashboard logic
└── data/            # Collector output (JSON files)

scripts/
└── deploy.sh        # Full system deployment
```