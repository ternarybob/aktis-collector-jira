// Sidepanel JavaScript for Aktis Jira Collector
console.log('Aktis Jira Collector sidepanel loaded');

let config = {
  serverUrl: 'http://localhost:8084',
  autoCollect: false,
  autoNavigate: false
};

// Load config on startup
loadConfig();
loadLastCollection();

// Listen for auto-collection completion messages from background script
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === 'AUTO_COLLECT_COMPLETE') {
    if (message.success) {
      updateLastCollection();
      console.log('Auto-collection completed for:', message.url);
    } else {
      console.log('Auto-collection failed:', message.error);
    }
  }
});

// Tab switching
document.querySelectorAll('.tab').forEach(tab => {
  tab.addEventListener('click', () => {
    const tabName = tab.getAttribute('data-tab');
    switchTab(tabName);
  });
});

function switchTab(tabName) {
  // Update tab buttons
  document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
  document.querySelector(`[data-tab="${tabName}"]`).classList.add('active');

  // Update tab content
  document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
  document.getElementById(`${tabName}-tab`).classList.add('active');

  // Load data for the active tab
  if (tabName === 'buffer') {
    loadBufferData();
  } else if (tabName === 'collect') {
    checkServerStatus();
    detectPageType();
  }
}

// Load configuration from storage
function loadConfig() {
  chrome.storage.sync.get(['config'], (result) => {
    if (result.config) {
      config = result.config;
      document.getElementById('server-url').value = config.serverUrl || 'http://localhost:8084';
      document.getElementById('auto-collect').checked = config.autoCollect || false;
    }
    checkServerStatus();
  });
}

// Save settings
document.getElementById('save-settings-btn').addEventListener('click', async () => {
  const wasAutoCollectEnabled = config.autoCollect;
  const isAutoCollectEnabled = document.getElementById('auto-collect').checked;

  config.serverUrl = document.getElementById('server-url').value;
  config.autoCollect = isAutoCollectEnabled;

  chrome.storage.sync.set({ config }, async () => {
    showMessage('settings-message', 'Settings saved successfully!', 'success');

    // If auto-collect was just enabled, immediately collect current page
    if (!wasAutoCollectEnabled && isAutoCollectEnabled) {
      try {
        const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });

        // Check if it's a Jira page
        if (tab.url && (tab.url.includes('.atlassian.net') || tab.url.includes('/jira/'))) {
          showMessage('settings-message', 'Auto-collect enabled. Collecting current page...', 'success');

          // Trigger collection via background script
          const response = await chrome.runtime.sendMessage({
            type: 'COLLECT_CURRENT_PAGE',
            tabId: tab.id
          });

          if (response && response.success) {
            showMessage('settings-message', 'Settings saved and current page collected!', 'success');
          }
        }
      } catch (error) {
        console.error('Failed to collect current page:', error);
      }
    }

    setTimeout(() => clearMessage('settings-message'), 3000);
  });
});

// Load last collection timestamp from storage
function loadLastCollection() {
  chrome.storage.local.get(['lastCollection'], (result) => {
    if (result.lastCollection) {
      document.getElementById('last-collection').textContent = result.lastCollection;
    } else {
      document.getElementById('last-collection').textContent = 'Never';
    }
  });
}

// Update last collection timestamp and refresh page info
async function updateLastCollection() {
  const timestamp = new Date().toLocaleTimeString();
  document.getElementById('last-collection').textContent = timestamp;

  // Persist to storage
  chrome.storage.local.set({ lastCollection: timestamp });

  // Re-detect page type to update confidence
  await detectPageType();
}

// Check server status
async function checkServerStatus() {
  const statusEl = document.getElementById('server-status');
  try {
    console.log('Checking server status at:', config.serverUrl + '/health');
    const response = await fetch(`${config.serverUrl}/health`);
    console.log('Server response status:', response.status);
    if (response.ok) {
      const data = await response.json();
      console.log('Server health data:', data);
      statusEl.textContent = 'Online';
      statusEl.className = 'status-value online';
    } else {
      console.warn('Server returned non-OK status:', response.status);
      statusEl.textContent = 'Offline';
      statusEl.className = 'status-value offline';
    }
  } catch (error) {
    console.error('Failed to check server status:', error);
    statusEl.textContent = 'Offline';
    statusEl.className = 'status-value offline';
  }
}


// Detect current page type using server assessment
async function detectPageType() {
  // Show processing status
  document.getElementById('page-type').textContent = 'assessing...';
  document.getElementById('confidence').textContent = 'processing...';
  document.getElementById('confidence').className = 'status-value';

  try {
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });

    // Get page HTML
    const results = await chrome.scripting.executeScript({
      target: { tabId: tab.id },
      func: () => {
        return {
          html: document.documentElement.outerHTML,
          url: window.location.href
        };
      }
    });

    const pageData = results[0].result;

    // Send to server for assessment
    const response = await fetch(`${config.serverUrl}/assess`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        url: pageData.url,
        html: pageData.html
      })
    });

    if (!response.ok) {
      throw new Error(`Server returned ${response.status}`);
    }

    const result = await response.json();
    const assessment = result.assessment;

    // Update UI with assessment results
    document.getElementById('page-type').textContent = assessment.page_type;

    // Update confidence display
    const confidenceEl = document.getElementById('confidence');
    confidenceEl.textContent = assessment.confidence;
    confidenceEl.className = `status-value confidence-${assessment.confidence}`;

  } catch (error) {
    console.error('Failed to detect page type:', error);
    document.getElementById('page-type').textContent = 'error';
    document.getElementById('confidence').textContent = '-';
  }
}




