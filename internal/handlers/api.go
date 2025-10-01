// -----------------------------------------------------------------------
// Last Modified: Tuesday, 30th September 2025 2:54:00 pm
// Modified By: Bob McAllan
// -----------------------------------------------------------------------

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"aktis-collector-jira/internal/common"
	"aktis-collector-jira/internal/interfaces"
	"aktis-collector-jira/internal/models"

	"github.com/ternarybob/arbor"
)

// APIHandlers contains all API endpoint handlers
type APIHandlers struct {
	config    *common.Config
	storage   interfaces.Storage
	logger    arbor.ILogger
	startTime time.Time
	assessor  interfaces.PageAssessor
	wsHub     *WebSocketHub
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

// VersionResponse represents version information for both server and extension
type VersionResponse struct {
	Server struct {
		Version string `json:"version"`
		Build   string `json:"build"`
		Commit  string `json:"commit"`
	} `json:"server"`
	Extension struct {
		Version        string `json:"version"`
		LatestVersion  string `json:"latest_version"`
		UpdateRequired bool   `json:"update_required"`
	} `json:"extension"`
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
	Collector *common.CollectorConfig `json:"collector"`
	Storage   *common.StorageConfig   `json:"storage"`
	Logging   *common.LoggingConfig   `json:"logging"`
}

// DatabaseResponse represents database operation responses
type DatabaseResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Count   int    `json:"count,omitempty"`
}

// NewAPIHandlers creates a new API handlers instance
func NewAPIHandlers(config *common.Config, storage interfaces.Storage, logger arbor.ILogger, assessor interfaces.PageAssessor, wsHub *WebSocketHub) *APIHandlers {
	return &APIHandlers{
		config:    config,
		storage:   storage,
		logger:    logger,
		startTime: time.Now(),
		assessor:  assessor,
		wsHub:     wsHub,
	}
}

// HealthHandler returns system health status
func (h *APIHandlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   common.GetVersion(),
		Build:     common.GetBuild(),
		Uptime:    time.Since(h.startTime).Seconds(),
	}

	// Test database connection
	health.Services.Database = h.testDatabaseConnection()
	health.Services.Jira = true // No external Jira connection needed (extension-based)

	// If database is down, mark as degraded
	if !health.Services.Database {
		health.Status = "degraded"
	}

	if err := json.NewEncoder(w).Encode(health); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode health response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// VersionHandler returns version information and checks for extension updates
func (h *APIHandlers) VersionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get client's extension version from query parameter
	clientExtVersion := r.URL.Query().Get("extension_version")

	// Read latest extension version from .version file
	latestExtVersion := common.GetExtensionVersion()

	versionResp := VersionResponse{}

	// Server version info
	versionResp.Server.Version = common.GetVersion()
	versionResp.Server.Build = common.GetBuild()
	versionResp.Server.Commit = common.GetGitCommit()

	// Extension version info
	versionResp.Extension.LatestVersion = latestExtVersion
	if clientExtVersion != "" {
		versionResp.Extension.Version = clientExtVersion
		versionResp.Extension.UpdateRequired = clientExtVersion != latestExtVersion
	} else {
		versionResp.Extension.Version = "unknown"
		versionResp.Extension.UpdateRequired = false
	}

	if err := json.NewEncoder(w).Encode(versionResp); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode version response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// StatusHandler returns collector status and metrics
func (h *APIHandlers) StatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := StatusResponse{
		Projects: make([]ProjectStatus, 0),
		Stats: CollectorStats{
			DatabaseSize: "N/A",
		},
	}

	// Collector status
	status.Collector.Running = true // Assume running if we can respond
	status.Collector.Uptime = time.Since(h.startTime).Seconds()
	status.Collector.ErrorCount = 0

	// Load all tickets to calculate stats
	allTickets, err := h.storage.LoadAllTickets()
	if err != nil {
		h.logger.Warn().Err(err).Msg("Failed to load tickets for status")
	}

	status.Stats.TotalTickets = len(allTickets)

	// Find most recent update
	var lastUpdate time.Time
	for _, ticket := range allTickets {
		if ticket.Updated != "" {
			if ticketTime, err := time.Parse(time.RFC3339, ticket.Updated); err == nil {
				if ticketTime.After(lastUpdate) {
					lastUpdate = ticketTime
				}
			}
		}
	}

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

	// Create sanitized config
	config := ConfigResponse{
		Collector: &h.config.Collector,
		Storage:   &h.config.Storage,
		Logging:   &h.config.Logging,
	}

	if err := json.NewEncoder(w).Encode(config); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode config response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ProjectsHandler returns list of projects with their metadata
