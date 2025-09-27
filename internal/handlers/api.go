package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	. "aktis-collector-jira/internal/common"
	. "aktis-collector-jira/internal/interfaces"
	"github.com/ternarybob/arbor"
)

// APIHandlers contains all API endpoint handlers
type APIHandlers struct {
	config    *Config
	collector Collector
	storage   Storage
	logger    arbor.ILogger
	startTime time.Time
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Build     string    `json:"build"`
	Uptime    float64   `json:"uptime_seconds"`
	Services  struct {
		Database bool `json:"database"`
		Jira     bool `json:"jira"`
	} `json:"services"`
}

// StatusResponse represents the collector status response
type StatusResponse struct {
	Collector struct {
		Running    bool      `json:"running"`
		Uptime     float64   `json:"uptime"`
		ErrorCount int       `json:"error_count"`
		LastRun    time.Time `json:"last_run,omitempty"`
	} `json:"collector"`
	Projects []ProjectStatus `json:"projects"`
	Stats    CollectorStats  `json:"stats"`
}

// ProjectStatus represents the status of a single project
type ProjectStatus struct {
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	TicketCount int       `json:"ticket_count"`
	LastUpdate  time.Time `json:"last_update,omitempty"`
	Status      string    `json:"status"`
}

// CollectorStats represents overall collector statistics
type CollectorStats struct {
	TotalTickets   int    `json:"total_tickets"`
	LastCollection string `json:"last_collection"`
	DatabaseSize   string `json:"database_size"`
}

// ConfigResponse represents the configuration display response
type ConfigResponse struct {
	Collector *CollectorConfig `json:"collector"`
	Jira      *JiraConfig      `json:"jira"`
	Projects  []ProjectConfig  `json:"projects"`
	Storage   *StorageConfig   `json:"storage"`
	Logging   *LoggingConfig   `json:"logging"`
}

// DatabaseResponse represents database operation responses
type DatabaseResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Count   int    `json:"count,omitempty"`
}

// NewAPIHandlers creates a new API handlers instance
func NewAPIHandlers(config *Config, collector Collector, storage Storage, logger arbor.ILogger) *APIHandlers {
	return &APIHandlers{
		config:    config,
		collector: collector,
		storage:   storage,
		logger:    logger,
		startTime: time.Now(),
	}
}

// HealthHandler returns system health status
func (h *APIHandlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   GetVersion(),
		Build:     GetBuild(),
		Uptime:    time.Since(h.startTime).Seconds(),
	}

	// Test database connection
	health.Services.Database = h.testDatabaseConnection()

	// Test Jira connection (basic check)
	health.Services.Jira = h.testJiraConnection()

	// If any service is down, mark as degraded
	if !health.Services.Database || !health.Services.Jira {
		health.Status = "degraded"
	}

	if err := json.NewEncoder(w).Encode(health); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode health response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// StatusHandler returns collector status and metrics
func (h *APIHandlers) StatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := StatusResponse{
		Projects: make([]ProjectStatus, 0, len(h.config.Projects)),
		Stats: CollectorStats{
			DatabaseSize: "N/A",
		},
	}

	// Collector status
	status.Collector.Running = true // Assume running if we can respond
	status.Collector.Uptime = time.Since(h.startTime).Seconds()
	status.Collector.ErrorCount = 0 // TODO: Track actual errors

	// Project status
	totalTickets := 0
	var lastUpdate time.Time

	for _, project := range h.config.Projects {
		projectStatus := ProjectStatus{
			Key:    project.Key,
			Name:   project.Name,
			Status: "active",
		}

		// Load tickets to get count and last update
		tickets, err := h.storage.LoadTickets(project.Key)
		if err != nil {
			h.logger.Warn().Err(err).Str("project", project.Key).Msg("Failed to load project tickets")
			projectStatus.Status = "error"
		} else {
			projectStatus.TicketCount = len(tickets)
			totalTickets += len(tickets)

			// Find most recent update
			for _, ticket := range tickets {
				if ticket.Updated != "" {
					if ticketTime, err := time.Parse(time.RFC3339, ticket.Updated); err == nil {
						if ticketTime.After(projectStatus.LastUpdate) {
							projectStatus.LastUpdate = ticketTime
						}
					}
				}
			}

			if projectStatus.LastUpdate.After(lastUpdate) {
				lastUpdate = projectStatus.LastUpdate
			}
		}

		status.Projects = append(status.Projects, projectStatus)
	}

	status.Stats.TotalTickets = totalTickets
	if !lastUpdate.IsZero() {
		status.Stats.LastCollection = lastUpdate.Format("2006-01-02 15:04:05")
	} else {
		status.Stats.LastCollection = "Never"
	}

	if err := json.NewEncoder(w).Encode(status); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode status response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ConfigHandler returns system configuration
