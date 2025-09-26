package collector

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	plugin "github.com/ternarybob/aktis-plugin-sdk"
)

// JiraCollector handles the main collection logic
type JiraCollector struct {
	config    *Config
	client    *JiraClient
	storage   *Storage
	startTime time.Time
}

// NewJiraCollector creates a new Jira collector instance
func NewJiraCollector(config *Config) (*JiraCollector, error) {
	storage, err := NewStorage(&config.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	return &JiraCollector{
		config:  config,
		client:  NewJiraClient(&config.Jira),
		storage: storage,
	}, nil
}

// CollectAllTickets collects all tickets from configured projects in batches
func (jc *JiraCollector) CollectAllTickets(batchSize int) ([]plugin.Payload, error) {
	jc.startTime = time.Now()
	var allPayloads []plugin.Payload

	for _, project := range jc.config.Projects {
		payloads, err := jc.collectProjectTickets(project, batchSize)
		if err != nil {
			return nil, fmt.Errorf("failed to collect tickets for project %s: %w", project.Key, err)
		}
		allPayloads = append(allPayloads, payloads...)
	}

	return allPayloads, nil
}

// UpdateTickets collects only tickets that have been updated since last collection
func (jc *JiraCollector) UpdateTickets(batchSize int) ([]plugin.Payload, error) {
	jc.startTime = time.Now()
	var allPayloads []plugin.Payload

	for _, project := range jc.config.Projects {
		// Get last update time for this project
		lastUpdate, err := jc.storage.GetLastUpdateTime(project.Key)
		if err != nil {
			// If no previous data, collect all tickets
			payloads, err := jc.collectProjectTickets(project, batchSize)
			if err != nil {
				return nil, fmt.Errorf("failed to collect tickets for project %s: %w", project.Key, err)
			}
			allPayloads = append(allPayloads, payloads...)
		} else {
			// Collect only updated tickets
			payloads, err := jc.collectUpdatedTickets(project, lastUpdate, batchSize)
			if err != nil {
				return nil, fmt.Errorf("failed to collect updated tickets for project %s: %w", project.Key, err)
			}
			allPayloads = append(allPayloads, payloads...)
		}
	}

	return allPayloads, nil
}

func (jc *JiraCollector) collectProjectTickets(project ProjectConfig, batchSize int) ([]plugin.Payload, error) {
	var allPayloads []plugin.Payload
	var allTickets []TicketData

	jql := BuildJQL(project, time.Time{})
	startAt := 0

	for {
		response, err := jc.client.SearchIssues(jql, startAt, batchSize)
		if err != nil {
			return nil, err
		}

		// Convert issues to tickets and payloads
		for _, issue := range response.Issues {
			ticket := jc.issueToTicketData(&issue, project.Key)
			allTickets = append(allTickets, ticket)

			payload := plugin.Payload{
				Timestamp: time.Now(),
				Type:      fmt.Sprintf("jira_%s", jc.getIssueType(&issue)),
				Data:      ticket.Data,
				Metadata: map[string]string{
					"project":   project.Key,
					"ticket_id": issue.Key,
					"source":    "jira",
					"mode":      "full_collection",
				},
			}
			allPayloads = append(allPayloads, payload)
		}

		// Check if we've collected all issues
		if startAt+len(response.Issues) >= response.Total {
			break
		}

		startAt += batchSize
	}

	// Save tickets to storage
	if err := jc.storage.SaveTickets(project.Key, allTickets); err != nil {
		return nil, fmt.Errorf("failed to save tickets for project %s: %w", project.Key, err)
	}

	return allPayloads, nil
}

func (jc *JiraCollector) collectUpdatedTickets(project ProjectConfig, since time.Time, batchSize int) ([]plugin.Payload, error) {
	var allPayloads []plugin.Payload
	var allTickets []TicketData

	jql := BuildJQL(project, since)
	startAt := 0

	for {
		response, err := jc.client.SearchIssues(jql, startAt, batchSize)
		if err != nil {
			return nil, err
		}

		// Convert issues to tickets and payloads
		for _, issue := range response.Issues {
			ticket := jc.issueToTicketData(&issue, project.Key)
			allTickets = append(allTickets, ticket)

			payload := plugin.Payload{
				Timestamp: time.Now(),
				Type:      fmt.Sprintf("jira_%s", jc.getIssueType(&issue)),
				Data:      ticket.Data,
				Metadata: map[string]string{
					"project":   project.Key,
					"ticket_id": issue.Key,
					"source":    "jira",
					"mode":      "update",
					"updated_since": since.Format(time.RFC3339),
				},
			}
			allPayloads = append(allPayloads, payload)
		}

		// Check if we've collected all issues
		if startAt+len(response.Issues) >= response.Total {
			break
		}

		startAt += batchSize
	}

	// Save tickets to storage
	if len(allTickets) > 0 {
		if err := jc.storage.SaveTickets(project.Key, allTickets); err != nil {
			return nil, fmt.Errorf("failed to save updated tickets for project %s: %w", project.Key, err)
		}
	}

	return allPayloads, nil
}

func (jc *JiraCollector) issueToTicketData(issue *JiraIssue, projectKey string) TicketData {
	data := ExtractIssueData(issue)
	
	// Generate hash for change detection
	hash := jc.generateDataHash(data)
	
	return TicketData{
		Key:     issue.Key,
		Project: projectKey,
		Data:    data,
		Hash:    hash,
	}
}

func (jc *JiraCollector) getIssueType(issue *JiraIssue) string {
	if issueType, ok := issue.Fields["issuetype"].(map[string]interface{}); ok {
		if name, ok := issueType["name"].(string); ok {
			return strings.ToLower(strings.ReplaceAll(name, " ", "_"))
		}
	}
	return "unknown"
}

func (jc *JiraCollector) generateDataHash(data map[string]interface{}) string {
	// Convert data to JSON for consistent hashing
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:8]) // Use first 8 characters
}

// GetCollectionStats returns statistics about the collected data
func (jc *JiraCollector) GetCollectionStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	for _, project := range jc.config.Projects {
		dataset, err := jc.storage.LoadProjectDataset(project.Key)
		if err != nil {
			continue
		}
		
		stats[project.Key] = map[string]interface{}{
			"total_tickets": dataset.TotalCount,
			"last_update":   dataset.LastUpdate,
		}
	}
	
	return stats, nil
}

// Cleanup performs maintenance tasks
func (jc *JiraCollector) Cleanup() error {
	return jc.storage.CleanupOldData()
}