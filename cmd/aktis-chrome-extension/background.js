// Background service worker for Aktis Jira Collector Extension

// Default configuration
const DEFAULT_CONFIG = {
  serverUrl: 'http://localhost:8084',
  autoCollect: false,
  followLinks: false
};

// Initialize extension
chrome.runtime.onInstalled.addListener(() => {
  console.log('Aktis Jira Collector extension installed');

  // Set default configuration
  chrome.storage.sync.get(['config'], (result) => {
    if (!result.config) {
      chrome.storage.sync.set({ config: DEFAULT_CONFIG });
    }
  });
});

// Monitor navigation and page refreshes for automatic collection
// Using webNavigation API for more reliable detection of both navigation and refresh
chrome.webNavigation.onCompleted.addListener(
  async (details) => {
    // Only process main frame (not iframes)
    if (details.frameId !== 0) {
      console.log('Skipping iframe navigation');
      return;
    }

    const tabId = details.tabId;
    const url = details.url;

    console.log('[AUTO-COLLECT] Page loaded:', url);

    // Check if URL is a Jira page
    if (!isJiraURL(url)) {
      console.log('[AUTO-COLLECT] Not a Jira page, skipping');
      return;
    }

    console.log('[AUTO-COLLECT] Jira page detected');

    // Get config to check if auto-collect is enabled
    try {
      const result = await chrome.storage.sync.get(['config']);
      const config = result.config || DEFAULT_CONFIG;

      console.log('[AUTO-COLLECT] Config loaded, autoCollect =', config.autoCollect);

      if (config.autoCollect) {
        console.log('[AUTO-COLLECT] Auto-collect enabled, starting collection');

        // Small delay to ensure dynamic content is loaded
        setTimeout(async () => {
          try {
            console.log('[AUTO-COLLECT] Collecting from tab:', tabId);
            const response = await collectPageFromTab(tabId);
            console.log('[AUTO-COLLECT] Collection successful:', response);

            // Notify side panel if open
            chrome.runtime.sendMessage({
              type: 'AUTO_COLLECT_COMPLETE',
              url: url,
              success: true
            }).catch(() => console.log('[AUTO-COLLECT] No sidepanel listener'));
          } catch (error) {
            console.error('[AUTO-COLLECT] Collection failed:', error);

            // Notify side panel if open
            chrome.runtime.sendMessage({
              type: 'AUTO_COLLECT_COMPLETE',
              url: url,
              success: false,
              error: error.message
            }).catch(() => console.log('[AUTO-COLLECT] No sidepanel listener'));
          }
        }, 1500);
      } else {
        console.log('[AUTO-COLLECT] Auto-collect disabled, skipping');
      }
    } catch (error) {
      console.error('[AUTO-COLLECT] Error loading config:', error);
    }
  },
  {
    url: [
      { hostContains: 'atlassian.net' },
      { hostContains: 'jira.com' },
      { urlContains: '/jira/' }
    ]
  }
);

// Also listen for SPA navigation (when URL changes without page reload, like clicking on an issue)
chrome.webNavigation.onHistoryStateUpdated.addListener(
  async (details) => {
    // Only process main frame (not iframes)
    if (details.frameId !== 0) {
      return;
    }

    const tabId = details.tabId;
    const url = details.url;

    console.log('[AUTO-COLLECT] SPA navigation detected:', url);

    // Check if URL is a Jira page
    if (!isJiraURL(url)) {
      return;
    }

    console.log('[AUTO-COLLECT] Jira SPA navigation detected');

    // Get config to check if auto-collect is enabled
    try {
      const result = await chrome.storage.sync.get(['config']);
      const config = result.config || DEFAULT_CONFIG;

      if (config.autoCollect) {
        console.log('[AUTO-COLLECT] Auto-collect enabled for SPA navigation, starting collection');

        // Small delay to ensure dynamic content is loaded
        setTimeout(async () => {
          try {
            console.log('[AUTO-COLLECT] Collecting from tab:', tabId);
            const response = await collectPageFromTab(tabId);
            console.log('[AUTO-COLLECT] SPA collection successful:', response);

            // Notify side panel if open
            chrome.runtime.sendMessage({
              type: 'AUTO_COLLECT_COMPLETE',
              url: url,
              success: true
            }).catch(() => console.log('[AUTO-COLLECT] No sidepanel listener'));
          } catch (error) {
            console.error('[AUTO-COLLECT] SPA collection failed:', error);

            // Notify side panel if open
            chrome.runtime.sendMessage({
              type: 'AUTO_COLLECT_COMPLETE',
              url: url,
              success: false,
              error: error.message
            }).catch(() => console.log('[AUTO-COLLECT] No sidepanel listener'));
          }
        }, 1500);
      }
    } catch (error) {
      console.error('[AUTO-COLLECT] Error loading config for SPA navigation:', error);
    }
  },
  {
    url: [
      { hostContains: 'atlassian.net' },
      { hostContains: 'jira.com' },
      { urlContains: '/jira/' }
    ]
  }
);

// Helper to check if URL is a Jira page
function isJiraURL(url) {
  return url.includes('.atlassian.net') ||
         url.includes('/jira/') ||
         url.includes('/browse/') ||
         url.includes('/projects/');
}

