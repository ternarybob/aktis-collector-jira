package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	. "aktis-collector-jira/internal/common"
	. "aktis-collector-jira/internal/interfaces"

	plugin "github.com/ternarybob/aktis-plugin-sdk"
)

type collector struct {
	config  *Config
	client  JiraClient
	scraper JiraScraper
	storage Storage
}

func NewCollector(config *Config, storage Storage) (Collector, error) {
	c := &collector{
		config:  config,
		storage: storage,
	}

	// Only initialize API client at startup (lightweight)
	// Scraper will be lazily initialized when first needed (to avoid opening browser)
	if config.UsesAPI() {
		c.client = NewJiraClient(&config.Jira)
	}

	return c, nil
}

func (c *collector) Close() error {
	if c.scraper != nil {
		c.scraper.Close()
	}
	if c.storage != nil {
		return c.storage.Close()
	}
	return nil
}

func (c *collector) CollectAllTickets(batchSize int) ([]plugin.Payload, error) {
	var allPayloads []plugin.Payload

	for _, project := range c.config.Projects {
		payloads, err := c.collectProjectTickets(project, batchSize)
		if err != nil {
			return nil, fmt.Errorf("failed to collect tickets for project %s: %w", project.Key, err)
		}
		allPayloads = append(allPayloads, payloads...)
	}

	sendLimit := c.config.Collector.SendLimit
	if sendLimit > 0 && len(allPayloads) > sendLimit {
		allPayloads = allPayloads[:sendLimit]
	}

	return allPayloads, nil
}

func (c *collector) CollectWithMethod(method string, batchSize int) ([]plugin.Payload, error) {
	var allPayloads []plugin.Payload

	for _, project := range c.config.Projects {
		payloads, err := c.collectProjectTicketsWithMethod(project, batchSize, method)
		if err != nil {
			return nil, fmt.Errorf("failed to collect tickets for project %s: %w", project.Key, err)
		}
		allPayloads = append(allPayloads, payloads...)
	}

	sendLimit := c.config.Collector.SendLimit
	if sendLimit > 0 && len(allPayloads) > sendLimit {
		allPayloads = allPayloads[:sendLimit]
	}

	return allPayloads, nil
}

func (c *collector) collectProjectTickets(project ProjectConfig, batchSize int) ([]plugin.Payload, error) {
	return c.collectProjectTicketsWithMethod(project, batchSize, c.config.GetPrimaryMethod())
}

func (c *collector) collectProjectTicketsWithMethod(project ProjectConfig, batchSize int, method string) ([]plugin.Payload, error) {
	var allPayloads []plugin.Payload
	ticketsData := make(map[string]*TicketData)

	if method == "scraper" {
		// Create scraper instance on-demand (connects to existing browser)
		scraper, err := NewJiraScraper(&c.config.Jira)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to browser: %w", err)
		}
		defer scraper.Close()

		tickets, err := scraper.ScrapeProject(project.Key, project.MaxResults)
		if err != nil {
			return nil, err
		}

		for _, ticket := range tickets {
			ticketsData[ticket.Key] = ticket

			payload := plugin.Payload{
				Timestamp: time.Now(),
				Type:      fmt.Sprintf("jira_%s", strings.ToLower(strings.ReplaceAll(ticket.IssueType, " ", "_"))),
				Data:      c.ticketDataToMap(ticket),
				Metadata: map[string]string{
					"project":   project.Key,
					"ticket_id": ticket.Key,
					"source":    "jira_scraper",
					"mode":      "full_collection",
				},
			}
			allPayloads = append(allPayloads, payload)
		}
	} else {
		jql := c.client.BuildJQL(project.Key, project.IssueTypes, project.Statuses, "")
		startAt := 0

		for {
			response, err := c.client.SearchIssues(jql, project.MaxResults, startAt)
			if err != nil {
				return nil, err
			}

			for _, issue := range response.Issues {
				ticket := c.issueToTicketData(&issue, project.Key)
				ticketsData[ticket.Key] = &ticket

				payload := plugin.Payload{
					Timestamp: time.Now(),
					Type:      fmt.Sprintf("jira_%s", c.getIssueType(&issue)),
					Data:      c.extractIssueData(&issue),
					Metadata: map[string]string{
						"project":   project.Key,
						"ticket_id": issue.Key,
						"source":    "jira_api",
						"mode":      "full_collection",
					},
				}
				allPayloads = append(allPayloads, payload)
			}

			if startAt+len(response.Issues) >= response.Total {
				break
			}

			startAt += batchSize
		}
	}

	if err := c.storage.SaveTickets(project.Key, ticketsData); err != nil {
		return nil, fmt.Errorf("failed to save tickets for project %s: %w", project.Key, err)
	}

	return allPayloads, nil
}

