package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	. "aktis-collector-jira/internal/common"
	. "aktis-collector-jira/internal/interfaces"
	plugin "github.com/ternarybob/aktis-plugin-sdk"
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
			Method:  h.config.Jira.Method,
			BaseURL: h.config.Jira.BaseURL,
			Timeout: h.config.Jira.Timeout,
			APIConfig: APIConfig{
				Username: h.config.Jira.APIConfig.Username,
				APIToken: "***REDACTED***", // Don't expose API token
			},
			ScraperConfig: h.config.Jira.ScraperConfig,
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
	if h.config.Jira.BaseURL == "" {
		return false
	}

	// If using API method, check API credentials
	if h.config.UsesAPI() {
		return h.config.Jira.APIConfig.Username != "" &&
			h.config.Jira.APIConfig.APIToken != "" &&
			h.config.Jira.APIConfig.APIToken != "your-api-token"
	}

	// If using scraper method, consider it valid
	if h.config.UsesScraper() {
		return true
	}

	return false
}

// CollectionResponse represents the response from a collection request
type CollectionResponse struct {
	Success       bool      `json:"success"`
	Message       string    `json:"message"`
	TicketsCount  int       `json:"tickets_count"`
	ProjectsCount int       `json:"projects_count"`
	Duration      string    `json:"duration"`
	Timestamp     time.Time `json:"timestamp"`
	Error         string    `json:"error,omitempty"`
}

// CollectRequest represents the collection request from UI
type CollectRequest struct {
	Method    string `json:"method"`     // "api" or "scraper"
	BatchSize int    `json:"batch_size"` // Default 50
}

