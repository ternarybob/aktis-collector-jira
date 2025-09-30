// Background service worker for Aktis Jira Collector Extension

// Default configuration
const DEFAULT_CONFIG = {
  serverUrl: 'http://localhost:8080',
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

// Handle browser action click
chrome.action.onClicked.addListener((tab) => {
  // Send message to content script to collect current page
  chrome.tabs.sendMessage(tab.id, { type: 'COLLECT_PAGE' });
});