func (c *collector) issueToTicketData(issue *Issue, projectKey string) TicketData {
	data := c.extractIssueData(issue)
	hash := c.generateDataHash(data)

	return TicketData{
		Key:          issue.Key,
		Summary:      c.getString(issue.Fields["summary"]),
		Description:  c.getString(issue.Fields["description"]),
		IssueType:    c.getIssueType(issue),
		Status:       c.getStatus(issue),
		Priority:     c.getPriority(issue),
		Created:      c.getString(issue.Fields["created"]),
		Updated:      c.getString(issue.Fields["updated"]),
		Reporter:     c.getPersonName(issue.Fields["reporter"]),
		Assignee:     c.getPersonName(issue.Fields["assignee"]),
		Labels:       c.getLabels(issue.Fields["labels"]),
		Components:   c.getComponents(issue.Fields["components"]),
		CustomFields: data,
		Hash:         hash,
	}
}

func (c *collector) extractIssueData(issue *Issue) map[string]interface{} {
	data := make(map[string]interface{})

	data["key"] = issue.Key

	if summary, ok := issue.Fields["summary"].(string); ok {
		data["summary"] = summary
	}

	if description, ok := issue.Fields["description"].(string); ok {
		data["description"] = description
	}

	if issueType, ok := issue.Fields["issuetype"].(map[string]interface{}); ok {
		if name, ok := issueType["name"].(string); ok {
			data["issue_type"] = name
		}
	}

	if status, ok := issue.Fields["status"].(map[string]interface{}); ok {
		if name, ok := status["name"].(string); ok {
			data["status"] = name
		}
	}

	if priority, ok := issue.Fields["priority"].(map[string]interface{}); ok {
		if name, ok := priority["name"].(string); ok {
			data["priority"] = name
		}
	}

	if assignee, ok := issue.Fields["assignee"].(map[string]interface{}); ok {
		if name, ok := assignee["displayName"].(string); ok {
			data["assignee"] = name
		} else if email, ok := assignee["emailAddress"].(string); ok {
			data["assignee"] = email
		}
	}

	if reporter, ok := issue.Fields["reporter"].(map[string]interface{}); ok {
		if name, ok := reporter["displayName"].(string); ok {
			data["reporter"] = name
		} else if email, ok := reporter["emailAddress"].(string); ok {
			data["reporter"] = email
		}
	}

	if created, ok := issue.Fields["created"].(string); ok {
		data["created"] = created
	}

	if updated, ok := issue.Fields["updated"].(string); ok {
		data["updated"] = updated
	}

	if project, ok := issue.Fields["project"].(map[string]interface{}); ok {
		if key, ok := project["key"].(string); ok {
			data["project_key"] = key
		}
		if name, ok := project["name"].(string); ok {
			data["project_name"] = name
		}
	}

	if labels, ok := issue.Fields["labels"].([]interface{}); ok {
		data["labels"] = labels
	}

	if components, ok := issue.Fields["components"].([]interface{}); ok {
		data["components"] = components
	}

	data["raw_fields"] = issue.Fields

	return data
}

func (c *collector) getIssueType(issue *Issue) string {
	if issueType, ok := issue.Fields["issuetype"].(map[string]interface{}); ok {
		if name, ok := issueType["name"].(string); ok {
			return strings.ToLower(strings.ReplaceAll(name, " ", "_"))
		}
	}
	return "unknown"
}

func (c *collector) getStatus(issue *Issue) string {
	if status, ok := issue.Fields["status"].(map[string]interface{}); ok {
		if name, ok := status["name"].(string); ok {
			return name
		}
	}
	return ""
}

func (c *collector) getPriority(issue *Issue) string {
	if priority, ok := issue.Fields["priority"].(map[string]interface{}); ok {
		if name, ok := priority["name"].(string); ok {
			return name
		}
	}
	return ""
}

func (c *collector) getPersonName(field interface{}) string {
	if person, ok := field.(map[string]interface{}); ok {
		if name, ok := person["displayName"].(string); ok {
			return name
		} else if email, ok := person["emailAddress"].(string); ok {
			return email
		}
	}
	return ""
}

func (c *collector) getLabels(field interface{}) []string {
	if labels, ok := field.([]interface{}); ok {
		result := make([]string, 0, len(labels))
		for _, label := range labels {
			if str, ok := label.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return []string{}
}

func (c *collector) getComponents(field interface{}) []string {
	if components, ok := field.([]interface{}); ok {
		result := make([]string, 0, len(components))
		for _, component := range components {
			if comp, ok := component.(map[string]interface{}); ok {
				if name, ok := comp["name"].(string); ok {
					result = append(result, name)
				}
			}
		}
		return result
	}
	return []string{}
}

func (c *collector) getString(field interface{}) string {
	if str, ok := field.(string); ok {
		return str
	}
	return ""
}

func (c *collector) generateDataHash(data map[string]interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:8])
}

func (c *collector) ticketDataToMap(ticket *TicketData) map[string]interface{} {
	data := make(map[string]interface{})

	data["key"] = ticket.Key
	data["summary"] = ticket.Summary
	data["description"] = ticket.Description
	data["issue_type"] = ticket.IssueType
	data["status"] = ticket.Status
	data["priority"] = ticket.Priority
	data["created"] = ticket.Created
	data["updated"] = ticket.Updated
	data["reporter"] = ticket.Reporter
	data["assignee"] = ticket.Assignee
	data["labels"] = ticket.Labels
	data["components"] = ticket.Components
	data["hash"] = ticket.Hash

	for key, value := range ticket.CustomFields {
		data[key] = value
	}

	return data
}
