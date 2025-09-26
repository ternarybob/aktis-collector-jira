# Jira Ticket Collector

A powerful Jira ticket collector plugin built with the Aktis Plugin SDK. This tool collects Jira tickets from configured projects in batches and maintains a local dataset with the ability to perform incremental updates.

## Features

- 🎯 **Batch Processing**: Collect tickets in configurable batches to handle large projects efficiently
- 🔄 **Incremental Updates**: Only fetch tickets that have been updated since the last collection
- 💾 **Local Dataset**: Maintains a local JSON dataset of all collected tickets
- 🗂️ **Multi-Project Support**: Configure multiple Jira projects to collect from
- 🔐 **Secure Authentication**: Uses Jira API tokens for secure access
- 📊 **Rich Metadata**: Captures comprehensive ticket information including history
- 🔄 **Automatic Backups**: Creates backups before updating datasets
- 🧹 **Data Retention**: Automatic cleanup of old data based on retention policies

## Installation

### Prerequisites

- Go 1.24 or higher
- Jira account with API token access
- Aktis Plugin SDK

### Build from Source

```bash
# Clone or download the project
cd jira-collector

# Download dependencies
go mod download

# Build the plugin
go build -o jira-collector main.go
```

## Configuration

Create a configuration file based on `config.example.json`:

```json
{
  "jira": {
    "base_url": "https://your-company.atlassian.net",
    "username": "your-email@company.com", 
    "api_token": "your-jira-api-token",
    "timeout_seconds": 30
  },
  "projects": [
    {
      "key": "DEV",
      "name": "Development Project",
      "issue_types": ["Bug", "Story", "Task", "Epic"],
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

### Jira API Token

To get your Jira API token:
1. Go to [https://id.atlassian.com/manage-profile/security/api-tokens](https://id.atlassian.com/manage-profile/security/api-tokens)
2. Click "Create API token"
3. Copy the generated token to your config file

## Usage

### Basic Collection

Collect all tickets from configured projects:
```bash
./jira-collector
```

### Production Mode

Run in production mode for automated collection:
```bash
./jira-collector -mode prod -quiet
```

### Update Mode

Only collect tickets that have been updated since last run:
```bash
./jira-collector -update
```

### Batch Size Control

Control the number of tickets processed in each batch:
```bash
./jira-collector -batch-size 100
```

### Custom Configuration

Use a custom configuration file:
```bash
./jira-collector -config /path/to/config.json
```

### Command Line Options

```
  -mode string        Environment mode: 'dev', 'development', 'prod', or 'production' (default "dev")
  -config string      Configuration file path
  -quiet              Suppress banner output (for aktis-collector integration)
  -version            Show version information
  -help               Show help message
  -update             Run in update mode (fetch only latest changes)
  -batch-size int     Number of tickets to process in each batch (default 50)
```

## Data Structure

### Stored Data Format

Collected tickets are stored in JSON format with the following structure:

```json
{
  "project_key": "DEV",
  "last_update": "2025-09-26T10:30:00Z",
  "total_count": 150,
  "tickets": {
    "DEV-123": {
      "key": "DEV-123",
      "project": "DEV",
      "data": {
        "summary": "Fix login bug",
        "status": "In Progress",
        "priority": "High",
        "assignee": "John Doe",
        "created": "2025-09-25T09:00:00Z",
        "updated": "2025-09-26T10:00:00Z"
      },
      "collected": "2025-09-26T10:30:00Z",
      "updated": "2025-09-26T10:30:00Z",
      "version": 1
    }
  }
}
```

### Output Payloads

The collector generates payloads with the following structure:

```json
{
  "success": true,
  "timestamp": "2025-09-26T10:30:00Z",
  "payloads": [
    {
      "timestamp": "2025-09-26T10:30:00Z",
      "type": "jira_bug",
      "data": {
        "key": "DEV-123",
        "summary": "Fix login bug",
        "status": "In Progress",
        "priority": "High"
      },
      "metadata": {
        "project": "DEV",
        "ticket_id": "DEV-123",
        "source": "jira",
        "mode": "full_collection"
      }
    }
  ],
  "collector": {
    "name": "jira-collector",
    "type": "data",
    "version": "1.0.0",
    "environment": "development"
  },
  "stats": {
    "duration": "2m15s",
    "payload_count": 150
  }
}
```

## Integration with Aktis

This plugin is designed to work with the Aktis data collection platform. When run with the `-quiet` flag, it outputs JSON that can be consumed by the aktis-collector.

```bash
# For use with aktis-collector
./jira-collector -mode prod -quiet
```

## Data Management

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

### Backup Strategy

- Automatic backups are created before each update
- Backups are timestamped for easy identification
- Configure retention policies in the storage config

### Data Retention

Configure automatic cleanup of old data:
```json
{
  "storage": {
    "retention_days": 90
  }
}
```

## Troubleshooting

### Common Issues

1. **Authentication Errors**
   - Verify your API token is correct
   - Check that your username matches your Jira email
   - Ensure the API token has proper permissions

2. **Connection Timeouts**
   - Increase the timeout in the configuration
   - Check network connectivity to your Jira instance
   - Verify the base URL is correct

3. **Rate Limiting**
   - Reduce the batch size to avoid hitting API limits
   - Add delays between batches if needed

4. **Memory Issues**
   - Decrease batch size for large projects
   - Ensure sufficient disk space for data storage

### Debug Mode

Run in development mode for detailed output:
```bash
./jira-collector -mode dev
```

## Development

### Project Structure

```
jira-collector/
├── main.go                 # Entry point
├── internal/
│   └── collector/
│       ├── config.go       # Configuration management
│       ├── jira_client.go  # Jira API client
│       ├── storage.go      # Data persistence
│       └── collector.go    # Main collection logic
├── config.example.json     # Example configuration
├── go.mod                  # Go module definition
└── README.md              # This file
```

### Adding Features

1. **New Issue Types**: Update the `ExtractIssueData` function in `jira_client.go`
2. **Custom Fields**: Add field extraction logic in the same function
3. **New Storage Formats**: Implement additional storage backends in `storage.go`
4. **Additional APIs**: Extend the `JiraClient` with new API methods

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues and questions:
- Check the troubleshooting section
- Review the example configuration
- Open an issue in the repository