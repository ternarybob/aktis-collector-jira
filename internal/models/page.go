package models

// PageAssessment represents the result of analyzing a Jira page
type PageAssessment struct {
	PageType    string   `json:"page_type"`
	Confidence  string   `json:"confidence"`
	Description string   `json:"description"`
	Indicators  []string `json:"indicators"`
	Collectable bool     `json:"collectable"`
}
