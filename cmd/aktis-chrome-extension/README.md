# Aktis Jira Collector - Chrome Extension

Chrome extension for collecting Jira page data and sending it to the Aktis Collector server.

## Features

- Collects data from Jira issue pages, boards, and search results
- Sends data to local Aktis Collector server
- Configurable auto-collection on page load
- Future: Follow links to chase project items

## Installation

### Development Mode

1. Open Chrome and navigate to `chrome://extensions/`
2. Enable "Developer mode" (toggle in top right)
3. Click "Load unpacked"
4. Select the `cmd/aktis-chrome-extension` directory

### Building for Distribution

Run the build script:
```bash
./cmd/aktis-chrome-extension/build.sh
```

This will create a `dist` directory with the packaged extension.

## Usage

1. Start the Aktis Collector server:
   ```bash
   ./bin/aktis-collector-jira.exe -config aktis-collector-jira.toml
   ```

2. Navigate to a Jira page (e.g., https://your-company.atlassian.net)

3. Click the Aktis extension icon in your browser toolbar

4. Configure settings:
   - **Server URL**: Address of your Aktis Collector server (default: `http://localhost:8080`)
   - **Auto-collect**: Enable to automatically collect data when Jira pages load
   - **Follow links**: Enable to automatically follow and collect linked items (future feature)

5. Click "Collect Current Page" to manually collect data from the current Jira page

## Configuration

The extension stores configuration in Chrome's sync storage:

```javascript
{
  serverUrl: 'http://localhost:8080',
  autoCollect: false,
  followLinks: false
}
```

## Supported Jira Pages

- **Issue pages**: `/browse/PROJECT-123`
- **Board/Backlog**: `/board/`, `/secure/RapidBoard`
- **Search results**: `/issues/`
- **Project pages**: `/projects/`

## Data Format

Data sent to server:

```json
{
  "timestamp": "2025-09-30T10:54:00Z",
  "url": "https://company.atlassian.net/browse/PROJ-123",
  "title": "Page Title",
  "data": {
    "pageType": "issue",
    "html": "...",
    "issue": {
      "key": "PROJ-123",
      "summary": "Issue summary",
      "description": "Issue description",
      "issueType": "Bug",
      "status": "In Progress",
      "priority": "High",
      "assignee": "John Doe",
      "labels": ["backend", "api"],
      "components": ["Core"]
    }
  },
  "collector": {
    "name": "aktis-jira-collector-extension",
    "version": "0.1.0"
  }
}
```

## Development

### Files

- `manifest.json`: Extension manifest (Manifest V3)
- `background.js`: Service worker (handles data forwarding to server)
- `content.js`: Content script (extracts data from Jira pages)
- `popup.html/js`: Extension popup UI
- `icons/`: Extension icons (16x16, 48x48, 128x128)

### Testing

1. Make changes to extension files
2. Go to `chrome://extensions/`
3. Click the reload icon for the Aktis extension
4. Test on Jira pages

## Icon Placeholders

The extension requires icons in the `icons/` directory:
- `icon16.png` (16x16)
- `icon48.png` (48x48)
- `icon128.png` (128x128)

You'll need to create these icons or the extension will show default Chrome extension icons.

## Server Endpoint

The extension sends data to: `POST http://localhost:8080/receiver`

See `internal/handlers/api.go` for server-side receiver implementation.