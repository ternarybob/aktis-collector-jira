package models

// ProjectData represents a Jira project
type ProjectData struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Updated     string `json:"updated"`
}
