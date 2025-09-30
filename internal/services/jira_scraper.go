package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "aktis-collector-jira/internal/common"
	. "aktis-collector-jira/internal/interfaces"

	"github.com/chromedp/chromedp"
)

type jiraScraper struct {
	config  *JiraConfig
	ctx     context.Context
	cancel  context.CancelFunc
	timeout time.Duration
}

func NewJiraScraper(config *JiraConfig) (JiraScraper, error) {
	// Always connect to existing browser via remote debugging
	// User must start Chrome with: chrome.exe --remote-debugging-port=9222
	debugURL := fmt.Sprintf("http://localhost:%d", config.ScraperConfig.RemoteDebugPort)

	allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), debugURL)
	ctx, cancel := chromedp.NewContext(allocCtx)

	wrappedCancel := func() {
		cancel()
		allocCancel()
	}

	return &jiraScraper{
		config:  config,
		ctx:     ctx,
		cancel:  wrappedCancel,
		timeout: time.Duration(config.Timeout) * time.Second,
	}, nil
}

func (js *jiraScraper) Close() error {
	if js.cancel != nil {
		js.cancel()
	}
	return nil
}

func (js *jiraScraper) ScrapeProject(projectKey string, batchSize int) ([]*TicketData, error) {
	var tickets []*TicketData

	url := fmt.Sprintf("%s/browse/%s", js.config.BaseURL, projectKey)

	ctx, cancel := context.WithTimeout(js.ctx, js.timeout)
	defer cancel()

	var issueKeys []string

	actions := []chromedp.Action{
		chromedp.Navigate(url),
	}

	if js.config.ScraperConfig.WaitBeforeScrape > 0 {
		actions = append(actions, chromedp.Sleep(time.Duration(js.config.ScraperConfig.WaitBeforeScrape)*time.Millisecond))
	}

	actions = append(actions,
		chromedp.WaitVisible(`[data-test-id="issue.views.issue-base.foundation.breadcrumbs.breadcrumb-current-issue-container"]`, chromedp.ByQuery),
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('[data-testid*="issue"]')).map(el => {
				const key = el.textContent.match(/[A-Z]+-\d+/);
				return key ? key[0] : null;
			}).filter(k => k !== null);
		`, &issueKeys))

	err := chromedp.Run(ctx, actions...)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape project list: %w", err)
	}

	for _, key := range issueKeys {
		ticket, err := js.scrapeTicket(key)
		if err != nil {
			continue
		}
		tickets = append(tickets, ticket)

		if len(tickets) >= batchSize {
			break
		}
	}

	return tickets, nil
}

func (js *jiraScraper) scrapeTicket(issueKey string) (*TicketData, error) {
	url := fmt.Sprintf("%s/browse/%s", js.config.BaseURL, issueKey)

	ctx, cancel := context.WithTimeout(js.ctx, js.timeout)
	defer cancel()

	var summary, description, issueType, status, priority, created, updated, reporter, assignee string
	var labels, components []string

	actions := []chromedp.Action{
		chromedp.Navigate(url),
	}

	if js.config.ScraperConfig.WaitBeforeScrape > 0 {
		actions = append(actions, chromedp.Sleep(time.Duration(js.config.ScraperConfig.WaitBeforeScrape)*time.Millisecond))
	}

	actions = append(actions,
		chromedp.WaitVisible(`[data-test-id="issue.views.issue-base.foundation.summary.heading"]`, chromedp.ByQuery),

		chromedp.Text(`[data-test-id="issue.views.issue-base.foundation.summary.heading"]`, &summary, chromedp.NodeVisible, chromedp.ByQuery),

		chromedp.Text(`[data-testid="issue.views.field.rich-text.description"]`, &description, chromedp.NodeVisible, chromedp.ByQuery),

		chromedp.Text(`[data-testid="issue.views.field.issue-type.common.ui.issue-type-select"]`, &issueType, chromedp.NodeVisible, chromedp.ByQuery),

		chromedp.Text(`[data-testid="issue.views.field.status.common.ui.status-lozenge"]`, &status, chromedp.NodeVisible, chromedp.ByQuery),

		chromedp.Text(`[data-testid="issue.views.field.priority.common.ui.priority-lozenge"]`, &priority, chromedp.NodeVisible, chromedp.ByQuery),

		chromedp.Text(`[data-testid="issue.views.field.created.common.ui.date-renderer"]`, &created, chromedp.NodeVisible, chromedp.ByQuery),

		chromedp.Text(`[data-testid="issue.views.field.updated.common.ui.date-renderer"]`, &updated, chromedp.NodeVisible, chromedp.ByQuery),

		chromedp.Text(`[data-testid="issue.views.field.reporter.common.ui.read-view.assignee-renderer"]`, &reporter, chromedp.NodeVisible, chromedp.ByQuery),

		chromedp.Text(`[data-testid="issue.views.field.assignee.common.ui.read-view.assignee-renderer"]`, &assignee, chromedp.NodeVisible, chromedp.ByQuery),

		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('[data-testid="issue.views.field.labels.common.ui.labels-view"] span')).map(el => el.textContent);
		`, &labels),

		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('[data-testid="issue.views.field.components.common.ui.read-view"] span')).map(el => el.textContent);
		`, &components))

	err := chromedp.Run(ctx, actions...)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape ticket %s: %w", issueKey, err)
	}

	return &TicketData{
		Key:          issueKey,
		Summary:      strings.TrimSpace(summary),
		Description:  strings.TrimSpace(description),
		IssueType:    strings.TrimSpace(issueType),
		Status:       strings.TrimSpace(status),
		Priority:     strings.TrimSpace(priority),
		Created:      strings.TrimSpace(created),
		Updated:      strings.TrimSpace(updated),
		Reporter:     strings.TrimSpace(reporter),
		Assignee:     strings.TrimSpace(assignee),
		Labels:       labels,
		Components:   components,
		CustomFields: make(map[string]interface{}),
	}, nil
}
