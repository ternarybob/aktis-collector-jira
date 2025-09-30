package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"

	. "aktis-collector-jira/internal/common"
	. "aktis-collector-jira/internal/interfaces"
	"github.com/ternarybob/arbor"
)

// UIHandlers contains all UI endpoint handlers
type UIHandlers struct {
	config    *Config
	storage   Storage
	logger    arbor.ILogger
	templates *template.Template
}

// TemplateData represents data passed to templates
type TemplateData struct {
	Title       string
	ServiceName string
	Version     string
	Build       string
	Environment string
}

// NewUIHandlers creates a new UI handlers instance
func NewUIHandlers(config *Config, storage Storage, logger arbor.ILogger, pagesDir string) (*UIHandlers, error) {
	// Load templates
	templatesPath := filepath.Join(pagesDir, "*.html")
	templates, err := template.ParseGlob(templatesPath)
	if err != nil {
		return nil, err
	}

	return &UIHandlers{
		config:    config,
		storage:   storage,
		logger:    logger,
		templates: templates,
	}, nil
}

// IndexHandler serves the main web interface
func (h *UIHandlers) IndexHandler(w http.ResponseWriter, r *http.Request) {
	data := TemplateData{
		Title:       "Jira Collector",
		ServiceName: h.config.Collector.Name,
		Version:     GetVersion(),
		Build:       GetBuild(),
		Environment: h.config.Collector.Environment,
	}

	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		h.logger.Error().Err(err).Msg("Failed to execute template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// BufferDataHandler provides HTMX endpoint for database data
func (h *UIHandlers) BufferDataHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetBufferData(w, r)
	case http.MethodDelete:
		h.handleClearBufferData(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *UIHandlers) handleGetBufferData(w http.ResponseWriter, r *http.Request) {
	// Get all stored tickets across all projects
	allTickets := make(map[string]interface{})

	// First, try to load tickets from configured projects
	for _, project := range h.config.Projects {
		tickets, err := h.storage.LoadTickets(project.Key)
		if err != nil {
			h.logger.Warn().Err(err).Str("project", project.Key).Msg("Failed to load tickets")
			continue
		}

		if len(tickets) > 0 {
			allTickets[project.Key] = tickets
		}
	}

	// Also try to discover tickets from any project (for extension-collected data)
	// Get all tickets without filtering by project
	allStoredTickets, err := h.storage.LoadAllTickets()
	if err != nil {
		h.logger.Warn().Err(err).Msg("Failed to load all tickets")
	} else {
		// Organize by project key
		projectTickets := make(map[string]map[string]*TicketData)
		for key, ticket := range allStoredTickets {
			// Extract project key from ticket key (e.g., "PROJ-123" -> "PROJ")
			projectKey := ""
			for i, c := range key {
				if c == '-' {
					projectKey = key[:i]
					break
				}
			}

			if projectKey != "" {
				if projectTickets[projectKey] == nil {
					projectTickets[projectKey] = make(map[string]*TicketData)
				}
				projectTickets[projectKey][key] = ticket
			}
		}

		// Merge with configured projects
		for pk, tickets := range projectTickets {
			if len(tickets) > 0 {
				allTickets[pk] = tickets
			}
		}
	}

	if len(allTickets) == 0 {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<div class="metric">
			<div class="metric-header">No data available</div>
			<p>No Jira tickets have been collected yet. Use the Chrome extension to collect ticket data.</p>
		</div>`))
		return
	}

	// Return JSON data for display
	w.Header().Set("Content-Type", "application/json")
	jsonOutput, err := json.MarshalIndent(allTickets, "", "  ")
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to marshal tickets data")
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<div class="error">Error formatting database data: ` + err.Error() + `</div>`))
		return
	}

	w.Write(jsonOutput)
}

func (h *UIHandlers) handleClearBufferData(w http.ResponseWriter, r *http.Request) {
	// Clear stored tickets for all projects
	var errors []string

	for _, project := range h.config.Projects {
		err := h.storage.SaveTickets(project.Key, make(map[string]*TicketData))
		if err != nil {
			h.logger.Error().Err(err).Str("project", project.Key).Msg("Failed to clear tickets")
			errors = append(errors, "Failed to clear "+project.Key+": "+err.Error())
		}
	}

	if len(errors) > 0 {
		w.Header().Set("Content-Type", "text/html")
		errorMsg := "Errors occurred while clearing data:<ul>"
		for _, err := range errors {
			errorMsg += "<li>" + err + "</li>"
		}
		errorMsg += "</ul>"
		w.Write([]byte(`<div class="error">` + errorMsg + `</div>`))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<div class="metric">
		<div class="metric-header">Database Cleared</div>
		<p>All stored Jira tickets have been successfully removed from the database.</p>
	</div>`))
}
