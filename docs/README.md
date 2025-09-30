# Aktis Collector Jira - Documentation Index

Welcome to the Aktis Collector Jira documentation. This system provides a Chrome extension-based solution for collecting Jira project and ticket data.

## 📚 Documentation

### For Users

**[User Guide](USER_GUIDE.md)** - Complete guide for installing and using the collector
- Quick start instructions
- Step-by-step collection process
- Configuration options
- Troubleshooting
- FAQ

### For Developers

**[Developer Guide](DEVELOPER_GUIDE.md)** - Technical guide for developers
- Project structure overview
- Code walkthrough
- Testing strategies
- Using this as a template
- Debugging tips
- Performance optimization

### Architecture

**[Architecture Documentation](ARCHITECTURE.md)** - System design and data flow
- Component overview
- System architecture diagrams
- Data flow processes
- API endpoint reference
- Page type detection
- HTML parsing strategy
- Template pattern explanation

## 🚀 Quick Start

### Prerequisites
- Chrome browser
- Go 1.24+
- Windows/Linux/Mac

### Installation (5 minutes)

1. **Start the server:**
   ```bash
   cd C:\development\aktis\aktis-collector-jira\bin
   .\aktis-collector-jira.exe
   ```

2. **Install Chrome extension:**
   - Open `chrome://extensions/`
   - Enable "Developer mode"
   - Click "Load unpacked"
   - Select: `cmd/aktis-chrome-extension`

3. **Configure extension:**
   - Click extension icon
   - Settings tab
   - Server URL: `http://localhost:8084`
   - Save Settings

### First Collection (2 minutes)

1. Navigate to `https://[your-company].atlassian.net/jira/projects`
2. Open extension side panel
3. Click "Collect Current Page"
4. View Buffer tab to see collected projects

## 📖 What to Read

### I want to...

#### Use the collector to collect Jira data
→ **[User Guide](USER_GUIDE.md)**

#### Understand how it works
→ **[Architecture Documentation](ARCHITECTURE.md)**

#### Adapt this for another website
→ **[Developer Guide](DEVELOPER_GUIDE.md)** → "Adapting as a Template" section

#### Fix a bug or contribute
→ **[Developer Guide](DEVELOPER_GUIDE.md)** → "Testing" and "Debugging" sections

#### Deploy to production
→ **[Developer Guide](DEVELOPER_GUIDE.md)** → "Build & Package" section

## 🏗️ System Overview

```
User → Chrome Extension → Go HTTP Server → BoltDB
  │                            │
  └─ Collects HTML            └─ Parses & Stores
```

**Key Features:**
- ✅ Extension-based collection (no API keys needed)
- ✅ Server-side HTML parsing
- ✅ Local BoltDB storage
- ✅ Real-time buffer viewing
- ✅ Template for other collectors

## 🔧 Components

| Component | Purpose | Technology |
|-----------|---------|------------|
| **Chrome Extension** | Collects HTML from Jira pages | JavaScript (Manifest V3) |
| **Go HTTP Server** | Processes and stores data | Go 1.24, Gorilla Mux |
| **HTML Parser** | Extracts structured data | golang.org/x/net/html |
| **BoltDB** | Persistent storage | BoltDB (embedded DB) |

## 📊 Data Flow

### Phase 1: Project Discovery
```
User navigates to projects page
  → Extension detects "projectsList" page type
  → User clicks "Collect"
  → Extension sends HTML to server
  → Server parses and extracts projects
  → Server stores in database
  → Extension displays projects in Buffer
```

### Phase 2: Ticket Collection
```
User navigates to project issue list
  → Extension detects "issueList" page type
  → User clicks "Collect"
  → Extension sends HTML to server
  → Server extracts ticket keys & summaries
  → Server stores tickets by project
  → Extension shows ticket count in Buffer
```

### Phase 3: Detailed Collection
```
User navigates to individual ticket page
  → Extension detects "issue" page type
  → User clicks "Collect"
  → Extension sends HTML to server
  → Server extracts full ticket details
  → Server merges with existing ticket
  → Extension displays updated data
```

## 🎯 Use Cases

