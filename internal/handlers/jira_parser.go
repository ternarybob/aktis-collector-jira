package handlers

import (
	"regexp"
	"strings"

	"aktis-collector-jira/internal/models"

	"golang.org/x/net/html"
)

// JiraParser handles parsing of Jira HTML pages
type JiraParser struct{}

// NewJiraParser creates a new Jira HTML parser
func NewJiraParser() *JiraParser {
	return &JiraParser{}
}

// ParseHTML parses Jira HTML and extracts issue data based on page type
func (p *JiraParser) ParseHTML(htmlContent, pageType, url string) ([]map[string]interface{}, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	switch pageType {
	case "projectsList":
		return p.parseProjectsListPage(doc, url)
	case "issue":
		return p.parseIssuePage(doc, url)
	case "issueList":
		return p.parseIssueListPage(doc, url)
	case "board":
		return p.parseBoardPage(doc, url)
	case "search":
		return p.parseSearchPage(doc, url)
	case "generic":
		// Try to extract any issues we can find
		return p.parseGenericPage(doc, url)
	default:
		return []map[string]interface{}{}, nil
	}
}

// parseProjectsListPage extracts all projects from the projects list page
func (p *JiraParser) parseProjectsListPage(doc *html.Node, url string) ([]map[string]interface{}, error) {
	projects := []map[string]interface{}{}

	// Find all project rows in the table
	projectRows := p.findProjectRows(doc)

	for _, row := range projectRows {
		project := p.extractProjectFromRow(row)
		if project != nil && project["key"] != nil {
			projects = append(projects, project)
		}
	}

	return projects, nil
}

// parseIssuePage extracts data from a single issue detail page
func (p *JiraParser) parseIssuePage(doc *html.Node, url string) ([]map[string]interface{}, error) {
	issue := make(map[string]interface{})

	// Extract issue key from URL
	keyRegex := regexp.MustCompile(`/browse/([A-Z]+-\d+)`)
	if matches := keyRegex.FindStringSubmatch(url); len(matches) > 1 {
		issueKey := matches[1]
		issue["key"] = issueKey
		issue["url"] = url

		// Extract project ID from issue key (e.g., API-123 -> API)
		projectKeyRegex := regexp.MustCompile(`^([A-Z]+)-\d+$`)
		if matches := projectKeyRegex.FindStringSubmatch(issueKey); len(matches) > 1 {
			issue["project_id"] = matches[1]
		}
	}

	// Store raw HTML for future processing
	var rawHTML strings.Builder
	htmlRender(doc, &rawHTML)
	issue["raw_html"] = rawHTML.String()

	// Parse HTML to extract basic fields
	p.traverseAndExtract(doc, issue, "issue")

	// Extract comprehensive details
	p.extractIssueDetails(doc, issue)
	p.extractComments(doc, issue)
	p.extractSubtasks(doc, issue)
	p.extractAttachments(doc, issue, url)
	p.extractLinks(doc, issue, url)

	if _, hasKey := issue["key"]; hasKey {
		return []map[string]interface{}{issue}, nil
	}

	return []map[string]interface{}{}, nil
}

// htmlRender converts HTML node to string
func htmlRender(n *html.Node, b *strings.Builder) {
	if n.Type == html.TextNode {
		b.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		htmlRender(c, b)
	}
}

