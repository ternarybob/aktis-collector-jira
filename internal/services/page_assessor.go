package services

import (
	"fmt"
	"regexp"
	"strings"

	"aktis-collector-jira/internal/interfaces"
	"aktis-collector-jira/internal/models"

	"github.com/ternarybob/arbor"
	"golang.org/x/net/html"
)

type pageAssessor struct {
	logger arbor.ILogger
}

// NewPageAssessor creates a new page assessment service
func NewPageAssessor(logger arbor.ILogger) interfaces.PageAssessor {
	return &pageAssessor{
		logger: logger,
	}
}

// AssessPage analyzes HTML and URL to determine page type without parsing full content
func (pa *pageAssessor) AssessPage(htmlContent, url string) (*models.PageAssessment, error) {
	assessment := &models.PageAssessment{
		PageType:    "unknown",
		Confidence:  "low",
		Description: "Page type could not be determined",
		Indicators:  []string{},
		Collectable: false,
	}

	// Parse HTML to check for indicators
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		pa.logger.Warn().Err(err).Msg("Failed to parse HTML for assessment")
		return assessment, nil // Return assessment anyway, don't error
	}

	// Check URL patterns first
	urlIndicators := pa.checkURLPatterns(url)
	assessment.Indicators = append(assessment.Indicators, urlIndicators...)

	// Check HTML structure
	htmlIndicators := pa.checkHTMLStructure(doc)
	assessment.Indicators = append(assessment.Indicators, htmlIndicators...)

	// Determine page type based on indicators
	assessment.PageType = pa.determinePageType(url, assessment.Indicators)
	assessment.Confidence = pa.calculateConfidence(assessment.Indicators)
	assessment.Description = pa.getPageDescription(assessment.PageType)
	assessment.Collectable = pa.isCollectable(assessment.PageType, assessment.Confidence)

	pa.logger.Debug().
		Str("page_type", assessment.PageType).
		Str("confidence", assessment.Confidence).
		Str("collectable", fmt.Sprintf("%v", assessment.Collectable)).
		Int("indicators", len(assessment.Indicators)).
		Msg("Page assessment completed")

	return assessment, nil
}

// checkURLPatterns checks URL for known Jira patterns
func (pa *pageAssessor) checkURLPatterns(url string) []string {
	indicators := []string{}

	// Check for Projects List page (multiple variations)
	if strings.Contains(url, "/jira/projects") && !strings.Contains(url, "/projects/") {
		indicators = append(indicators, "url_pattern:projects_list")
	}
	// Also check for direct /projects endpoint
	if strings.HasSuffix(url, "/projects") || strings.Contains(url, "/projects?") {
		indicators = append(indicators, "url_pattern:projects_list")
	}

	// Check for Issue Detail page
	if strings.Contains(url, "/browse/") {
		indicators = append(indicators, "url_pattern:issue_detail")
	}

	// Check for Issue List page
	if strings.Contains(url, "/jira/software/c/projects/") && strings.Contains(url, "/issues") {
		indicators = append(indicators, "url_pattern:issue_list")
	}

	// Check for Board page
	if strings.Contains(url, "/board/") || strings.Contains(url, "/secure/RapidBoard") {
		indicators = append(indicators, "url_pattern:board")
	}

	// Check for Search page
	if strings.Contains(url, "/issues/") || strings.Contains(url, "?jql=") {
		indicators = append(indicators, "url_pattern:search")
	}

	// Check if it's any Jira page
	if strings.Contains(url, ".atlassian.net") || strings.Contains(url, "/jira/") {
		indicators = append(indicators, "url_pattern:jira_domain")
	}

	return indicators
}

