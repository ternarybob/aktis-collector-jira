package interfaces

import (
	"context"
)

type Storage interface {
	SaveTickets(projectKey string, tickets map[string]*TicketData) error
	LoadTickets(projectKey string) (map[string]*TicketData, error)
	LoadAllTickets() (map[string]*TicketData, error)
	ClearAllTickets() error
	ClearAllProjects() error
	GetLastUpdate(projectKey string) (string, error)
	SaveProjects(projects []*ProjectData) error
	LoadProjects() ([]*ProjectData, error)
	Close() error
}

type TicketData struct {
	Key          string                 `json:"key"`
	ProjectID    string                 `json:"project_id"`
	URL          string                 `json:"url"`
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

	// Extended fields for comprehensive ticket details
	Comments    []Comment      `json:"comments,omitempty"`
	Subtasks    []Subtask      `json:"subtasks,omitempty"`
	Attachments []Attachment   `json:"attachments,omitempty"`
	Links       []IssueLink    `json:"links,omitempty"`
	WorkLog     []WorkLogEntry `json:"worklog,omitempty"`
	RawHTML     string         `json:"raw_html,omitempty"` // Keep raw HTML for future parsing

	Hash string `json:"hash"`
}

type Comment struct {
	ID      string `json:"id"`
	Author  string `json:"author"`
	Body    string `json:"body"`
	Created string `json:"created"`
	Updated string `json:"updated"`
}

type Subtask struct {
	Key       string `json:"key"`
	Summary   string `json:"summary"`
	Status    string `json:"status"`
	IssueType string `json:"issue_type"`
	URL       string `json:"url"`
}

type Attachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
	Created  string `json:"created"`
	Author   string `json:"author"`
	URL      string `json:"url"`
}

type IssueLink struct {
	LinkType     string `json:"link_type"`
	Direction    string `json:"direction"` // inward or outward
	IssueKey     string `json:"issue_key"`
	IssueSummary string `json:"issue_summary"`
	URL          string `json:"url"`
}

type WorkLogEntry struct {
	ID        string `json:"id"`
	Author    string `json:"author"`
	TimeSpent string `json:"time_spent"`
	Comment   string `json:"comment"`
	Created   string `json:"created"`
	Updated   string `json:"updated"`
}

type ProjectData struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Updated     string `json:"updated"`
}

type WebService interface {
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool
}

type PageAssessor interface {
	AssessPage(htmlContent, url string) (*PageAssessment, error)
}

type PageAssessment struct {
	PageType    string   `json:"page_type"`
	Confidence  string   `json:"confidence"`
	Description string   `json:"description"`
	Indicators  []string `json:"indicators"`
	Collectable bool     `json:"collectable"`
}