// parseIssueListPage extracts multiple issues from a list/project page
func (p *JiraParser) parseIssueListPage(doc *html.Node, url string) ([]map[string]interface{}, error) {
	issues := []map[string]interface{}{}

	// Extract project key from URL to filter only relevant issues
	projectKey := p.extractProjectKeyFromURL(url)

	// Extract base URL for building issue URLs
	baseURLRegex := regexp.MustCompile(`(https?://[^/]+)`)
	var baseURL string
	if matches := baseURLRegex.FindStringSubmatch(url); len(matches) > 1 {
		baseURL = matches[1]
	}

	// Find issue rows in the table (more precise than scanning entire HTML)
	issueRows := p.findIssueRows(doc)

	for _, row := range issueRows {
		issue := p.extractIssueFromRow(row, projectKey)
		if issue != nil && issue["key"] != nil {
			// Add project_id and URL
			if issueKey, ok := issue["key"].(string); ok {
				projectKeyRegex := regexp.MustCompile(`^([A-Z]+)-\d+$`)
				if matches := projectKeyRegex.FindStringSubmatch(issueKey); len(matches) > 1 {
					issue["project_id"] = matches[1]
				}
				if baseURL != "" {
					issue["url"] = baseURL + "/browse/" + issueKey
				}
			}
			issues = append(issues, issue)
		}
	}

	// Fallback: if no rows found, try generic extraction but filter by project
	if len(issues) == 0 && projectKey != "" {
		allKeys := p.extractIssueKeys(doc)
		for key := range allKeys {
			// Only include keys matching the current project
			if strings.HasPrefix(key, projectKey+"-") {
				issue := make(map[string]interface{})
				issue["key"] = key
				issue["project_id"] = projectKey
				if baseURL != "" {
					issue["url"] = baseURL + "/browse/" + key
				}
				issues = append(issues, issue)
			}
		}
	}

	return issues, nil
}

// parseBoardPage extracts issues from a board/kanban view
func (p *JiraParser) parseBoardPage(doc *html.Node, url string) ([]map[string]interface{}, error) {
	issues := []map[string]interface{}{}

	// Extract issue keys from board
	issueKeys := p.extractIssueKeys(doc)

	for key := range issueKeys {
		issue := make(map[string]interface{})
		issue["key"] = key
		issues = append(issues, issue)
	}

	return issues, nil
}

// parseSearchPage extracts issues from search results
func (p *JiraParser) parseSearchPage(doc *html.Node, url string) ([]map[string]interface{}, error) {
	return p.parseIssueListPage(doc, url)
}

// parseGenericPage tries to extract any issue data from unknown page types
func (p *JiraParser) parseGenericPage(doc *html.Node, url string) ([]map[string]interface{}, error) {
	issues := []map[string]interface{}{}

	// Try to extract any issue keys we can find
	issueKeys := p.extractIssueKeys(doc)

	for key := range issueKeys {
		issue := make(map[string]interface{})
		issue["key"] = key
		issues = append(issues, issue)
	}

	return issues, nil
}