// Load buffer data (projects) from server
async function loadBufferData() {
  const bufferContent = document.getElementById('buffer-content');
  bufferContent.innerHTML = '<div class="loading">Loading projects...</div>';

  try {
    const response = await fetch(`${config.serverUrl}/projects`);

    if (!response.ok) {
      throw new Error(`Server returned ${response.status}`);
    }

    const result = await response.json();
    displayProjects(result.projects || []);
  } catch (error) {
    bufferContent.innerHTML = `<div class="error">Failed to load projects: ${error.message}</div>`;
  }
}

// Display projects in buffer
function displayProjects(projects) {
  const bufferContent = document.getElementById('buffer-content');
  const bufferCount = document.getElementById('buffer-count');

  if (!projects || projects.length === 0) {
    bufferContent.innerHTML = '<div class="loading">No projects found. Collect a projects list page first.</div>';
    bufferCount.textContent = '0 projects';
    return;
  }

  bufferCount.textContent = `${projects.length} project${projects.length !== 1 ? 's' : ''}`;

  let html = '';
  for (const project of projects) {
    const firstLetter = (project.name || project.key || 'U')[0].toUpperCase();
    const iconColor = getProjectColor(project.key);

    html += `
      <div class="project-card" data-project-key="${project.key}">
        <div class="project-header">
          <div class="project-info">
            <div class="project-name">${project.name || 'Unnamed Project'}</div>
            <div class="project-type">${project.type || 'Unknown Type'}</div>
          </div>
          <div class="project-icon" style="background-color: ${iconColor};">${firstLetter}</div>
        </div>

        <div class="project-stats">
          <div class="stat-item">
            <span class="stat-label">Tickets:</span>
            <span id="ticket-count-${project.key}">${project.ticket_count || 0}</span>
          </div>
        </div>

        <div class="project-actions">
          <button class="button collect-tickets-btn" data-project-key="${project.key}" data-project-url="${escapeHtml(project.url)}">
            Collect Tickets
          </button>
        </div>

        <div id="collection-status-${project.key}" style="margin-top: 8px;"></div>
      </div>
    `;
  }

  bufferContent.innerHTML = html;

  // Add event listeners to all collect buttons
  const collectButtons = document.querySelectorAll('.collect-tickets-btn');
  collectButtons.forEach(button => {
    button.addEventListener('click', function() {
      const projectKey = this.getAttribute('data-project-key');
      const projectUrl = this.getAttribute('data-project-url');
      collectProjectTickets(projectKey, projectUrl);
    });
  });
}

// Generate consistent color for project based on key
function getProjectColor(projectKey) {
  const colors = [
    '#0052CC', // Blue
    '#00875A', // Green
    '#FF8B00', // Orange
    '#6554C0', // Purple
    '#FF5630', // Red
    '#00B8D9', // Cyan
    '#36B37E', // Teal
    '#FFAB00', // Yellow
    '#403294', // Dark Purple
    '#E34935', // Dark Red
  ];

  // Generate consistent index from project key
  let hash = 0;
  for (let i = 0; i < projectKey.length; i++) {
    hash = projectKey.charCodeAt(i) + ((hash << 5) - hash);
  }
  const index = Math.abs(hash) % colors.length;
  return colors[index];
}

