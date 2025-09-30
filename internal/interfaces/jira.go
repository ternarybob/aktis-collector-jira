package interfaces

import (
	"context"

	"aktis-collector-jira/internal/models"
)

// Storage defines the interface for persistent data storage operations
type Storage interface {
	SaveTickets(projectKey string, tickets map[string]*models.TicketData) error
	LoadTickets(projectKey string) (map[string]*models.TicketData, error)
	LoadAllTickets() (map[string]*models.TicketData, error)
	ClearAllTickets() error
	ClearAllProjects() error
	GetLastUpdate(projectKey string) (string, error)
	SaveProjects(projects []*models.ProjectData) error
	LoadProjects() ([]*models.ProjectData, error)
	Close() error
}

// WebService defines the interface for web server operations
type WebService interface {
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool
}

// PageAssessor defines the interface for analyzing web page types
type PageAssessor interface {
	AssessPage(htmlContent, url string) (*models.PageAssessment, error)
}