### Data Migration
Collect all Jira data before migrating to another system

### Backup & Archive
Create local backups of Jira projects

### Analytics
Export data for custom analysis and reporting

### Integration
Feed data to other systems without API limitations

### Template for Other Collectors
Adapt for Confluence, Trello, GitHub, Linear, etc.

## 🔒 Privacy & Security

- ✅ All data stored locally
- ✅ No external transmission
- ✅ No API keys or credentials needed
- ✅ User controls all collection
- ✅ Open source and auditable

## 🛠️ Configuration

### Server Configuration
File: `deployments/aktis-collector-jira.toml`

```toml
[collector]
name = "aktis-collector-jira"
environment = "development"
port = 8084

[storage]
database_path = "./data/aktis-collector-jira.db"

[logging]
level = "info"
log_file = "./logs/aktis-collector-jira.log"
```

### Extension Configuration
- **Server URL**: `http://localhost:8084` (default)
- **Auto-collect**: Disabled (collect on button click)
- **Auto-navigate**: Disabled (manual navigation)

## 📈 Statistics

View collection statistics:

```bash
curl http://localhost:8084/status
```

Example output:
```json
{
  "collector": {
    "running": true,
    "uptime": 3600
  },
  "stats": {
    "total_tickets": 247,
    "last_collection": "2025-09-30 14:45:00"
  }
}
```

## 🧪 Testing

### Test Server Connection
```bash
curl http://localhost:8084/health
```

### Test Data Collection
```bash
curl http://localhost:8084/database/data
```

### View Extension Logs
1. Go to `chrome://extensions/`
2. Find "Aktis Jira Collector"
3. Click "Inspect views: service worker"
4. Check Console tab

## 🐛 Troubleshooting

### Extension shows "Offline"
**Solution**: Ensure server is running on port 8084

### No data collected
**Solution**: Check browser console for errors, verify HTML structure matches parser

### "Receiving end does not exist" error
**Solution**: Extension now includes fallback, reload extension if persists

**More troubleshooting**: See [User Guide - Troubleshooting](USER_GUIDE.md#troubleshooting)

## 📦 Database Structure

```
aktis-collector-jira.db
├── projects/
│   └── [project_key] → JSON
└── tickets/
    └── [project_key]_tickets/
        └── [ticket_key] → JSON
```

**Export database:**
```bash
curl http://localhost:8084/database/data > export.json
```

## 🔄 Version History

- **v0.1.0** - Initial release
  - Chrome extension with side panel UI
  - Server-side HTML parsing
  - BoltDB storage
  - Projects and tickets collection

## 🤝 Contributing

Contributions welcome! See [Developer Guide](DEVELOPER_GUIDE.md) for:
- Code structure
- Testing guidelines
- Pull request process

## 📝 License

[Add your license here]

## 🔗 Related Projects

- **aktis-receiver**: Backend system for receiving collected data
- **aktis-plugin-sdk**: Plugin framework for collectors

## 📞 Support

- **Issues**: Report bugs or request features
- **Documentation**: Check guides in this folder
- **Logs**: Review server logs in `logs/` directory

## 🎓 Learning Resources

### Chrome Extensions
- [Chrome Extension Docs](https://developer.chrome.com/docs/extensions/)
- [Manifest V3 Guide](https://developer.chrome.com/docs/extensions/mv3/intro/)

### Go
- [Go Documentation](https://go.dev/doc/)
- [BoltDB Guide](https://github.com/etcd-io/bbolt)

### HTML Parsing
- [golang.org/x/net/html](https://pkg.go.dev/golang.org/x/net/html)
- [Regex Reference](https://regex101.com/)

---

## 📋 Document Map

```
docs/
├── README.md              ← You are here
├── USER_GUIDE.md         ← For end users
├── DEVELOPER_GUIDE.md    ← For developers
└── ARCHITECTURE.md       ← System design

../
├── cmd/
│   └── aktis-chrome-extension/  ← Extension source
├── internal/                    ← Go server source
└── deployments/                 ← Configuration files
```

---

**Last Updated**: 2025-09-30
**Version**: 0.1.58