// Escape HTML to prevent XSS
function escapeHtml(text) {
  if (!text) return '';
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

// Collect tickets for a specific project
async function collectProjectTickets(projectKey, projectUrl) {
  const statusEl = document.getElementById(`collection-status-${projectKey}`);

  if (!projectUrl) {
    statusEl.innerHTML = `<span class="error">No project URL available</span>`;
    return;
  }

  // Construct issue list URL
  let issueListUrl = projectUrl;
  if (!issueListUrl.includes('/issues')) {
    issueListUrl = issueListUrl.replace(/\/projects\/([^\/]+)/, '/jira/software/c/projects/$1/issues');
  }

  statusEl.innerHTML = `<span class="collection-status status-pending">Navigate to: <a href="${issueListUrl}" target="_blank" style="color: #0052CC; text-decoration: underline;">${projectKey} issues</a> and click "Collect Current Page"</span>`;
}

// Clear buffer data
document.getElementById('clear-buffer-btn').addEventListener('click', async () => {
  if (!confirm('Are you sure you want to clear all data? This will delete all projects and tickets permanently.')) {
    return;
  }

  try {
    const response = await fetch(`${config.serverUrl}/database`, {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' }
    });

    if (response.ok) {
      showMessage('settings-message', 'All data cleared successfully!', 'success');

      // Clear the buffer display if user is on buffer tab
      if (document.querySelector('[data-tab="buffer"]').classList.contains('active')) {
        const bufferContent = document.getElementById('buffer-content');
        bufferContent.innerHTML = '<div class="loading">No projects found. Collect a projects list page first.</div>';
        document.getElementById('buffer-count').textContent = '0 projects';
      }
    } else {
      const result = await response.json();
      throw new Error(result.message || 'Failed to clear data');
    }
  } catch (error) {
    showMessage('settings-message', 'Failed to clear data: ' + error.message, 'error');
  }

  setTimeout(() => clearMessage('settings-message'), 5000);
});

// Helper: Show message
function showMessage(elementId, message, type) {
  const el = document.getElementById(elementId);
  el.innerHTML = `<div class="${type}" style="margin-top: 12px;">${message}</div>`;
}

// Helper: Clear message
function clearMessage(elementId) {
  document.getElementById(elementId).innerHTML = '';
}

// Auto-refresh server status every 30 seconds
setInterval(() => {
  if (document.querySelector('[data-tab="collect"]').classList.contains('active')) {
    checkServerStatus();
  }
}, 30000);

// Auto-refresh buffer data every 30 seconds if buffer tab is active (reduced frequency)
// Only update ticket counts, don't re-render entire UI to prevent flashing
setInterval(() => {
  if (document.querySelector('[data-tab="buffer"]').classList.contains('active')) {
    refreshProjectTicketCounts();
  }
}, 30000);

// Refresh only ticket counts without re-rendering entire UI
async function refreshProjectTicketCounts() {
  try {
    const response = await fetch(`${config.serverUrl}/projects`);
    if (!response.ok) return;

    const result = await response.json();
    const projects = result.projects || [];

    // Update ticket counts in place
    for (const project of projects) {
      const countEl = document.getElementById(`ticket-count-${project.key}`);
      if (countEl) {
        countEl.textContent = project.ticket_count || 0;
      }
    }

    // Update total count
    const bufferCount = document.getElementById('buffer-count');
    if (bufferCount) {
      bufferCount.textContent = `${projects.length} project${projects.length !== 1 ? 's' : ''}`;
    }
  } catch (error) {
    // Silently fail - don't disrupt user experience
    console.error('Failed to refresh ticket counts:', error);
  }
}
// Wire up "Process Page" button in header
document.getElementById('process-page-btn').addEventListener('click', async () => {
  const btn = document.getElementById('process-page-btn');
  btn.disabled = true;
  btn.textContent = 'Processing...';

  try {
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });

    // Use background script to collect
    const response = await chrome.runtime.sendMessage({
      type: 'COLLECT_CURRENT_PAGE',
      tabId: tab.id
    });

    if (response && response.success) {
      btn.textContent = '✓ Processed';
      await updateLastCollection();
      await loadCounts();
      setTimeout(() => {
        btn.textContent = 'Process Page';
        btn.disabled = false;
      }, 2000);
    } else {
      throw new Error(response.error || 'Processing failed');
    }
  } catch (error) {
    btn.textContent = 'Error';
    console.error('Process error:', error);
    setTimeout(() => {
      btn.textContent = 'Process Page';
      btn.disabled = false;
    }, 2000);
  }
});

// Load last collection timestamp from server
async function loadLastCollectionFromServer() {
  try {
    const response = await fetch(`${config.serverUrl}/status`);
    if (response.ok) {
      const data = await response.json();
      if (data.stats && data.stats.last_collection && data.stats.last_collection !== "Never") {
        document.getElementById('last-collection').textContent = data.stats.last_collection;
      } else {
        document.getElementById('last-collection').textContent = 'Never';
      }
    }
  } catch (error) {
    console.error('Failed to load last collection from server:', error);
    document.getElementById('last-collection').textContent = 'Unknown';
  }
}

// Load projects and tickets counts
async function loadCounts() {
  try {
    // Load projects
    const projectsResp = await fetch(`${config.serverUrl}/projects`);
    if (projectsResp.ok) {
      const projectsData = await projectsResp.json();
      const projectCount = projectsData.count || (projectsData.projects ? projectsData.projects.length : 0);
      document.getElementById('projects-count').textContent = projectCount;
    }

    // Load tickets
    const ticketsResp = await fetch(`${config.serverUrl}/database`);
    if (ticketsResp.ok) {
      const data = await ticketsResp.json();
      document.getElementById('tickets-count').textContent = data.count || 0;
    }
  } catch (error) {
    console.error('Failed to load counts:', error);
  }
}

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

// Initialize on load
loadLastCollectionFromServer();
loadCounts();

// Refresh counts when switching to collect tab
document.querySelectorAll('.tab').forEach(tab => {
  tab.addEventListener('click', () => {
    const tabName = tab.getAttribute('data-tab');
    if (tabName === 'collect') {
      loadLastCollectionFromServer();
      loadCounts();
    }
  });
});
