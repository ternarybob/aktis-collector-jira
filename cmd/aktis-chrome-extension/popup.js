// Popup script for Aktis Jira Collector

// Load saved configuration on popup open
document.addEventListener('DOMContentLoaded', () => {
  chrome.runtime.sendMessage({ type: 'GET_CONFIG' }, (response) => {
    const config = response.config;
    document.getElementById('serverUrl').value = config.serverUrl;
    document.getElementById('autoCollect').checked = config.autoCollect;
    document.getElementById('followLinks').checked = config.followLinks;
  });
});

// Collect current page button
document.getElementById('collectBtn').addEventListener('click', async () => {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });

  try {
    // Send to background script instead of content script
    const response = await chrome.runtime.sendMessage({
      type: 'COLLECT_CURRENT_PAGE',
      tabId: tab.id
    });

    if (response.success) {
      showStatus('Page data collected successfully', 'success');
    } else {
      showStatus(`Failed: ${response.error}`, 'error');
    }
  } catch (error) {
    showStatus(`Error: ${error.message}`, 'error');
  }
});

// Save settings button
document.getElementById('saveBtn').addEventListener('click', () => {
  const config = {
    serverUrl: document.getElementById('serverUrl').value,
    autoCollect: document.getElementById('autoCollect').checked,
    followLinks: document.getElementById('followLinks').checked
  };

  chrome.runtime.sendMessage({ type: 'UPDATE_CONFIG', config }, (response) => {
    if (response.success) {
      showStatus('Settings saved', 'success');
    } else {
      showStatus('Failed to save settings', 'error');
    }
  });
});

// Show status message
function showStatus(message, type) {
  const statusEl = document.getElementById('status');
  statusEl.textContent = message;
  statusEl.className = `status ${type}`;

  setTimeout(() => {
    statusEl.className = 'status';
  }, 3000);
}