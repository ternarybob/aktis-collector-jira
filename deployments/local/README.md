# Local Deployment Configuration

This directory contains configuration files for local development and testing of the Aktis Jira Collector.

## Files

- `aktis-collector-jira.toml` - Local development configuration

## Quick Start

### 1. Build the Application

```bash
# Windows
.\scripts\build.ps1

# Linux/Mac
./scripts/build.sh
```

### 2. Configure

Edit `aktis-collector-jira.toml` and set your Jira instance URL:

```toml
[jira]
base_url = "https://your-company.atlassian.net"
```

### 3. Run Locally

```bash
# Windows
.\bin\aktis-collector-jira.exe -config deployments\local\aktis-collector-jira.toml

# Linux/Mac
./bin/aktis-collector-jira -config deployments/local/aktis-collector-jira.toml
```

Or use the deployment script:

```bash
# Windows
.\scripts\deploy.ps1 -Target local

# With rebuild
.\scripts\deploy.ps1 -Target local -Build
```

### 4. Access Web Interface

Open your browser to:
```
http://localhost:8080
```

### 5. Install Chrome Extension

1. Open Chrome and navigate to `chrome://extensions/`
2. Enable "Developer mode" (top right)
3. Click "Load unpacked"
4. Select the directory: `bin/aktis-chrome-extension/`

The extension will appear in your browser toolbar.

## Configuration Notes

### Current Implementation

The system currently uses **Chrome Extension-based collection**:
- Navigate to Jira ticket pages in your browser
- Click the extension icon to collect data
- Data is sent to the local server via POST /receiver
- Server parses and stores data in BBolt database

### Planned Features

Future versions will support:
- **API Collection**: Direct Jira REST API access
- **Browser Scraping**: Automated browser-based collection

## Development Workflow

### Testing Changes

1. Make code changes
2. Rebuild: `.\scripts\build.ps1`
3. Restart server: `.\scripts\deploy.ps1 -Restart`
4. Reload extension in Chrome (if needed)

### Running Tests

```bash
# Windows
.\scripts\test.ps1

# With coverage
.\scripts\test.ps1 -Coverage
```

### Viewing Logs

Check console output or use the web interface:
- Activity log tab shows recent operations
- Database tab shows stored tickets

## Troubleshooting

### Server Won't Start

- Check if port 8080 is already in use
- Verify config file path is correct
- Check for errors in console output

### Extension Not Working

- Ensure server is running
- Check CORS settings in browser console
- Verify extension permissions in Chrome

### Data Not Appearing

- Check Activity Log in web interface
- Verify project key extraction from tickets
- Review console output for parsing errors

## Directory Structure

```
deployments/local/
├── aktis-collector-jira.toml   # Local configuration
└── README.md                    # This file
```

## See Also

- [Main README](../../README.md) - Full project documentation
- [CLAUDE.md](../../CLAUDE.md) - Developer documentation
- [Docker Deployment](../docker/README.md) - Docker deployment guide
