package models

// TicketData represents a Jira ticket/issue with comprehensive details
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

// Comment represents a ticket comment
type Comment struct {
	ID      string `json:"id"`
	Author  string `json:"author"`
	Body    string `json:"body"`
	Created string `json:"created"`
	Updated string `json:"updated"`
}

// Subtask represents a subtask of a parent ticket
type Subtask struct {
	Key       string `json:"key"`
	Summary   string `json:"summary"`
	Status    string `json:"status"`
	IssueType string `json:"issue_type"`
	URL       string `json:"url"`
}

// Attachment represents a file attached to a ticket
type Attachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
	Created  string `json:"created"`
	Author   string `json:"author"`
	URL      string `json:"url"`
}

// IssueLink represents a link between two issues
type IssueLink struct {
	LinkType     string `json:"link_type"`
	Direction    string `json:"direction"` // inward or outward
	IssueKey     string `json:"issue_key"`
	IssueSummary string `json:"issue_summary"`
	URL          string `json:"url"`
}

// WorkLogEntry represents a work log entry for time tracking
type WorkLogEntry struct {
	ID        string `json:"id"`
	Author    string `json:"author"`
	TimeSpent string `json:"time_spent"`
	Comment   string `json:"comment"`
	Created   string `json:"created"`
	Updated   string `json:"updated"`
}
