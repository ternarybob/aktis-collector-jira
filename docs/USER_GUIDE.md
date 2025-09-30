# Aktis Collector Jira - User Guide

## Quick Start

### Prerequisites
- Chrome browser (or Chromium-based browser)
- Windows/Linux/Mac with Go 1.24+
- Access to Jira (Atlassian Cloud)

### Installation

#### 1. Start the Server

```bash
cd C:\development\aktis\aktis-collector-jira
.\bin\aktis-collector-jira.exe
```

The server will start on **http://localhost:8084**

Look for this output:
```
13:55:35 INF > Starting Aktis Collector Jira Service
13:55:35 INF > Web server started successfully port=8084
```

#### 2. Install Chrome Extension

1. Open Chrome and navigate to `chrome://extensions/`
2. Enable "Developer mode" (toggle in top-right)
3. Click "Load unpacked"
4. Select folder: `C:\development\aktis\aktis-collector-jira\cmd\aktis-chrome-extension`
5. Extension icon appears in toolbar

#### 3. Configure Extension

1. Click extension icon to open side panel
2. Go to **Settings** tab
3. Verify Server URL: `http://localhost:8084`
4. Click **Save Settings**
5. Server Status should show **Online** ✓

## Step-by-Step Collection Process

### Step 1: Collect Projects List

1. **Navigate** to your Jira projects page:
   - URL format: `https://[your-company].atlassian.net/jira/projects`
   - Example: `https://acme.atlassian.net/jira/projects`

2. **Open extension** side panel (click extension icon)

3. **Verify detection**:
   - Go to **Collect** tab
   - Server Status: **Online** (green)
   - Page Type: **projectsList**
   - Last Collection: **Never**

4. **Click "Collect Current Page"** button

5. **Wait for confirmation**:
   - Success message: "Page collected successfully!"
   - Last Collection updates to current time

6. **View collected projects**:
   - Switch to **Buffer** tab
   - See list of projects with counts
   - Example:
     ```
     PROJ (0 tickets)
     DEV (0 tickets)
     TEST (0 tickets)
     ```

### Step 2: Collect Project Tickets

#### Option A: Collect from Issue List Page

1. **Navigate** to a project's issue list:
   - URL format: `https://[company].atlassian.net/jira/software/c/projects/[KEY]/issues`
   - Example: `https://acme.atlassian.net/jira/software/c/projects/PROJ/issues`

2. **Open extension** → **Collect** tab

3. **Verify detection**:
   - Page Type: **issueList**

4. **Click "Collect Current Page"**

5. **Check Buffer tab**:
   - Project now shows ticket count: `PROJ (15 tickets)`
   - Expand project to see ticket keys and summaries

#### Option B: Collect Individual Ticket

1. **Navigate** to specific ticket:
   - URL format: `https://[company].atlassian.net/browse/[KEY-123]`
   - Example: `https://acme.atlassian.net/browse/PROJ-123`

2. **Open extension** → **Collect** tab

3. **Verify detection**:
   - Page Type: **issue**

4. **Click "Collect Current Page"**
   - Collects detailed ticket information:
     - Full description
     - Comments
     - Custom fields
     - History (if available)

5. **Check Buffer tab**:
   - Ticket appears with full details

### Step 3: Collect Multiple Pages

To collect all tickets from a project:

1. **Navigate** to project issue list (Step 2, Option A)
2. **Collect current page** (first page of results)
3. **Click "Next"** pagination button in Jira
4. **Collect current page** again
5. **Repeat** for all pages

**Tips:**
- Jira typically shows 50-100 tickets per page
- Watch the ticket count increase in Buffer tab
- Take breaks to avoid rate limiting

### Step 4: View Collected Data

#### In Extension Buffer Tab

- **Expandable project groups**:
  ```
  ▼ PROJ (47 tickets)
    PROJ-1: Fix login bug
    PROJ-2: Add dark mode
    PROJ-3: Update documentation
    ...
  ```

- **Click project name** to expand/collapse

- **Ticket count** shows in header: "47 tickets"

#### Check Database

View raw database data:

```bash
curl http://localhost:8084/database/data
```

Example response:
```json
{
  "PROJ": {
    "PROJ-1": {
      "key": "PROJ-1",
      "summary": "Fix login bug",
      "description": "Users cannot login after...",
      "issue_type": "Bug",
      "status": "In Progress",
      "priority": "High",
      "updated": "2025-09-30T14:30:00Z"
    }
  }
}
```

### Step 5: Refresh Data

#### Full Refresh

1. Go to **Buffer** tab
2. Note current ticket counts
3. Navigate back to Jira pages
4. Re-collect each page (Steps 1-2)
5. Server updates existing tickets
6. New tickets are added

#### Incremental Update

*Coming soon*: Server will track last update time and only refresh changed tickets

### Step 6: Clear Buffer

**Warning**: This deletes all collected data!

1. Go to **Buffer** tab
2. Click **"Clear Buffer"** button
3. Confirm deletion
4. Buffer shows "No tickets in buffer"
5. Database is empty

