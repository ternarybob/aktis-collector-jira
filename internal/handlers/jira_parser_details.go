package handlers

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// extractIssueDetails extracts comprehensive issue details from HTML
func (p *JiraParser) extractIssueDetails(doc *html.Node, issue map[string]interface{}) {
	// Extract description - look for description field
	p.findAndExtractField(doc, "description", issue, []string{
		"data-testid=issue.views.field.rich-text.description",
		"id=description-val",
		"data-testid=issue.views.issue-base.foundation.description",
	})

	// Extract summary if not already present
	if _, exists := issue["summary"]; !exists {
		p.findAndExtractField(doc, "summary", issue, []string{
			"data-testid=issue.views.issue-base.foundation.summary",
			"data-testid=issue-summary",
			"id=summary-val",
		})
	}

	// Extract labels
	labels := p.extractListField(doc, []string{
		"data-testid=issue.views.field.labels",
		"data-testid=issue.views.field.labels.common.ui.labels",
	})
	if len(labels) > 0 {
		issue["labels"] = labels
	}

	// Extract components
	components := p.extractListField(doc, []string{
		"data-testid=issue.views.field.components",
	})
	if len(components) > 0 {
		issue["components"] = components
	}
}

// extractComments extracts all comments from the issue page
func (p *JiraParser) extractComments(doc *html.Node, issue map[string]interface{}) {
	comments := []map[string]interface{}{}

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Look for comment containers
			for _, attr := range n.Attr {
				if (attr.Key == "data-testid" && strings.Contains(attr.Val, "comment")) ||
					(attr.Key == "class" && strings.Contains(attr.Val, "activity-comment")) {

					comment := p.extractSingleComment(n)
					if len(comment) > 0 {
						comments = append(comments, comment)
					}
					return // Don't traverse into found comments
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)

	if len(comments) > 0 {
		issue["comments"] = comments
	}
}

// extractSingleComment extracts a single comment's data
func (p *JiraParser) extractSingleComment(n *html.Node) map[string]interface{} {
	comment := make(map[string]interface{})

	// Extract comment ID
	for _, attr := range n.Attr {
		if attr.Key == "id" || attr.Key == "data-comment-id" {
			comment["id"] = attr.Val
		}
	}

	// Extract text content
	text := p.extractText(n)
	if text != "" {
		comment["body"] = text
	}

	// Try to extract author and timestamps
	var extractMeta func(*html.Node)
	extractMeta = func(node *html.Node) {
		if node.Type == html.ElementNode {
			for _, attr := range node.Attr {
				val := strings.ToLower(attr.Val)
				if strings.Contains(val, "author") || strings.Contains(val, "user") {
					if authorText := strings.TrimSpace(p.extractText(node)); authorText != "" && len(authorText) < 100 {
						comment["author"] = authorText
					}
				}
				if strings.Contains(val, "date") || strings.Contains(val, "time") {
					if timeText := strings.TrimSpace(p.extractText(node)); timeText != "" && len(timeText) < 50 {
						if _, exists := comment["created"]; !exists {
							comment["created"] = timeText
						}
					}
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			extractMeta(c)
		}
	}

	extractMeta(n)

	return comment
}

// extractSubtasks extracts all subtasks from the issue page
func (p *JiraParser) extractSubtasks(doc *html.Node, issue map[string]interface{}) {
	subtasks := []map[string]interface{}{}

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				if (attr.Key == "data-testid" && strings.Contains(attr.Val, "subtask")) ||
					(attr.Key == "class" && strings.Contains(attr.Val, "subtask")) {

					subtask := p.extractSingleSubtask(n)
					if len(subtask) > 0 {
						subtasks = append(subtasks, subtask)
					}
					return
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)

	if len(subtasks) > 0 {
		issue["subtasks"] = subtasks
	}
}

// extractSingleSubtask extracts a single subtask's data
func (p *JiraParser) extractSingleSubtask(n *html.Node) map[string]interface{} {
	subtask := make(map[string]interface{})

	// Look for issue key in links
	var findKey func(*html.Node)
	findKey = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "a" {
			for _, attr := range node.Attr {
				if attr.Key == "href" && strings.Contains(attr.Val, "/browse/") {
					keyRegex := regexp.MustCompile(`/browse/([A-Z]+-\d+)`)
					if matches := keyRegex.FindStringSubmatch(attr.Val); len(matches) > 1 {
						subtask["key"] = matches[1]
						subtask["url"] = attr.Val
					}
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			findKey(c)
		}
	}

	findKey(n)

	// Extract summary
	text := strings.TrimSpace(p.extractText(n))
	if text != "" {
		subtask["summary"] = text
	}

	return subtask
}

// extractAttachments extracts all attachments from the issue page
func (p *JiraParser) extractAttachments(doc *html.Node, issue map[string]interface{}, baseURL string) {
	attachments := []map[string]interface{}{}

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				if (attr.Key == "data-testid" && strings.Contains(attr.Val, "attachment")) ||
					(attr.Key == "class" && strings.Contains(attr.Val, "attachment")) {

					attachment := p.extractSingleAttachment(n, baseURL)
					if len(attachment) > 0 {
						attachments = append(attachments, attachment)
					}
					return
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)

	if len(attachments) > 0 {
		issue["attachments"] = attachments
	}
}

// extractSingleAttachment extracts a single attachment's data
func (p *JiraParser) extractSingleAttachment(n *html.Node, baseURL string) map[string]interface{} {
	attachment := make(map[string]interface{})

	// Extract filename and URL from links
	var findFile func(*html.Node)
	findFile = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "a" {
			for _, attr := range node.Attr {
				if attr.Key == "href" {
					attachment["url"] = attr.Val
				}
				if attr.Key == "download" || attr.Key == "title" {
					attachment["filename"] = attr.Val
				}
			}
			// Get text as filename if not found
			if _, exists := attachment["filename"]; !exists {
				if text := strings.TrimSpace(p.extractText(node)); text != "" {
					attachment["filename"] = text
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			findFile(c)
		}
	}

	findFile(n)

	return attachment
}

// extractLinks extracts all issue links from the page
func (p *JiraParser) extractLinks(doc *html.Node, issue map[string]interface{}, baseURL string) {
	links := []map[string]interface{}{}

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				if (attr.Key == "data-testid" && strings.Contains(attr.Val, "issue-link")) ||
					(attr.Key == "class" && strings.Contains(attr.Val, "issue-link")) {

					link := p.extractSingleLink(n)
					if len(link) > 0 {
						links = append(links, link)
					}
					return
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)

	if len(links) > 0 {
		issue["links"] = links
	}
}

// extractSingleLink extracts a single issue link's data
func (p *JiraParser) extractSingleLink(n *html.Node) map[string]interface{} {
	link := make(map[string]interface{})

	// Look for linked issue key
	var findLinked func(*html.Node)
	findLinked = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "a" {
			for _, attr := range node.Attr {
				if attr.Key == "href" && strings.Contains(attr.Val, "/browse/") {
					keyRegex := regexp.MustCompile(`/browse/([A-Z]+-\d+)`)
					if matches := keyRegex.FindStringSubmatch(attr.Val); len(matches) > 1 {
						link["issue_key"] = matches[1]
						link["url"] = attr.Val
					}
				}
			}
			// Get summary text
			if text := strings.TrimSpace(p.extractText(node)); text != "" {
				link["issue_summary"] = text
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			findLinked(c)
		}
	}

	findLinked(n)

	// Extract link type from surrounding text
	text := p.extractText(n)
	if strings.Contains(strings.ToLower(text), "blocks") {
		link["link_type"] = "blocks"
	} else if strings.Contains(strings.ToLower(text), "blocked") {
		link["link_type"] = "is blocked by"
	} else if strings.Contains(strings.ToLower(text), "relates") {
		link["link_type"] = "relates to"
	}

	return link
}

// findAndExtractField looks for a field by test IDs and extracts its text
func (p *JiraParser) findAndExtractField(doc *html.Node, fieldName string, issue map[string]interface{}, testIDs []string) {
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				for _, testID := range testIDs {
					if (attr.Key == "data-testid" && attr.Val == testID) ||
						(attr.Key == "id" && attr.Val == testID) {
						text := strings.TrimSpace(p.extractText(n))
						if text != "" {
							issue[fieldName] = text
							return
						}
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)
}

// extractListField extracts a list of values from a field
func (p *JiraParser) extractListField(doc *html.Node, testIDs []string) []string {
	items := []string{}

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				for _, testID := range testIDs {
					if strings.Contains(attr.Val, testID) {
						// Extract all text items from child spans or divs
						var extractItems func(*html.Node)
						extractItems = func(node *html.Node) {
							if node.Type == html.ElementNode && (node.Data == "span" || node.Data == "a") {
								text := strings.TrimSpace(p.extractText(node))
								if text != "" && len(text) < 100 {
									items = append(items, text)
								}
							}
							for c := node.FirstChild; c != nil; c = c.NextSibling {
								extractItems(c)
							}
						}
						extractItems(n)
						return
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)

	return items
}