// extractIssueKeys finds all Jira issue keys in the HTML
func (p *JiraParser) extractIssueKeys(node *html.Node) map[string]bool {
	keys := make(map[string]bool)
	keyRegex := regexp.MustCompile(`\b([A-Z]+-\d+)\b`)

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.TextNode {
			matches := keyRegex.FindAllString(n.Data, -1)
			for _, match := range matches {
				keys[match] = true
			}
		}

		// Also check href attributes
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				if attr.Key == "href" && strings.Contains(attr.Val, "/browse/") {
					matches := keyRegex.FindAllString(attr.Val, -1)
					for _, match := range matches {
						keys[match] = true
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(node)
	return keys
}

// traverseAndExtract walks the HTML tree and extracts issue field data
func (p *JiraParser) traverseAndExtract(node *html.Node, issue map[string]interface{}, pageType string) {
	if node.Type == html.ElementNode {
		// Extract data-testid and data-test-id attributes
		var testId string
		for _, attr := range node.Attr {
			if attr.Key == "data-testid" || attr.Key == "data-test-id" {
				testId = attr.Val
				break
			}
		}

		// Map test IDs to issue fields
		if testId != "" {
			text := p.extractText(node)
			if text != "" {
				if strings.Contains(testId, "summary") {
					issue["summary"] = text
				} else if strings.Contains(testId, "description") {
					issue["description"] = text
				} else if strings.Contains(testId, "issue-type") {
					issue["issueType"] = text
				} else if strings.Contains(testId, "status") {
					issue["status"] = text
				} else if strings.Contains(testId, "priority") {
					issue["priority"] = text
				}
			}
		}
	}

	// Recursively process children
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		p.traverseAndExtract(c, issue, pageType)
	}
}

// extractText gets all text content from a node and its children
func (p *JiraParser) extractText(node *html.Node) string {
	var text strings.Builder

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.TextNode {
			text.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(node)
	return strings.TrimSpace(text.String())
}

// findProjectRows finds all table rows containing project data
func (p *JiraParser) findProjectRows(node *html.Node) []*html.Node {
	var rows []*html.Node

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			// Check if this row contains project data
			// Look for cells with project key
			hasProjectKey := false
			var checkCells func(*html.Node)
			checkCells = func(cell *html.Node) {
				if cell.Type == html.ElementNode && (cell.Data == "td" || cell.Data == "th") {
					text := p.extractText(cell)
					// Project keys are typically 2-10 uppercase letters/numbers
					projectKeyRegex := regexp.MustCompile(`^[A-Z0-9]{2,10}$`)
					if projectKeyRegex.MatchString(strings.TrimSpace(text)) {
						hasProjectKey = true
					}
				}
				for c := cell.FirstChild; c != nil; c = c.NextSibling {
					checkCells(c)
				}
			}
			checkCells(n)

			if hasProjectKey {
				rows = append(rows, n)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(node)
	return rows
}

// extractProjectFromRow extracts project data from a table row
func (p *JiraParser) extractProjectFromRow(row *html.Node) map[string]interface{} {
	project := make(map[string]interface{})

	cells := []*html.Node{}
	var getCells func(*html.Node)
	getCells = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "td" {
			cells = append(cells, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			getCells(c)
		}
	}
	getCells(row)

	// Extract data from cells
	// Typically: [Name, Key, Type, Lead, URL]
	for _, cell := range cells {
		text := strings.TrimSpace(p.extractText(cell))

		// Find project key (uppercase letters/numbers, 2-10 chars)
		projectKeyRegex := regexp.MustCompile(`^[A-Z0-9]{2,10}$`)
		if projectKeyRegex.MatchString(text) && project["key"] == nil {
			project["key"] = text
			continue
		}

		// Find project name (usually has a link)
		links := p.findLinks(cell)
		for _, link := range links {
			if strings.Contains(link, "/projects/") || strings.Contains(link, "/browse/") {
				linkText := strings.TrimSpace(p.extractText(cell))
				if linkText != "" && project["name"] == nil {
					project["name"] = linkText
					project["url"] = link
					// Extract project ID from URL
					// Example: /projects/12345 or /browse/API-123 (where API is the key)
					projectIDRegex := regexp.MustCompile(`/projects/(\d+)`)
					if matches := projectIDRegex.FindStringSubmatch(link); len(matches) > 1 {
						project["id"] = matches[1]
					}
				}
			}
		}

		// Extract type
		if strings.Contains(strings.ToLower(text), "software") ||
			strings.Contains(strings.ToLower(text), "managed") {
			if project["type"] == nil {
				project["type"] = text
			}
		}
	}

	return project
}

// findLinks extracts all href URLs from a node
func (p *JiraParser) findLinks(node *html.Node) []string {
	var links []string

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					links = append(links, attr.Val)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(node)
	return links
}

// extractProjectKeyFromURL extracts the project key from the Jira URL
func (p *JiraParser) extractProjectKeyFromURL(url string) string {
	// Try to extract from /projects/PROJECTKEY/ pattern
	projectRegex := regexp.MustCompile(`/projects/([A-Z0-9]+)`)
	if matches := projectRegex.FindStringSubmatch(url); len(matches) > 1 {
		return matches[1]
	}

	// Try to extract from JQL query parameter
	jqlRegex := regexp.MustCompile(`project\s*=\s*"?([A-Z0-9]+)"?`)
	if matches := jqlRegex.FindStringSubmatch(url); len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// findIssueRows finds all table rows or list items that contain issue data
// Supports both old Jira (tables) and new Jira Cloud (divs with virtual scrolling)
func (p *JiraParser) findIssueRows(node *html.Node) []*html.Node {
	var rows []*html.Node
	keyRegex := regexp.MustCompile(`\b([A-Z]+-\d+)\b`)

	var traverse func(*html.Node, int)
	traverse = func(n *html.Node, depth int) {
		if n.Type == html.ElementNode {
			isCandidate := false
			hasIssueKey := false

			// Strategy 1: Check for explicit data-issue-key attribute (old Jira)
			if n.Data == "tr" {
				for _, attr := range n.Attr {
					if attr.Key == "data-issue-key" {
						rows = append(rows, n)
						return
					}
				}
			}

			// Strategy 2: Check for specific test IDs (both old and new Jira)
			for _, attr := range n.Attr {
				val := strings.ToLower(attr.Val)

				// Check data-testid patterns
				if attr.Key == "data-testid" || attr.Key == "data-test-id" {
					// Modern Jira Cloud: issue container patterns
					if strings.Contains(val, "issue.") ||
						strings.Contains(val, "issue-") ||
						(strings.Contains(val, "issue") && strings.Contains(val, "row")) ||
						(strings.Contains(val, "issue") && strings.Contains(val, "container")) {
						isCandidate = true
					}
				}

				// Check for issue keys in data attributes
				if strings.Contains(attr.Key, "data-") && keyRegex.MatchString(attr.Val) {
					hasIssueKey = true
					isCandidate = true
				}
			}

			// Strategy 3: Modern Jira Cloud - divs containing issue links
			// Look for divs that contain links to /browse/ with issue keys
			if (n.Data == "div" || n.Data == "li" || n.Data == "article") && depth < 15 {
				hasLink := false
				var checkForIssueLink func(*html.Node, int) bool
				checkForIssueLink = func(child *html.Node, childDepth int) bool {
					if childDepth > 5 { // Only look 5 levels deep
						return false
					}
					if child.Type == html.ElementNode && child.Data == "a" {
						for _, attr := range child.Attr {
							if attr.Key == "href" && strings.Contains(attr.Val, "/browse/") {
								if keyRegex.MatchString(attr.Val) {
									return true
								}
							}
						}
					}
					for c := child.FirstChild; c != nil; c = c.NextSibling {
						if checkForIssueLink(c, childDepth+1) {
							return true
						}
					}
					return false
				}
				hasLink = checkForIssueLink(n, 0)

				if hasLink {
					// Check if this element is a reasonable size for an issue row
					// by checking if it has multiple child elements (not just a tiny wrapper)
					childCount := 0
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						if c.Type == html.ElementNode {
							childCount++
						}
					}
					if childCount >= 2 { // Must have at least 2 child elements
						isCandidate = true
						hasIssueKey = true
					}
				}
			}

			// Strategy 4: Check for issue keys in text content (for elements with specific roles/classes)
			if !isCandidate && (n.Data == "div" || n.Data == "li" || n.Data == "tr") {
				for _, attr := range n.Attr {
					// Check for semantic attributes
					if (attr.Key == "role" && (attr.Val == "row" || attr.Val == "listitem")) ||
						(attr.Key == "class" && (strings.Contains(attr.Val, "issue") || strings.Contains(attr.Val, "row"))) {
						// Check if it contains an issue key in immediate children
						text := ""
						for c := n.FirstChild; c != nil && len(text) < 200; c = c.NextSibling {
							if c.Type == html.TextNode {
								text += c.Data
							} else if c.Type == html.ElementNode {
								text += p.extractText(c)
							}
						}
						if keyRegex.MatchString(text) {
							isCandidate = true
							hasIssueKey = true
						}
					}
				}
			}

			// Add to results if it's a candidate and has issue key
			if isCandidate && hasIssueKey {
				rows = append(rows, n)
				return // Don't traverse children of matched rows
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c, depth+1)
		}
	}

	traverse(node, 0)
	return rows
}

// extractIssueFromRow extracts issue data from a table row or list item
// Enhanced to support modern Jira Cloud's diverse HTML structures
func (p *JiraParser) extractIssueFromRow(row *html.Node, projectFilter string) map[string]interface{} {
	issue := make(map[string]interface{})
	keyRegex := regexp.MustCompile(`\b([A-Z]+-\d+)\b`)

	// Strategy 1: Check data-issue-key attribute first (old Jira)
	for _, attr := range row.Attr {
		if attr.Key == "data-issue-key" {
			if projectFilter == "" || strings.HasPrefix(attr.Val, projectFilter+"-") {
				issue["key"] = attr.Val
			}
		}

		// Also check other data attributes that might contain the key
		if strings.Contains(attr.Key, "data-") {
			matches := keyRegex.FindStringSubmatch(attr.Val)
			if len(matches) > 1 {
				key := matches[1]
				if projectFilter == "" || strings.HasPrefix(key, projectFilter+"-") {
					issue["key"] = key
				}
			}
		}
	}

	// Strategy 2: If no key yet, search links first (highest priority)
	if issue["key"] == nil {
		var findKeyFromLink func(*html.Node) string
		findKeyFromLink = func(n *html.Node) string {
			if n.Type == html.ElementNode && n.Data == "a" {
				for _, attr := range n.Attr {
					if attr.Key == "href" && strings.Contains(attr.Val, "/browse/") {
						matches := keyRegex.FindStringSubmatch(attr.Val)
						if len(matches) > 1 {
							key := matches[1]
							if projectFilter == "" || strings.HasPrefix(key, projectFilter+"-") {
								return key
							}
						}
					}
				}
			}

			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if key := findKeyFromLink(c); key != "" {
					return key
				}
			}
			return ""
		}

		if key := findKeyFromLink(row); key != "" {
			issue["key"] = key
		}
	}

	// Strategy 3: If still no key, search all text content
	if issue["key"] == nil {
		allText := p.extractText(row)
		matches := keyRegex.FindAllString(allText, -1)

		// Find the first valid key that matches project filter
		for _, match := range matches {
			if projectFilter == "" || strings.HasPrefix(match, projectFilter+"-") {
				issue["key"] = match
				break
			}
		}
	}

	// Extract additional fields from the row
	if issue["key"] != nil {
		issueKey := issue["key"].(string)

		// Extract summary
		summary := p.extractSummaryFromRow(row, issueKey)
		if summary != "" {
			issue["summary"] = summary
		}

		// Extract status, priority, issue type, assignee from table cells or data attributes
		p.extractAdditionalFieldsFromRow(row, issue)
	}

	return issue
}

// extractAdditionalFieldsFromRow extracts status, priority, type, assignee from a row
func (p *JiraParser) extractAdditionalFieldsFromRow(row *html.Node, issue map[string]interface{}) {
	// Extract all text and attributes, then use pattern matching
	allText := p.extractText(row)

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Collect all attributes
			attrs := make(map[string]string)
			for _, attr := range n.Attr {
				attrs[attr.Key] = attr.Val
			}

			// Get element text
			elemText := strings.TrimSpace(p.extractText(n))

			// Try multiple strategies for each field

			// Status - look for badge/lozenge patterns
			if issue["status"] == nil {
				for key, val := range attrs {
					if strings.Contains(strings.ToLower(key+val), "status") ||
						strings.Contains(strings.ToLower(val), "lozenge") {
						if elemText != "" && len(elemText) < 30 {
							issue["status"] = elemText
							break
						}
					}
				}
			}

			// Priority - check aria-label, title, or img alt
			if issue["priority"] == nil {
				for _, val := range attrs {
					lowerVal := strings.ToLower(val)
					if strings.Contains(lowerVal, "priority") {
						// Extract priority name from patterns like "Priority: High" or "High Priority"
						priority := strings.TrimSpace(val)
						priority = strings.ReplaceAll(priority, "Priority:", "")
						priority = strings.ReplaceAll(priority, "Priority", "")
						priority = strings.TrimSpace(priority)
						if priority != "" && len(priority) < 20 {
							issue["priority"] = priority
							break
						}
					}
				}
			}

			// Issue Type
			if issue["issue_type"] == nil {
				for _, val := range attrs {
					if strings.Contains(strings.ToLower(val), "issue") &&
						strings.Contains(strings.ToLower(val), "type") {
						issueType := strings.TrimSpace(val)
						issueType = strings.ReplaceAll(issueType, "Issue Type:", "")
						issueType = strings.TrimSpace(issueType)
						if issueType != "" && len(issueType) < 30 {
							issue["issue_type"] = issueType
							break
						}
					}
				}
			}

			// Assignee
			if issue["assignee"] == nil {
				for _, val := range attrs {
					if strings.Contains(strings.ToLower(val), "assignee") {
						assignee := strings.TrimSpace(val)
						assignee = strings.ReplaceAll(assignee, "Assignee:", "")
						assignee = strings.TrimSpace(assignee)
						if assignee != "" && len(assignee) < 100 {
							issue["assignee"] = assignee
							break
						}
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(row)

	// Fallback: parse common text patterns
	if issue["status"] == nil {
		// Look for status keywords in text
		statusKeywords := []string{"To Do", "In Progress", "Done", "Closed", "Open", "Resolved", "In Review"}
		for _, keyword := range statusKeywords {
			if strings.Contains(allText, keyword) {
				issue["status"] = keyword
				break
			}
		}
	}
}

// extractSummaryFromRow extracts the summary text from a row
func (p *JiraParser) extractSummaryFromRow(row *html.Node, issueKey string) string {
	var summary string

	var findSummary func(*html.Node)
	findSummary = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Look for elements with summary-related attributes
			for _, attr := range n.Attr {
				if (attr.Key == "data-testid" || attr.Key == "data-test-id") &&
					strings.Contains(attr.Val, "summary") {
					text := p.extractText(n)
					// Remove the issue key from the summary if present
					text = strings.ReplaceAll(text, issueKey, "")
					text = strings.TrimSpace(text)
					if text != "" && len(text) > 5 { // Ensure it's not just whitespace or very short
						summary = text
						return
					}
				}
			}

			// Look for links that might contain the summary
			if n.Data == "a" {
				for _, attr := range n.Attr {
					if attr.Key == "href" && strings.Contains(attr.Val, "/browse/"+issueKey) {
						text := p.extractText(n)
						text = strings.ReplaceAll(text, issueKey, "")
						text = strings.TrimSpace(text)
						if text != "" && len(text) > 5 {
							summary = text
							return
						}
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findSummary(c)
			if summary != "" {
				return
			}
		}
	}

	findSummary(row)
	return summary
}

// ConvertToTicketData converts parsed issue data to TicketData struct
func (p *JiraParser) ConvertToTicketData(issueData map[string]interface{}, timestamp string) *models.TicketData {
	ticket := &models.TicketData{
		Updated: timestamp,
	}

	if key, ok := issueData["key"].(string); ok {
		ticket.Key = key
	}
	if summary, ok := issueData["summary"].(string); ok {
		ticket.Summary = summary
	}
	if description, ok := issueData["description"].(string); ok {
		ticket.Description = description
	}
	if issueType, ok := issueData["issueType"].(string); ok {
		ticket.IssueType = issueType
	}
	if status, ok := issueData["status"].(string); ok {
		ticket.Status = status
	}
	if priority, ok := issueData["priority"].(string); ok {
		ticket.Priority = priority
	}

	return ticket
}