// CollectHandler triggers a collection/scrape operation
func (h *APIHandlers) CollectHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body for method selection
	var req CollectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If no body provided, use query params as fallback
		req.Method = r.URL.Query().Get("method")
		if req.Method == "" {
			req.Method = h.config.GetPrimaryMethod()
		}
		req.BatchSize = 50
		if bs := r.URL.Query().Get("batch_size"); bs != "" {
			fmt.Sscanf(bs, "%d", &req.BatchSize)
		}
	}

	if req.BatchSize <= 0 {
		req.BatchSize = 50
	}

	startTime := time.Now()
	h.logger.Info().
		Str("method", req.Method).
		Int("batch_size", req.BatchSize).
		Msg("Starting collection via web interface")

	// Trigger collection with specified method
	payloads, err := h.collectWithMethod(req.Method, req.BatchSize)

	duration := time.Since(startTime)

	if err != nil {
		h.logger.Error().Err(err).Msg("Collection failed via web interface")
		response := CollectionResponse{
			Success:   false,
			Message:   "Collection failed",
			Error:     err.Error(),
			Duration:  duration.String(),
			Timestamp: time.Now(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	h.logger.Info().
		Int("tickets", len(payloads)).
		Str("duration", duration.String()).
		Msg("Collection completed via web interface")

	response := CollectionResponse{
		Success:       true,
		Message:       "Collection completed successfully",
		TicketsCount:  len(payloads),
		ProjectsCount: len(h.config.Projects),
		Duration:      duration.String(),
		Timestamp:     time.Now(),
	}

	json.NewEncoder(w).Encode(response)
}

// collectWithMethod performs collection using the specified method
func (h *APIHandlers) collectWithMethod(method string, batchSize int) ([]plugin.Payload, error) {
	var allPayloads []plugin.Payload

	for _, project := range h.config.Projects {
		payloads, err := h.collectProjectWithMethod(project, batchSize, method)
		if err != nil {
			return nil, fmt.Errorf("failed to collect tickets for project %s: %w", project.Key, err)
		}
		allPayloads = append(allPayloads, payloads...)
	}

	sendLimit := h.config.Collector.SendLimit
	if sendLimit > 0 && len(allPayloads) > sendLimit {
		allPayloads = allPayloads[:sendLimit]
	}

	return allPayloads, nil
}

// collectProjectWithMethod collects a single project using specified method
func (h *APIHandlers) collectProjectWithMethod(project ProjectConfig, batchSize int, method string) ([]plugin.Payload, error) {
	// Use collector's method-specific collection
	return h.collector.CollectWithMethod(method, batchSize)
}

// ExtensionDataPayload represents data received from Chrome extension
type ExtensionDataPayload struct {
	Timestamp string                 `json:"timestamp"`
	URL       string                 `json:"url"`
	Title     string                 `json:"title"`
	Data      map[string]interface{} `json:"data"`
	Collector struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"collector"`
}

// ReceiverResponse represents the response to extension data
type ReceiverResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error,omitempty"`
}

// ReceiverHandler accepts data from Chrome extension
func (h *APIHandlers) ReceiverHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload ExtensionDataPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logger.Error().Err(err).Msg("Failed to decode extension data")
		response := ReceiverResponse{
			Success:   false,
			Message:   "Invalid payload format",
			Error:     err.Error(),
			Timestamp: time.Now(),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	h.logger.Info().
		Str("url", payload.URL).
		Str("collector", payload.Collector.Name).
		Str("version", payload.Collector.Version).
		Msg("Received data from Chrome extension")

	// Extract page type and process accordingly
	pageType := "unknown"
	if data, ok := payload.Data["pageType"].(string); ok {
		pageType = data
	}

	h.logger.Debug().
		Str("page_type", pageType).
		Str("title", payload.Title).
		Msg("Processing extension data")

	// Store the received data
	if err := h.storeExtensionData(payload); err != nil {
		h.logger.Error().Err(err).Msg("Failed to store extension data")
		response := ReceiverResponse{
			Success:   false,
			Message:   "Failed to store data",
			Error:     err.Error(),
			Timestamp: time.Now(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ReceiverResponse{
		Success:   true,
		Message:   fmt.Sprintf("Successfully received and stored %s page data", pageType),
		Timestamp: time.Now(),
	}

	h.logger.Info().
		Str("page_type", pageType).
		Msg("Successfully processed extension data")

	json.NewEncoder(w).Encode(response)
}

// storeExtensionData stores data received from the extension
func (h *APIHandlers) storeExtensionData(payload ExtensionDataPayload) error {
	// TODO: Implement proper storage logic for extension data
	// For now, just extract and store issue data if available

	pageType := "unknown"
	if pt, ok := payload.Data["pageType"].(string); ok {
		pageType = pt
	}

	// If this is an issue page with structured data, store it
	if pageType == "issue" {
		if issueData, ok := payload.Data["issue"].(map[string]interface{}); ok {
			if key, ok := issueData["key"].(string); ok {
				// Extract project key from issue key (e.g., "PROJ-123" -> "PROJ")
				projectKey := ""
				for i, c := range key {
					if c == '-' {
						projectKey = key[:i]
						break
					}
				}

				if projectKey != "" {
					// Load existing tickets for project
					tickets, err := h.storage.LoadTickets(projectKey)
					if err != nil {
						// If project doesn't exist, create new map
						tickets = make(map[string]*TicketData)
					}

					// Convert extension data to TicketData structure
					ticket := &TicketData{
						Key:     key,
						Updated: payload.Timestamp,
					}

					if summary, ok := issueData["summary"].(string); ok {
						ticket.Summary = summary
					}
					if description, ok := issueData["description"].(string); ok {
						ticket.Description = description
					}
					if issueType, ok := issueData["issueType"].(string); ok {
						ticket.IssueType = issueType
					}
					if status, ok := issueData["status"].(string); ok {
						ticket.Status = status
					}
					if priority, ok := issueData["priority"].(string); ok {
						ticket.Priority = priority
					}

					// Store the ticket
					tickets[key] = ticket

					// Save back to storage
					if err := h.storage.SaveTickets(projectKey, tickets); err != nil {
						return fmt.Errorf("failed to save ticket: %w", err)
					}

					h.logger.Info().
						Str("ticket", key).
						Str("project", projectKey).
						Msg("Stored issue data from extension")
				}
			}
		}
	}

	return nil
}