// Listen for messages from content script and popup
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  if (request.type === 'PAGE_DATA') {
    handlePageData(request.data, sender.tab)
      .then(response => sendResponse({ success: true, response }))
      .catch(error => sendResponse({ success: false, error: error.message }));
    return true; // Keep channel open for async response
  }

  if (request.type === 'COLLECT_CURRENT_PAGE') {
    // Handle manual collection from popup
    collectPageFromTab(request.tabId)
      .then(response => sendResponse({ success: true, response }))
      .catch(error => sendResponse({ success: false, error: error.message }));
    return true;
  }

  if (request.type === 'GET_CONFIG') {
    chrome.storage.sync.get(['config'], (result) => {
      sendResponse({ config: result.config || DEFAULT_CONFIG });
    });
    return true;
  }

  if (request.type === 'UPDATE_CONFIG') {
    chrome.storage.sync.set({ config: request.config }, () => {
      sendResponse({ success: true });
    });
    return true;
  }

  if (request.type === 'COLLECT_PROJECT_TICKETS') {
    // Navigate to project URL and collect tickets
    navigateAndCollect(request.projectUrl, request.projectKey)
      .then(response => sendResponse({ success: true, response }))
      .catch(error => sendResponse({ success: false, error: error.message }));
    return true;
  }
});

// Handle page data from content script
async function handlePageData(pageData, tab) {
  // Get server URL from config
  const config = await new Promise((resolve) => {
    chrome.storage.sync.get(['config'], (result) => {
      resolve(result.config || DEFAULT_CONFIG);
    });
  });

  const serverUrl = `${config.serverUrl}/receiver`;

  // Add metadata
  const payload = {
    timestamp: new Date().toISOString(),
    url: tab.url,
    title: tab.title,
    data: pageData,
    collector: {
      name: 'aktis-jira-collector-extension',
      version: chrome.runtime.getManifest().version
    }
  };

  console.log('Sending page data to server:', serverUrl);

  // Send to server
  try {
    const response = await fetch(serverUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(payload)
    });

    if (!response.ok) {
      throw new Error(`Server returned ${response.status}: ${response.statusText}`);
    }

    const result = await response.json();
    console.log('Server response:', result);

    return result;
  } catch (error) {
    console.error('Failed to send data to server:', error);
    throw error;
  }
}

// Collect page data from a specific tab
async function collectPageFromTab(tabId) {
  // Get tab info
  const tab = await chrome.tabs.get(tabId);
  console.log('Collecting from tab:', tabId, 'URL:', tab.url);

  // Verify this is a valid Jira URL
  if (!tab.url || tab.url.startsWith('chrome://') || tab.url.startsWith('chrome-extension://')) {
    throw new Error(`Cannot collect from invalid URL: ${tab.url}`);
  }

  // Get page HTML by executing script in the tab
  const results = await chrome.scripting.executeScript({
    target: { tabId: tabId },
    func: () => {
      return {
        html: document.documentElement.outerHTML,
        url: window.location.href,
        title: document.title
      };
    }
  });

  if (!results || results.length === 0) {
    throw new Error('Failed to collect page data');
  }

  const pageData = results[0].result;

  // Create payload
  const config = await new Promise((resolve) => {
    chrome.storage.sync.get(['config'], (result) => {
      resolve(result.config || DEFAULT_CONFIG);
    });
  });

  const serverUrl = `${config.serverUrl}/receiver`;

  const payload = {
    timestamp: new Date().toISOString(),
    url: pageData.url,
    title: pageData.title,
    data: {
      pageType: 'generic',
      html: pageData.html,
      url: pageData.url
    },
    collector: {
      name: 'aktis-jira-collector-extension',
      version: chrome.runtime.getManifest().version
    }
  };

  console.log('Sending page data to server:', serverUrl);

  // Send to server
  const response = await fetch(serverUrl, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(payload)
  });

  if (!response.ok) {
    throw new Error(`Server returned ${response.status}: ${response.statusText}`);
  }

  const result = await response.json();
  console.log('Server response:', result);

  return result;
}

// Navigate to a URL and automatically collect the page
async function navigateAndCollect(projectUrl, projectKey) {
  // Construct issue list URL from project URL
  let issueListUrl = projectUrl;
  if (!issueListUrl.includes('/issues')) {
    issueListUrl = issueListUrl.replace(/\/projects\/([^\/]+)/, '/jira/software/c/projects/$1/issues');
  }

  // Always create a new tab to avoid any context confusion
  const targetTab = await chrome.tabs.create({
    url: issueListUrl,
    active: true
  });

  console.log('Created new tab:', targetTab.id, 'URL:', issueListUrl);

  // Wait for the tab to finish loading
  await new Promise((resolve) => {
    const listener = (tabId, changeInfo, tab) => {
      if (tabId === targetTab.id && changeInfo.status === 'complete') {
        console.log('Tab loaded:', tabId, 'URL:', tab.url);
        chrome.tabs.onUpdated.removeListener(listener);
        resolve();
      }
    };
    chrome.tabs.onUpdated.addListener(listener);
  });

  // Wait a bit more for dynamic content to load
  await new Promise(resolve => setTimeout(resolve, 2000));

  console.log('About to collect from tab:', targetTab.id);

  // Now collect the page
  const result = await collectPageFromTab(targetTab.id);
  return result;
}

// Handle browser action click - open side panel
chrome.action.onClicked.addListener((tab) => {
  chrome.sidePanel.open({ windowId: tab.windowId });
});
// Trigger initial assessment when extension loads (on browser startup)
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

// Also trigger on extension installation/update
chrome.runtime.onInstalled.addListener(async () => {
  console.log('[AUTO-COLLECT] Extension installed/updated, checking current tab...');
  const tabs = await chrome.tabs.query({ active: true, currentWindow: true });
  if (tabs.length > 0) {
    const tab = tabs[0];
    if (isJiraURL(tab.url)) {
      const result = await chrome.storage.sync.get(['config']);
      const config = result.config || DEFAULT_CONFIG;
      if (config.autoCollect) {
        console.log('[AUTO-COLLECT] Auto-collect enabled, processing current page');
        setTimeout(async () => {
          await collectPageFromTab(tab.id);
        }, 2000);
      }
    }
  }
});