// checkHTMLStructure looks for HTML patterns that indicate page type
// Enhanced to detect modern Jira Cloud structures
func (pa *pageAssessor) checkHTMLStructure(doc *html.Node) []string {
	indicators := []string{}
	issueKeyRegex := regexp.MustCompile(`\b([A-Z]+-\d+)\b`)

	issueLinksCount := 0
	issueRowsCount := 0
	projectLinksCount := 0

	var traverse func(*html.Node, int)
	traverse = func(n *html.Node, depth int) {
		if n.Type == html.ElementNode {
			// Check for project table structure
			if n.Data == "table" {
				for _, attr := range n.Attr {
					if attr.Key == "data-testid" && strings.Contains(attr.Val, "project") {
						indicators = append(indicators, "html_structure:project_table")
					}
				}
			}

			// Check for explicit issue rows (old Jira)
			if n.Data == "tr" || n.Data == "div" || n.Data == "li" {
				for _, attr := range n.Attr {
					if attr.Key == "data-issue-key" {
						issueRowsCount++
						if issueRowsCount == 1 {
							indicators = append(indicators, "html_structure:issue_rows")
						}
					}
					if attr.Key == "data-testid" || attr.Key == "data-test-id" {
						val := strings.ToLower(attr.Val)
						if strings.Contains(val, "issue") {
							if strings.Contains(val, "row") || strings.Contains(val, "container") || strings.Contains(val, "issue.") {
								issueRowsCount++
								if issueRowsCount == 1 {
									indicators = append(indicators, "html_structure:issue_elements")
								}
							}
						}
					}
				}
			}

			// Check for issue links (modern Jira Cloud)
			// Count links to /browse/ with issue keys
			if n.Data == "a" && depth < 20 {
				for _, attr := range n.Attr {
					if attr.Key == "href" && strings.Contains(attr.Val, "/browse/") {
						if issueKeyRegex.MatchString(attr.Val) {
							issueLinksCount++
						}
					}
				}
			}

			// Check for board columns
			if n.Data == "div" {
				for _, attr := range n.Attr {
					if (attr.Key == "class" || attr.Key == "data-testid") &&
						(strings.Contains(attr.Val, "board") || strings.Contains(attr.Val, "column")) {
						indicators = append(indicators, "html_structure:board_layout")
					}
				}
			}

			// Check for issue detail page elements
			if n.Data == "div" || n.Data == "section" {
				for _, attr := range n.Attr {
					if (attr.Key == "data-testid" || attr.Key == "id") &&
						(strings.Contains(attr.Val, "issue-view") ||
							strings.Contains(attr.Val, "issue-details") ||
							strings.Contains(attr.Val, "issue.views.issue-details")) {
						indicators = append(indicators, "html_structure:issue_detail_layout")
					}
				}
			}

			// Check for project links (modern Jira Cloud projects list)
			if n.Data == "a" && depth < 20 {
				for _, attr := range n.Attr {
					if attr.Key == "href" && strings.Contains(attr.Val, "/projects/") {
						projectLinksCount++
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c, depth+1)
		}
	}

	traverse(doc, 0)

	// Add indicators based on counts
	if issueLinksCount >= 3 {
		indicators = append(indicators, "html_structure:multiple_issue_links")
		pa.logger.Debug().Int("count", issueLinksCount).Msg("Found multiple issue links")
	}
	if issueRowsCount >= 3 {
		indicators = append(indicators, "html_structure:multiple_issue_rows")
		pa.logger.Debug().Int("count", issueRowsCount).Msg("Found multiple issue rows")
	}
	if projectLinksCount >= 3 {
		indicators = append(indicators, "html_structure:multiple_project_links")
		pa.logger.Debug().Int("count", projectLinksCount).Msg("Found multiple project links")
	}

	return indicators
}

// determinePageType determines the page type based on indicators
// Enhanced to prioritize content-based detection over URL patterns
func (pa *pageAssessor) determinePageType(url string, indicators []string) string {
	// Count indicator types
	hasProjectsURLPattern := false
	hasIssueDetailURLPattern := false
	hasIssueListURLPattern := false
	hasBoardURLPattern := false
	hasSearchURLPattern := false

	hasProjectTable := false
	hasProjectLinks := false
	hasIssueRows := false
	hasIssueLinks := false
	hasBoardLayout := false
	hasIssueDetailLayout := false

	for _, indicator := range indicators {
		switch indicator {
		case "url_pattern:projects_list":
			hasProjectsURLPattern = true
		case "url_pattern:issue_detail":
			hasIssueDetailURLPattern = true
		case "url_pattern:issue_list":
			hasIssueListURLPattern = true
		case "url_pattern:board":
			hasBoardURLPattern = true
		case "url_pattern:search":
			hasSearchURLPattern = true
		case "html_structure:project_table":
			hasProjectTable = true
		case "html_structure:multiple_project_links":
			hasProjectLinks = true
		case "html_structure:issue_rows", "html_structure:issue_elements", "html_structure:multiple_issue_rows":
			hasIssueRows = true
		case "html_structure:multiple_issue_links":
			hasIssueLinks = true
		case "html_structure:board_layout":
			hasBoardLayout = true
		case "html_structure:issue_detail_layout":
			hasIssueDetailLayout = true
		}
	}

	// Determine page type with priority
	// Priority 1: Issue detail (specific page, high priority)
	if hasIssueDetailURLPattern || hasIssueDetailLayout {
		return "issue"
	}

	// Priority 2: Projects list
	if hasProjectsURLPattern || hasProjectTable || hasProjectLinks {
		return "projectsList"
	}

	// Priority 3: Issue list (content-based detection takes priority over URL)
	// If we see multiple issue links/rows, it's likely a list regardless of URL
	if hasIssueLinks || hasIssueRows {
		// Check URL to differentiate between board, search, and issueList
		if hasBoardURLPattern || hasBoardLayout {
			return "board"
		}
		if hasSearchURLPattern {
			return "search"
		}
		// Default to issueList if we have issue content but unclear URL
		return "issueList"
	}

	// Priority 4: URL-based detection when content is unclear
	if hasIssueListURLPattern {
		return "issueList"
	}
	if hasBoardURLPattern || hasBoardLayout {
		return "board"
	}
	if hasSearchURLPattern {
		return "search"
	}

	// Check if it's at least a Jira page
	for _, indicator := range indicators {
		if indicator == "url_pattern:jira_domain" {
			return "generic"
		}
	}

	return "unknown"
}

// calculateConfidence determines confidence level based on number and quality of indicators
func (pa *pageAssessor) calculateConfidence(indicators []string) string {
	if len(indicators) == 0 {
		return "none"
	}

	// Count URL and HTML indicators separately
	urlIndicators := 0
	htmlIndicators := 0

	for _, indicator := range indicators {
		if strings.HasPrefix(indicator, "url_pattern:") {
			urlIndicators++
		}
		if strings.HasPrefix(indicator, "html_structure:") {
			htmlIndicators++
		}
	}

	// High confidence: both URL and HTML indicators present
	if urlIndicators > 0 && htmlIndicators > 0 {
		return "high"
	}

	// Medium confidence: either URL or HTML indicators (but not both)
	if urlIndicators > 0 || htmlIndicators > 0 {
		return "medium"
	}

	return "low"
}

// getPageDescription returns a human-readable description of the page type
func (pa *pageAssessor) getPageDescription(pageType string) string {
	descriptions := map[string]string{
		"projectsList": "Jira Projects Directory - Lists all available projects",
		"issue":        "Jira Issue Detail - Single ticket with full details",
		"issueList":    "Jira Issue List - Multiple tickets in a project",
		"board":        "Jira Board - Kanban or Scrum board view",
		"search":       "Jira Search Results - Filtered ticket list",
		"generic":      "Generic Jira Page - May contain ticket references",
		"unknown":      "Unknown Page Type - Not a recognized Jira page",
	}

	if desc, ok := descriptions[pageType]; ok {
		return desc
	}
	return "Unknown page type"
}

// isCollectable determines if the page can be collected based on type and confidence
func (pa *pageAssessor) isCollectable(pageType, confidence string) bool {
	// Define collectable page types
	collectableTypes := map[string]bool{
		"projectsList": true,
		"issue":        true,
		"issueList":    true,
		"board":        true,
		"search":       true,
		"generic":      false, // Generic pages are not collectable
		"unknown":      false, // Unknown pages are not collectable
	}

	isKnownType := collectableTypes[pageType]

	// For auto-collection, we want to be more permissive
	// Allow low confidence for known page types if we have clear URL indicators
	// This ensures auto-collection works even when HTML parsing is incomplete
	if isKnownType {
		// High or medium confidence: always collect
		if confidence == "high" || confidence == "medium" {
			return true
		}
		// Low confidence: still collect for known types (URL-based detection is often sufficient)
		// This handles cases where page is still loading or HTML is dynamic
		if confidence == "low" {
			pa.logger.Debug().
				Str("page_type", pageType).
				Str("confidence", confidence).
				Msg("Allowing collection with low confidence for known page type")
			return true
		}
	}

	return false
}
