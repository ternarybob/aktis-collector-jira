// Content script for Aktis Jira Collector
// Runs on Jira pages - provides helper functions for data collection

console.log('Aktis Jira Collector content script loaded');

// Detect what type of Jira page we're on
function detectPageType() {
  const url = window.location.href;
  const path = window.location.pathname;

  // Check for Projects List page
  // Example: https://company.atlassian.net/jira/projects?page=1
  if ((url.includes('/jira/projects') || path === '/jira/projects') && !url.includes('/projects/')) {
    return 'projectsList';
  }

  // Check for Jira Cloud issue detail page (new format)
  if (url.includes('/browse/') || path.includes('/browse/')) {
    return 'issue';
  }

  // Check for Jira board/sprint pages
  if (url.includes('/board/') || path.includes('/secure/RapidBoard')) {
    return 'board';
  }

  // Check for Jira Cloud project issue list (new format)
  // Example: https://company.atlassian.net/jira/software/c/projects/TEST435/issues
  if (url.includes('/jira/software/c/projects/') && url.includes('/issues')) {
    return 'issueList';
  }

  // Check for Jira issue search/filter pages
  if (url.includes('/issues/') || url.includes('?jql=')) {
    return 'search';
  }

  // Check for project overview pages
  if (url.includes('/projects/')) {
    return 'project';
  }

  // Check if we're on any Atlassian Jira domain
  if (url.includes('.atlassian.net') || url.includes('/jira/')) {
    return 'generic';
  }

  return 'unknown';
}

// Extract Jira issue links from current page for auto-navigation
function extractJiraLinks() {
  const links = [];
  const seen = new Set();

  // Find all links that point to Jira issues
  const linkElements = document.querySelectorAll('a[href*="/browse/"], a[href*="/issues/"]');

  linkElements.forEach(link => {
    const href = link.href;

    // Extract issue key if present
    const issueMatch = href.match(/\/browse\/([A-Z]+-\d+)/);
    if (issueMatch && !seen.has(href)) {
      seen.add(href);
      links.push({
        url: href,
        issueKey: issueMatch[1],
        text: link.textContent.trim().substring(0, 100) // Limit text length
      });
    }
  });

  console.log(`Found ${links.length} Jira links on page`);
  return links;
}

// Extract ticket data directly from visible DOM elements
function extractTicketsFromDOM() {
  const tickets = [];

  // Try to find issue rows - Jira uses various selectors
  const rowSelectors = [
    '[data-issue-key]',  // Standard attribute
    '[data-testid*="issue"]',  // Test IDs
    'tr[id^="row-"]',  // Row IDs
    '.issue-row',  // Class names
  ];

  let rows = [];
  for (const selector of rowSelectors) {
    rows = document.querySelectorAll(selector);
    if (rows.length > 0) {
      console.log(`Found ${rows.length} rows using selector: ${selector}`);
      break;
    }
  }

  rows.forEach(row => {
    const ticket = {};

    // Extract issue key
    const key = row.getAttribute('data-issue-key') ||
                row.getAttribute('data-key') ||
                extractKeyFromElement(row);

    if (!key) return; // Skip if no key found

    ticket.key = key;
    ticket.url = `${window.location.origin}/browse/${key}`;

    // Extract project_id from key
    const projectMatch = key.match(/^([A-Z]+)-\d+$/);
    if (projectMatch) {
      ticket.project_id = projectMatch[1];
    }

    // Extract summary - usually in a link or span
    const summaryEl = row.querySelector('[data-testid*="summary"]') ||
                      row.querySelector('a[href*="/browse/"]') ||
                      row.querySelector('.summary');
    if (summaryEl) {
      ticket.summary = summaryEl.textContent.trim().replace(key, '').trim();
    }

    // Extract status - usually in a badge/lozenge
    const statusEl = row.querySelector('[data-testid*="status"]') ||
                     row.querySelector('.status-badge') ||
                     row.querySelector('[class*="lozenge"]');
    if (statusEl) {
      ticket.status = statusEl.textContent.trim();
    }

    // Extract priority
    const priorityEl = row.querySelector('[aria-label*="riority"]') ||
                       row.querySelector('[title*="riority"]') ||
                       row.querySelector('.priority');
    if (priorityEl) {
      const priorityText = priorityEl.getAttribute('aria-label') ||
                          priorityEl.getAttribute('title') ||
                          priorityEl.textContent.trim();
      ticket.priority = priorityText.replace('Priority:', '').trim();
    }

    // Extract issue type
    const typeEl = row.querySelector('[aria-label*="Issue Type"]') ||
                   row.querySelector('[data-testid*="type"]') ||
                   row.querySelector('.issuetype');
    if (typeEl) {
      const typeText = typeEl.getAttribute('aria-label') ||
                      typeEl.getAttribute('title') ||
                      typeEl.textContent.trim();
      ticket.issue_type = typeText.replace('Issue Type:', '').trim();
    }

    // Extract assignee
    const assigneeEl = row.querySelector('[data-testid*="assignee"]') ||
                       row.querySelector('[aria-label*="ssignee"]') ||
                       row.querySelector('.assignee');
    if (assigneeEl) {
      const assigneeText = assigneeEl.getAttribute('aria-label') ||
                          assigneeEl.getAttribute('title') ||
                          assigneeEl.textContent.trim();
      ticket.assignee = assigneeText.replace('Assignee:', '').trim();
    }

    tickets.push(ticket);
  });

  return tickets;
}

// Helper function to extract issue key from element text
function extractKeyFromElement(element) {
  const text = element.textContent;
  const keyMatch = text.match(/\b([A-Z]+-\d+)\b/);
  return keyMatch ? keyMatch[1] : null;
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