## Page Type Reference

| Page Type | What It Collects | Best For |
|-----------|------------------|----------|
| **projectsList** | Project keys, names, types | Initial discovery |
| **issueList** | Ticket keys, summaries | Bulk collection |
| **issue** | Full ticket details | Deep data collection |
| **board** | Tickets on board | Kanban/Scrum boards |
| **search** | Search result tickets | Filtered queries |
| **generic** | Any Jira page | Fallback extraction |

## Configuration Options

### Auto-collect on page load
- **Disabled** (default): User must click "Collect Current Page"
- **Enabled**: Automatically collects when Jira page loads

**Use case**: When manually navigating many pages

### Auto-navigate to follow links
- **Disabled** (default): User navigates manually
- **Enabled**: Extension follows links automatically

**Use case**: Future feature for automated crawling

### Server URL
- **Default**: `http://localhost:8084`
- **Change to**: Custom port if needed

## Troubleshooting

### Extension shows "Offline"

**Check server is running:**
```bash
# Windows PowerShell
netstat -ano | findstr "8084"

# Should show:
TCP    0.0.0.0:8084    0.0.0.0:0    LISTENING
```

**Restart server:**
```bash
cd C:\development\aktis\aktis-collector-jira\bin
.\aktis-collector-jira.exe
```

### "Collection failed: Could not establish connection"

1. Verify server URL in Settings tab
2. Check server logs for errors
3. Try accessing http://localhost:8084/health in browser
4. Ensure no firewall blocking

### Page Type shows "unknown"

**Cause**: Not on a recognized Jira page

**Solutions**:
- Navigate to a Jira Cloud URL (*.atlassian.net)
- Check URL matches patterns:
  - `/jira/projects` → Projects list
  - `/browse/[KEY]` → Issue page
  - `/jira/software/c/projects/[KEY]/issues` → Issue list

### No tickets collected

**Possible causes:**
1. Page HTML structure doesn't match parser expectations
2. Jira using different layout/theme
3. Content loaded via JavaScript after page load

**Debug steps:**
1. Open browser DevTools (F12)
2. Go to Console tab
3. Look for extension errors
4. Check server logs for parsing errors

**Check server logs:**
```
13:55:42 INF > Received data from Chrome extension
13:55:42 DBG > Processing extension data page_type=issueList
13:55:42 INF > Extracted issues from HTML issue_count=0
```

If `issue_count=0`, parser didn't find tickets.

### Extension icon not appearing

1. Go to `chrome://extensions/`
2. Find "Aktis Jira Collector"
3. Ensure toggle is **ON** (blue)
4. Click "Reload" icon if needed

## Data Export

### JSON Export

Export all data as JSON:

```bash
curl http://localhost:8084/database/data > jira-export.json
```

### View Statistics

```bash
curl http://localhost:8084/status | python -m json.tool
```

Output:
```json
{
  "collector": {
    "running": true,
    "uptime": 3600.5,
    "error_count": 0
  },
  "stats": {
    "total_tickets": 150,
    "last_collection": "2025-09-30 14:45:23"
  }
}
```

## Advanced Usage

### Collecting Specific JQL Queries

1. In Jira, run a search/filter
2. URL will be: `https://[company].atlassian.net/issues/?jql=...`
3. Extension detects as `search` page type
4. Collect normally

### Collecting from Boards

1. Navigate to Scrum/Kanban board
2. URL: `https://[company].atlassian.net/jira/software/c/projects/[KEY]/boards/[ID]`
3. Extension detects as `board` page type
4. Collects visible tickets on board

### Database Location

Default: `C:\development\aktis\aktis-collector-jira\data\aktis-collector-jira.db`

**Backup database:**
```bash
copy data\aktis-collector-jira.db data\backup-2025-09-30.db
```

## Performance Tips

1. **Pagination**: Collect page-by-page for large projects
2. **Rate Limiting**: Wait 1-2 seconds between collections
3. **Browser Resources**: Close other tabs to free memory
4. **Server Resources**: Monitor CPU usage if collecting large datasets

## Keyboard Shortcuts

*Coming soon*

## FAQ

**Q: Can I collect from multiple Jira instances?**
A: Yes, just navigate to different Atlassian URLs. Data is stored together.

**Q: Is my data sent anywhere?**
A: No, all data stays local on your machine in the BoltDB database.

**Q: How much data can I collect?**
A: BoltDB can handle millions of records. Typical project: 1000 tickets = ~5MB.

**Q: Can I run multiple collectors at once?**
A: No, BoltDB locks the database file. Only one server instance at a time.

**Q: Does this work with Jira Server/Data Center?**
A: Currently optimized for Jira Cloud. May work with self-hosted instances if URLs similar.

## Support

For issues or questions:
1. Check server logs in `logs/aktis-collector-jira.{timestamp}.log`
2. Review browser console (F12 → Console)
3. Open issue at: https://github.com/ternarybob/aktis/issues