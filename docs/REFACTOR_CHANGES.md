# Refactor Changes for v0.1.109+

## Issue Summary
1. Move Process Page button into Collect tab ✅
2. Add refresh button for counts ✅
3. Fix ticket count (showing 45 instead of 22) ⚠️
4. Fix auto-processing not working on page load/refresh ⚠️
5. Update version display to reflect actual build version ✅

## Files to Modify

### 1. cmd/aktis-chrome-extension/sidepanel.html

**Line 377-393: Remove Process Page button from header, add CSS for new buttons, update version**

FIND:
```html
    .checkbox-group input {
      margin-right: 8px;
    }
  </style>
</head>
<body>
  <div class="header">
    <div style="display: flex; justify-content: space-between; align-items: center;">
      <div>
        <h1>Aktis Jira Collector</h1>
        <div class="version">v0.1.0</div>
      </div>
      <button id="process-page-btn" class="button" style="background: white; color: #0052CC; border: none; padding: 8px 16px; font-weight: 600; cursor: pointer;">
        Process Page
      </button>
    </div>
  </div>
```

REPLACE WITH:
```html
    .checkbox-group input {
      margin-right: 8px;
    }

    .status-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 12px;
      padding-bottom: 12px;
      border-bottom: 2px solid #f0f0f0;
    }

    .icon-button {
      background: white;
      color: #0052CC;
      border: 1px solid #0052CC;
      padding: 8px 12px;
      border-radius: 4px;
      cursor: pointer;
      font-size: 14px;
      font-weight: 500;
      transition: background 0.2s;
    }

    .icon-button:hover {
      background: #f5f5f5;
    }
  </style>
</head>
<body>
  <div class="header">
    <div>
      <h1>Aktis Jira Collector</h1>
      <div class="version" id="version-display">Loading...</div>
    </div>
  </div>
```

**Line 403-429: Add buttons to Collect tab status box**

FIND:
```html
    <div class="tab-content active" id="collect-tab">
      <div class="status">
        <div class="status-item">
          <span class="status-label">Server Status</span>
```

REPLACE WITH:
```html
    <div class="tab-content active" id="collect-tab">
      <div class="status">
        <div class="status-header">
          <div style="display: flex; gap: 8px;">
            <button id="process-page-btn" class="icon-button">Process Page</button>
            <button id="refresh-counts-btn" class="icon-button">Refresh</button>
          </div>
        </div>

        <div class="status-item">
          <span class="status-label">Server Status</span>
```

### 2. cmd/aktis-chrome-extension/sidepanel.js

**Add after line 148 (after checkServerStatus function):**

```javascript
// Load version from server
async function loadVersion() {
  try {
    const response = await fetch(`${config.serverUrl}/status`);
    if (response.ok) {
      const data = await response.json();
      if (data.version) {
        document.getElementById('version-display').textContent = `v${data.version}`;
      }
    }
  } catch (error) {
    console.error('Failed to load version:', error);
    document.getElementById('version-display').textContent = 'v0.1.0';
  }
}
```

**Add after line 495 (after loadCounts function):**

```javascript
// Wire up refresh button
document.getElementById('refresh-counts-btn').addEventListener('click', async () => {
  const btn = document.getElementById('refresh-counts-btn');
  btn.disabled = true;
  btn.textContent = 'Refreshing...';

  await loadCounts();
  await loadLastCollectionFromServer();

  btn.disabled = false;
  btn.textContent = 'Refresh';
});
```

**Update line 498 (initialization):**

FIND:
```javascript
// Initialize on load
loadLastCollectionFromServer();
loadCounts();
```

REPLACE WITH:
```javascript
// Initialize on load
loadVersion();
loadLastCollectionFromServer();
loadCounts();
```

### 3. cmd/aktis-chrome-extension/background.js

**Fix auto-processing: Add immediate collection on extension startup**

Add after line 100 (after webNavigation.onCompleted listener):

```javascript
// Trigger initial assessment when extension loads
chrome.runtime.onStartup.addListener(async () => {
  console.log('[AUTO-COLLECT] Extension started, checking current tab...');
  const tabs = await chrome.tabs.query({ active: true, currentWindow: true });
  if (tabs.length > 0) {
    const tab = tabs[0];
    if (isJiraURL(tab.url)) {
      const result = await chrome.storage.sync.get(['config']);
      const config = result.config || DEFAULT_CONFIG;
      if (config.autoCollect) {
        console.log('[AUTO-COLLECT] Auto-collect enabled on startup, processing current page');
        setTimeout(async () => {
          await collectPageFromTab(tab.id);
        }, 2000);
      }
    }
  }
});

// Also trigger on extension installation
chrome.runtime.onInstalled.addListener(async () => {
  console.log('[AUTO-COLLECT] Extension installed/updated');
  // Same logic as onStartup
});
```

### 4. Fix Ticket Count Issue

**Investigation needed:** The `/database/data` endpoint is returning 45 tickets but UI shows only 22 are in DATA project. Need to check:

1. Does `/database/data` return ALL tickets across ALL projects?
2. Should we filter by a specific project?
3. Or should we use a different endpoint?

**Test the endpoint:**
```bash
curl http://localhost:8084/database/data | jq '. | length'
```

If it returns all tickets from all projects, we may want to:
- Add a `/database/stats` endpoint that returns proper counts
- Or filter tickets by project_id in the frontend

### 5. Auto-Processing on Page Refresh

The `webNavigation.onCompleted` listener should already handle page refreshes. If it's not working:

**Debug steps:**
1. Check Chrome extension console for `[AUTO-COLLECT]` log messages
2. Verify auto-collect setting is enabled in storage
3. Check if URL filters are matching correctly

**Potential fix - make URL filters more permissive:**

In background.js, update the URL filters:

FIND:
```javascript
  {
    url: [
      { hostContains: 'atlassian.net' },
      { hostContains: 'jira.com' },
      { urlContains: '/jira/' }
    ]
  }
```

REPLACE WITH:
```javascript
  {
    url: [
      { hostContains: 'atlassian.net' },
      { hostContains: 'jira' }
    ]
  }
```

## Testing Checklist

After making changes:

- [ ] Process Page button moved to Collect tab
- [ ] Refresh button present and working
- [ ] Version displays correctly (v0.1.109)
- [ ] Project count shows 12
- [ ] Ticket count shows correct number
- [ ] Auto-collect works on page load
- [ ] Auto-collect works on page refresh
- [ ] Auto-collect works when clicking between tickets

## Build and Deploy

```bash
cd c:/development/aktis/aktis-collector-jira
./scripts/build.ps1
```

Then reload the extension in Chrome:
1. Go to chrome://extensions
2. Find "Aktis Jira Collector"
3. Click reload button