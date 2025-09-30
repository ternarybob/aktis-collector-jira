// Content script for Aktis Jira Collector
// Runs on Jira pages to extract ticket data

console.log('Aktis Jira Collector content script loaded');

// Listen for messages from background script or popup
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  if (request.type === 'COLLECT_PAGE') {
    collectCurrentPage()
      .then(data => sendResponse({ success: true, data }))
      .catch(error => sendResponse({ success: false, error: error.message }));
    return true; // Keep channel open for async
  }
});

// Auto-collect on page load if enabled
chrome.runtime.sendMessage({ type: 'GET_CONFIG' }, (response) => {
  if (response.config && response.config.autoCollect) {
    // Wait for page to be fully loaded
    if (document.readyState === 'complete') {
      collectCurrentPage();
    } else {
      window.addEventListener('load', () => collectCurrentPage());
    }
  }
});

// Collect data from current page
async function collectCurrentPage() {
  console.log('Collecting page data...');

  const pageType = detectPageType();
  console.log('Page type detected:', pageType);

  let pageData = {
    pageType: pageType,
    url: window.location.href,
    timestamp: new Date().toISOString(),
    html: document.documentElement.outerHTML
  };

  // Collect structured data based on page type
  if (pageType === 'issue') {
    pageData.issue = extractIssueData();
  } else if (pageType === 'board') {
    pageData.board = extractBoardData();
  } else if (pageType === 'search') {
    pageData.search = extractSearchResults();
  }

  // Send to background script
  const response = await chrome.runtime.sendMessage({
    type: 'PAGE_DATA',
    data: pageData
  });

  console.log('Data sent to background:', response);

  // Show notification
  showNotification('Page data collected', pageType);

  return pageData;
}

// Detect what type of Jira page we're on
function detectPageType() {
  const url = window.location.href;
  const path = window.location.pathname;

  if (url.includes('/browse/') || path.includes('/browse/')) {
    return 'issue';
  }
  if (url.includes('/board/') || path.includes('/secure/RapidBoard')) {
    return 'board';
  }
  if (url.includes('/issues/') || url.includes('/jira/software/c/projects')) {
    return 'search';
  }
  if (url.includes('/projects/')) {
    return 'project';
  }

  return 'unknown';
}

// Extract data from issue page
function extractIssueData() {
  const data = {
    key: null,
    summary: null,
    description: null,
    issueType: null,
    status: null,
    priority: null,
    assignee: null,
    reporter: null,
    created: null,
    updated: null,
    labels: [],
    components: [],
    customFields: {}
  };

  // Try to extract issue key from URL or page
  const urlMatch = window.location.href.match(/\/browse\/([A-Z]+-\d+)/);
  if (urlMatch) {
    data.key = urlMatch[1];
  }

  // Try different selectors for different Jira versions
  // Summary
  const summarySelectors = [
    '[data-test-id="issue.views.field.rich-text.summary"]',
    '[data-testid="issue.views.issue-base.foundation.summary.heading"]',
    '#summary-val',
    'h1[data-test-id*="summary"]',
    'h1.summary'
  ];
  for (const selector of summarySelectors) {
    const element = document.querySelector(selector);
    if (element) {
      data.summary = element.textContent.trim();
      break;
    }
  }

  // Description
  const descSelectors = [
    '[data-test-id="issue.views.field.rich-text.description"]',
    '#description-val',
    '.description'
  ];
  for (const selector of descSelectors) {
    const element = document.querySelector(selector);
    if (element) {
      data.description = element.textContent.trim();
      break;
    }
  }

  // Issue Type
  const typeSelectors = [
    '[data-test-id="issue.views.field.issue-type.common.ui.read-view.value"]',
    '#type-val',
    '[data-testid*="issue-type"]'
  ];
  for (const selector of typeSelectors) {
    const element = document.querySelector(selector);
    if (element) {
      data.issueType = element.textContent.trim();
      break;
    }
  }

  // Status
  const statusSelectors = [
    '[data-test-id="issue.views.field.status.common.ui.read-view.status-button"]',
    '#status-val span',
    '[data-testid*="status"]'
  ];
  for (const selector of statusSelectors) {
    const element = document.querySelector(selector);
    if (element) {
      data.status = element.textContent.trim();
      break;
    }
  }

  // Priority
  const prioritySelectors = [
    '[data-test-id="issue.views.field.priority.common.ui.read-view.priority"]',
    '#priority-val',
    '[data-testid*="priority"]'
  ];
  for (const selector of prioritySelectors) {
    const element = document.querySelector(selector);
    if (element) {
      data.priority = element.textContent.trim();
      break;
    }
  }

  // Labels
  const labelElements = document.querySelectorAll('[data-test-id*="label"], .labels a, #wrap-labels a');
  data.labels = Array.from(labelElements).map(el => el.textContent.trim()).filter(Boolean);

  // Components
  const componentElements = document.querySelectorAll('[data-test-id*="component"], #components-val a');
  data.components = Array.from(componentElements).map(el => el.textContent.trim()).filter(Boolean);

  return data;
}

// Extract data from board/backlog page
function extractBoardData() {
  const data = {
    boardName: null,
    columns: [],
    issues: []
  };

  // Board name
  const boardNameElement = document.querySelector('[data-test-id="navigation-apps.ui.board-name"]');
  if (boardNameElement) {
    data.boardName = boardNameElement.textContent.trim();
  }

  // Extract visible issues from board
  const issueCards = document.querySelectorAll('[data-testid*="issue"], .ghx-issue');
  data.issues = Array.from(issueCards).map(card => {
    const key = card.querySelector('[data-testid*="issue-key"], .ghx-key')?.textContent.trim();
    const summary = card.querySelector('[data-testid*="summary"], .ghx-summary')?.textContent.trim();
    return { key, summary };
  }).filter(issue => issue.key);

  return data;
}

// Extract search results
function extractSearchResults() {
  const data = {
    query: null,
    count: 0,
    issues: []
  };

  // Try to get search query
  const searchInput = document.querySelector('input[name="jql"], [data-test-id*="search"]');
  if (searchInput) {
    data.query = searchInput.value;
  }

  // Extract issue list
  const issueRows = document.querySelectorAll('[data-testid*="issue-row"], .issue-list tr[data-issue-key]');
  data.issues = Array.from(issueRows).map(row => {
    const key = row.getAttribute('data-issue-key') ||
                row.querySelector('[data-testid*="issue-key"]')?.textContent.trim();
    const summary = row.querySelector('[data-testid*="summary"], .summary')?.textContent.trim();
    return { key, summary };
  }).filter(issue => issue.key);

  data.count = data.issues.length;

  return data;
}

// Show in-page notification
function showNotification(message, detail) {
  const notification = document.createElement('div');
  notification.style.cssText = `
    position: fixed;
    top: 20px;
    right: 20px;
    background: #00cc00;
    color: white;
    padding: 15px 20px;
    border-radius: 4px;
    font-family: sans-serif;
    font-size: 14px;
    z-index: 10000;
    box-shadow: 0 2px 8px rgba(0,0,0,0.3);
  `;
  notification.textContent = `${message} (${detail})`;
  document.body.appendChild(notification);

  setTimeout(() => {
    notification.remove();
  }, 3000);
}