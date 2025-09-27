package interfaces

import (
	"context"
	plugin "github.com/ternarybob/aktis-plugin-sdk"
)

type JiraClient interface {
	SearchIssues(jql string, maxResults, startAt int) (*SearchResponse, error)
	BuildJQL(projectKey string, issueTypes, statuses []string, updatedAfter string) string
}

type SearchResponse struct {
	Issues     []Issue `json:"issues"`
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
}

type Issue struct {
	Key    string                 `json:"key"`
	Fields map[string]interface{} `json:"fields"`
}

type Collector interface {
	CollectAllTickets(batchSize int) ([]plugin.Payload, error)
	Close() error
}

type Storage interface {
	SaveTickets(projectKey string, tickets map[string]*TicketData) error
	LoadTickets(projectKey string) (map[string]*TicketData, error)
	GetLastUpdate(projectKey string) (string, error)
	Close() error
}

type TicketData struct {
	Key          string                 `json:"key"`
	Summary      string                 `json:"summary"`
	Description  string                 `json:"description"`
	IssueType    string                 `json:"issue_type"`
	Status       string                 `json:"status"`
	Priority     string                 `json:"priority"`
	Created      string                 `json:"created"`
	Updated      string                 `json:"updated"`
	Reporter     string                 `json:"reporter"`
	Assignee     string                 `json:"assignee"`
	Labels       []string               `json:"labels"`
	Components   []string               `json:"components"`
	CustomFields map[string]interface{} `json:"custom_fields"`
	Hash         string                 `json:"hash"`
}

type WebService interface {
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool
}