func (h *APIHandlers) ProjectsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Load projects from storage
	projects, err := h.storage.LoadProjects()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to load projects")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get ticket counts for each project
	projectsResponse := make([]map[string]interface{}, 0, len(projects))
	for _, project := range projects {
		tickets, err := h.storage.LoadTickets(project.Key)
		ticketCount := 0
		if err == nil {
			ticketCount = len(tickets)
		}

		projectsResponse = append(projectsResponse, map[string]interface{}{
			"id":           project.ID,
			"key":          project.Key,
			"name":         project.Name,
			"type":         project.Type,
			"url":          project.URL,
			"description":  project.Description,
			"updated":      project.Updated,
			"ticket_count": ticketCount,
		})
	}

	response := map[string]interface{}{
		"success":  true,
		"projects": projectsResponse,
		"count":    len(projectsResponse),
	}

	json.NewEncoder(w).Encode(response)
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
	// Load all tickets from database
	allTickets, err := h.storage.LoadAllTickets()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to load tickets")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	response := DatabaseResponse{
		Success: true,
		Message: fmt.Sprintf("Retrieved %d tickets", len(allTickets)),
		Count:   len(allTickets),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode database response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *APIHandlers) handleClearDatabase(w http.ResponseWriter, r *http.Request) {
	h.logger.Info().Msg("Clearing all stored data (projects and tickets) from database")

	// Clear all projects from storage
	if err := h.storage.ClearAllProjects(); err != nil {
		h.logger.Error().Err(err).Msg("Failed to clear projects from database")
		response := DatabaseResponse{
			Success: false,
			Message: "Failed to clear projects",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Clear all tickets from storage
	if err := h.storage.ClearAllTickets(); err != nil {
		h.logger.Error().Err(err).Msg("Failed to clear tickets from database")
		response := DatabaseResponse{
			Success: false,
			Message: "Failed to clear tickets",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	h.logger.Info().Msg("Successfully cleared all data from database")

	response := DatabaseResponse{
		Success: true,
		Message: "All data cleared from database",
		Count:   0,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode database response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *APIHandlers) testDatabaseConnection() bool {
	// Test by trying to load all tickets
	_, err := h.storage.LoadAllTickets()
	return err == nil
}

// storeIssuesArray stores multiple issues from an array
func (h *APIHandlers) storeIssuesArray(issuesArray []interface{}, timestamp string) error {
	storedCount := 0
	errorCount := 0

	// Group issues by project
	projectTickets := make(map[string]map[string]*models.TicketData)

	for _, issueInterface := range issuesArray {
		issueData, ok := issueInterface.(map[string]interface{})
		if !ok {
			continue
		}

		key, ok := issueData["key"].(string)
		if !ok || key == "" {
			continue
		}

		// Extract project key from issue key
		projectKey := ""
		for i, c := range key {
			if c == '-' {
				projectKey = key[:i]
				break
			}
		}

		if projectKey == "" {
			h.logger.Warn().Str("key", key).Msg("Could not extract project key from issue key")
			errorCount++
			continue
		}

		// Initialize project map if needed
		if projectTickets[projectKey] == nil {
			// Load existing tickets for this project
			existing, err := h.storage.LoadTickets(projectKey)
			if err != nil {
				// Create new map if project doesn't exist
				projectTickets[projectKey] = make(map[string]*models.TicketData)
			} else {
				projectTickets[projectKey] = existing
			}
		}

		// Convert to TicketData
		ticket := &models.TicketData{
			Key:       key,
			ProjectID: projectKey,
			Updated:   timestamp,
		}

		if projectID, ok := issueData["project_id"].(string); ok && projectID != "" {
			ticket.ProjectID = projectID
		}
		if url, ok := issueData["url"].(string); ok {
			ticket.URL = url
		}
		if summary, ok := issueData["summary"].(string); ok {
			ticket.Summary = summary
		}
		if description, ok := issueData["description"].(string); ok {
			ticket.Description = description
		}
		if issueType, ok := issueData["issue_type"].(string); ok {
			ticket.IssueType = issueType
		}
		if status, ok := issueData["status"].(string); ok {
			ticket.Status = status
		}
		if priority, ok := issueData["priority"].(string); ok {
			ticket.Priority = priority
		}
		if reporter, ok := issueData["reporter"].(string); ok {
			ticket.Reporter = reporter
		}
		if assignee, ok := issueData["assignee"].(string); ok {
			ticket.Assignee = assignee
		}

		projectTickets[projectKey][key] = ticket
		storedCount++
	}

	// Save all projects
	for projectKey, tickets := range projectTickets {
		if err := h.storage.SaveTickets(projectKey, tickets); err != nil {
			h.logger.Error().
				Err(err).
				Str("project", projectKey).
				Msg("Failed to save tickets for project")
			errorCount++
		} else {
			h.logger.Info().
				Str("project", projectKey).
				Int("count", len(tickets)).
				Msg("Stored tickets for project")
		}
	}

	h.logger.Info().
		Int("stored", storedCount).
		Int("errors", errorCount).
		Msg("Completed storing issues array")

	if errorCount > 0 && storedCount == 0 {
		return fmt.Errorf("failed to store any issues (%d errors)", errorCount)
	}

	return nil
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
	Success       bool             `json:"success"`
	Message       string           `json:"message"`
	Timestamp     time.Time        `json:"timestamp"`
	Error         string           `json:"error,omitempty"`
	Data          interface{}      `json:"data,omitempty"`
	PageType      string           `json:"page_type,omitempty"`
	TransactionID string           `json:"transaction_id,omitempty"`
	Stats         *CollectionStats `json:"stats,omitempty"`
}

// CollectionStats represents statistics from a collection operation
type CollectionStats struct {
	ProjectsAdded int `json:"projects_added"`
	ProjectsTotal int `json:"projects_total"`
	TicketsAdded  int `json:"tickets_added"`
	TicketsTotal  int `json:"tickets_total"`
}

// ProjectResponse represents a project in the response
type ProjectResponse struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// AssessPagePayload represents page assessment request
type AssessPagePayload struct {
	URL  string `json:"url"`
	HTML string `json:"html"`
}

// AssessHandler assesses page type without storing data
func (h *APIHandlers) AssessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload AssessPagePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logger.Error().Err(err).Msg("Failed to decode assessment payload")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid payload format",
		})
		return
	}

	h.logger.Info().
		Str("url", payload.URL).
		Msg("Assessing page type")

	// Use assessor service to analyze page
	assessment, err := h.assessor.AssessPage(payload.HTML, payload.URL)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to assess page")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to assess page",
		})
		return
	}

	h.logger.Info().
		Str("page_type", assessment.PageType).
		Str("confidence", assessment.Confidence).
		Str("collectable", fmt.Sprintf("%v", assessment.Collectable)).
		Msg("Page assessment completed")

	response := map[string]interface{}{
		"success":    true,
		"assessment": assessment,
		"timestamp":  time.Now(),
	}

	json.NewEncoder(w).Encode(response)
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

	// Generate transaction ID for tracking
	transactionID := fmt.Sprintf("txn-%d", time.Now().UnixNano())

	h.logger.Info().
		Str("transaction_id", transactionID).
		Str("url", payload.URL).
		Str("collector", payload.Collector.Name).
		Str("version", payload.Collector.Version).
		Msg("Received data from Chrome extension")

	// Broadcast collection started event to WebSocket clients
	if h.wsHub != nil {
		h.wsHub.SendCollectionUpdate("collection_started", map[string]interface{}{
			"transaction_id": transactionID,
			"url":            payload.URL,
			"title":          payload.Title,
			"timestamp":      payload.Timestamp,
		})
	}

	// Use page assessor to intelligently determine page type and processability
	htmlContent := ""
	if html, ok := payload.Data["html"].(string); ok {
		htmlContent = html
	}

	assessment, err := h.assessor.AssessPage(htmlContent, payload.URL)
	if err != nil {
		h.logger.Warn().Err(err).Msg("Failed to assess page, will attempt processing anyway")
		assessment = &models.PageAssessment{
			PageType:    "unknown",
			Confidence:  "low",
			Collectable: false,
		}
	}

	h.logger.Info().
		Str("page_type", assessment.PageType).
		Str("confidence", assessment.Confidence).
		Str("collectable", fmt.Sprintf("%v", assessment.Collectable)).
		Str("title", payload.Title).
		Msg("Page assessed, processing data")

	// If not collectable, return early with info
	if !assessment.Collectable {
		// Broadcast non-collectable status
		if h.wsHub != nil {
			h.wsHub.SendCollectionUpdate("collection_skipped", map[string]interface{}{
				"transaction_id": transactionID,
				"url":            payload.URL,
				"page_type":      assessment.PageType,
				"confidence":     assessment.Confidence,
				"reason":         "not collectable",
				"description":    assessment.Description,
			})
		}

		response := ReceiverResponse{
			Success:       true,
			Message:       fmt.Sprintf("Page received but not collectable: %s", assessment.Description),
			Timestamp:     time.Now(),
			PageType:      assessment.PageType,
			TransactionID: transactionID,
			Data: map[string]interface{}{
				"assessment": assessment,
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Store the received data and get response data with stats
	responseData, stats, err := h.storeExtensionDataWithStats(payload, assessment.PageType, transactionID)
	if err != nil {
		h.logger.Error().
			Str("transaction_id", transactionID).
			Err(err).
			Msg("Failed to store extension data")

		// Broadcast failure event
		if h.wsHub != nil {
			h.wsHub.SendCollectionUpdate("collection_failed", map[string]interface{}{
				"transaction_id": transactionID,
				"url":            payload.URL,
				"page_type":      assessment.PageType,
				"error":          err.Error(),
			})
		}

		response := ReceiverResponse{
			Success:       false,
			Message:       "Failed to store data",
			Error:         err.Error(),
			Timestamp:     time.Now(),
			PageType:      assessment.PageType,
			TransactionID: transactionID,
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Build success message with stats
	successMsg := fmt.Sprintf("Successfully processed %s page", assessment.PageType)
	if stats != nil {
		if stats.ProjectsAdded > 0 {
			successMsg += fmt.Sprintf(" - Added %d project(s)", stats.ProjectsAdded)
		}
		if stats.TicketsAdded > 0 {
			successMsg += fmt.Sprintf(" - Added %d ticket(s)", stats.TicketsAdded)
		}
	}

	response := ReceiverResponse{
		Success:       true,
		Message:       successMsg,
		Timestamp:     time.Now(),
		PageType:      assessment.PageType,
		TransactionID: transactionID,
		Data:          responseData,
		Stats:         stats,
	}

	h.logger.Info().
		Str("transaction_id", transactionID).
		Str("page_type", assessment.PageType).
		Int("projects_added", stats.ProjectsAdded).
		Int("tickets_added", stats.TicketsAdded).
		Msg("Successfully processed extension data")

	// Broadcast success event with collection details
	if h.wsHub != nil {
		h.wsHub.SendCollectionUpdate("collection_success", map[string]interface{}{
			"transaction_id": transactionID,
			"url":            payload.URL,
			"page_type":      assessment.PageType,
			"stats":          stats,
			"data":           responseData,
		})
	}

	json.NewEncoder(w).Encode(response)
}

// makeAbsoluteURL converts a relative URL to an absolute URL using the base page URL
func (h *APIHandlers) makeAbsoluteURL(relativeURL, baseURL string) string {
	// If already absolute, return as-is
	if strings.HasPrefix(relativeURL, "http://") || strings.HasPrefix(relativeURL, "https://") {
		return relativeURL
	}

	// Parse base URL to get scheme and host
	// Extract scheme and host from baseURL (e.g., https://company.atlassian.net)
	if strings.HasPrefix(baseURL, "http://") || strings.HasPrefix(baseURL, "https://") {
		// Find the end of scheme://host
		schemeEnd := strings.Index(baseURL, "://")
		if schemeEnd == -1 {
			return relativeURL // Fallback
		}
		hostStart := schemeEnd + 3
		pathStart := strings.Index(baseURL[hostStart:], "/")
		if pathStart == -1 {
			// No path, entire remaining is host
			return baseURL + relativeURL
		}
		baseHost := baseURL[:hostStart+pathStart]
		return baseHost + relativeURL
	}

	return relativeURL
}

// storeExtensionData stores data received from the extension and returns response data
func (h *APIHandlers) storeExtensionData(payload ExtensionDataPayload, assessedPageType string) (interface{}, error) {
	pageType := assessedPageType
	if pageType == "" {
		pageType = "unknown"
	}

	h.logger.Debug().
		Str("page_type", pageType).
		Str("url", payload.URL).
		Msg("Processing extension data with server-side HTML parsing")

	// Get HTML content
	htmlContent, ok := payload.Data["html"].(string)
	if !ok || htmlContent == "" {
		h.logger.Warn().Msg("No HTML content in payload")
		return nil, nil
	}

	// Parse HTML on server side
	parser := NewJiraParser()
	results, err := parser.ParseHTML(htmlContent, pageType, payload.URL)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to parse HTML")
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	if len(results) == 0 {
		h.logger.Warn().
			Str("page_type", pageType).
			Str("url", payload.URL).
			Int("html_size", len(htmlContent)).
			Msg("No data found in HTML - parser returned empty results")

		// Log HTML snippet for debugging (first 1000 chars)
		snippet := htmlContent
		if len(snippet) > 1000 {
			snippet = snippet[:1000]
		}
		h.logger.Debug().
			Str("html_snippet", snippet).
			Msg("HTML content preview for debugging")

		return nil, nil
	}

	// Handle based on page type
	if pageType == "projectsList" {
		// Convert to ProjectData and store
		projects := make([]*models.ProjectData, 0, len(results))
		for _, result := range results {
			projectMap := result
			project := &models.ProjectData{
				Updated: payload.Timestamp,
			}
			if id, ok := projectMap["id"].(string); ok {
				project.ID = id
			}
			if key, ok := projectMap["key"].(string); ok {
				project.Key = key
				// Use key as ID if no numeric ID was found
				if project.ID == "" {
					project.ID = key
				}
			}
			if name, ok := projectMap["name"].(string); ok {
				project.Name = name
			}
			if ptype, ok := projectMap["type"].(string); ok {
				project.Type = ptype
			}
			if url, ok := projectMap["url"].(string); ok {
				// Convert relative URL to absolute URL
				project.URL = h.makeAbsoluteURL(url, payload.URL)
			}
			if desc, ok := projectMap["description"].(string); ok {
				project.Description = desc
			}
			if project.Key != "" {
				projects = append(projects, project)
			}
		}
		if len(projects) > 0 {
			h.logger.Info().Int("project_count", len(projects)).Msg("Storing projects")
			if err := h.storage.SaveProjects(projects); err != nil {
				return nil, fmt.Errorf("failed to save projects: %w", err)
			}
		}

		// Convert projects to response format and return
		projectResponses := make([]ProjectResponse, len(projects))
		for i, p := range projects {
			projectResponses[i] = ProjectResponse{
				Key:         p.Key,
				Name:        p.Name,
				Type:        p.Type,
				URL:         p.URL,
				Description: p.Description,
			}
		}
		return projectResponses, nil
	}

	// Check if extension already extracted tickets (from DOM)
	if ticketsData, ok := payload.Data["tickets"].([]interface{}); ok && len(ticketsData) > 0 {
		h.logger.Info().Int("ticket_count", len(ticketsData)).Msg("Using pre-extracted tickets from extension")

		err = h.storeIssuesArray(ticketsData, payload.Timestamp)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"tickets_collected": len(ticketsData),
		}, nil
	}

	// For issue pages, store as tickets
	h.logger.Info().Int("issue_count", len(results)).Msg("Extracted issues from HTML")

	// Convert parsed issues to interface array and store
	issuesArray := make([]interface{}, len(results))
	for i, issue := range results {
		issuesArray[i] = issue
	}

	err = h.storeIssuesArray(issuesArray, payload.Timestamp)
	if err != nil {
		return nil, err
	}

	// Return ticket count for issue pages
	return map[string]interface{}{
		"tickets_collected": len(results),
	}, nil
}

// storeExtensionDataWithStats wraps storeExtensionData and calculates collection statistics
func (h *APIHandlers) storeExtensionDataWithStats(payload ExtensionDataPayload, assessedPageType string, transactionID string) (interface{}, *CollectionStats, error) {
	// Get counts before processing
	projectsBefore, _ := h.storage.LoadProjects()
	ticketsBefore, _ := h.storage.LoadAllTickets()

	// Store the data
	responseData, err := h.storeExtensionData(payload, assessedPageType)
	if err != nil {
		return nil, nil, err
	}

	// Get counts after processing
	projectsAfter, _ := h.storage.LoadProjects()
	ticketsAfter, _ := h.storage.LoadAllTickets()

	// Calculate statistics
	stats := &CollectionStats{
		ProjectsAdded: len(projectsAfter) - len(projectsBefore),
		ProjectsTotal: len(projectsAfter),
		TicketsAdded:  len(ticketsAfter) - len(ticketsBefore),
		TicketsTotal:  len(ticketsAfter),
	}

	h.logger.Debug().
		Str("transaction_id", transactionID).
		Int("projects_added", stats.ProjectsAdded).
		Int("projects_total", stats.ProjectsTotal).
		Int("tickets_added", stats.TicketsAdded).
		Int("tickets_total", stats.TicketsTotal).
		Msg("Collection statistics calculated")

	return responseData, stats, nil
}