func (h *APIHandlers) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Create sanitized config (remove sensitive data)
	config := ConfigResponse{
		Collector: &h.config.Collector,
		Projects:  h.config.Projects,
		Storage:   &h.config.Storage,
		Logging:   &h.config.Logging,
		Jira: &JiraConfig{
			BaseURL:  h.config.Jira.BaseURL,
			Username: h.config.Jira.Username,
			APIToken: "***REDACTED***", // Don't expose API token
			Timeout:  h.config.Jira.Timeout,
		},
	}

	if err := json.NewEncoder(w).Encode(config); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode config response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// DatabaseHandler handles database operations
func (h *APIHandlers) DatabaseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		h.handleGetDatabase(w, r)
	case http.MethodDelete:
		h.handleClearDatabase(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *APIHandlers) handleGetDatabase(w http.ResponseWriter, r *http.Request) {
	allTickets := make(map[string]interface{})
	totalCount := 0

	for _, project := range h.config.Projects {
		tickets, err := h.storage.LoadTickets(project.Key)
		if err != nil {
			h.logger.Warn().Err(err).Str("project", project.Key).Msg("Failed to load tickets")
			continue
		}

		if len(tickets) > 0 {
			allTickets[project.Key] = tickets
			totalCount += len(tickets)
		}
	}

	response := DatabaseResponse{
		Success: true,
		Message: fmt.Sprintf("Retrieved %d tickets from %d projects", totalCount, len(allTickets)),
		Count:   totalCount,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode database response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *APIHandlers) handleClearDatabase(w http.ResponseWriter, r *http.Request) {
	errors := []string{}
	clearedCount := 0

	for _, project := range h.config.Projects {
		// Get current count before clearing
		tickets, err := h.storage.LoadTickets(project.Key)
		if err == nil {
			clearedCount += len(tickets)
		}

		// Clear the project data
		err = h.storage.SaveTickets(project.Key, make(map[string]*TicketData))
		if err != nil {
			h.logger.Error().Err(err).Str("project", project.Key).Msg("Failed to clear project data")
			errors = append(errors, fmt.Sprintf("Failed to clear %s: %v", project.Key, err))
		}
	}

	if len(errors) > 0 {
		response := DatabaseResponse{
			Success: false,
			Message: fmt.Sprintf("Partially cleared database with errors: %v", errors),
		}
		w.WriteHeader(http.StatusPartialContent)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := DatabaseResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully cleared %d tickets from database", clearedCount),
		Count:   clearedCount,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode database response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *APIHandlers) testDatabaseConnection() bool {
	// Test by trying to load tickets from the first project
	if len(h.config.Projects) == 0 {
		return false
	}

	_, err := h.storage.LoadTickets(h.config.Projects[0].Key)
	return err == nil
}

func (h *APIHandlers) testJiraConnection() bool {
	// Basic test - check if Jira config is valid
	return h.config.Jira.BaseURL != "" &&
		h.config.Jira.Username != "" &&
		h.config.Jira.APIToken != "" &&
		h.config.Jira.APIToken != "your-api-token"
}
