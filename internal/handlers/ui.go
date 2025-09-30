package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"

	"aktis-collector-jira/internal/common"
	"aktis-collector-jira/internal/interfaces"

	"github.com/ternarybob/arbor"
)

// UIHandlers contains all UI endpoint handlers
type UIHandlers struct {
	config    *common.Config
	storage   interfaces.Storage
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
func NewUIHandlers(config *common.Config, storage interfaces.Storage, logger arbor.ILogger, pagesDir string) (*UIHandlers, error) {
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
		Version:     common.GetVersion(),
		Build:       common.GetBuild(),
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
	// Get all stored tickets from extension-collected data
	allStoredTickets, err := h.storage.LoadAllTickets()
	if err != nil {
		h.logger.Warn().Err(err).Msg("Failed to load all tickets")
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<div class="metric">
			<div class="metric-header">Error loading data</div>
			<p>Failed to load tickets from database</p>
		</div>`))
		return
	}

	if len(allStoredTickets) == 0 {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<div class="metric">
			<div class="metric-header">No data available</div>
			<p>No Jira tickets have been collected yet. Use the Chrome extension to collect ticket data.</p>
		</div>`))
		return
	}

	// Return flat JSON array for display
	w.Header().Set("Content-Type", "application/json")
	jsonOutput, err := json.MarshalIndent(allStoredTickets, "", "  ")
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to marshal tickets data")
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<div class="error">Error formatting database data: ` + err.Error() + `</div>`))
		return
	}

	w.Write(jsonOutput)
}

func (h *UIHandlers) handleClearBufferData(w http.ResponseWriter, r *http.Request) {
	h.logger.Info().Msg("Clearing all stored tickets from buffer")

	// Clear all tickets from storage
	if err := h.storage.ClearAllTickets(); err != nil {
		h.logger.Error().Err(err).Msg("Failed to clear tickets from buffer")
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`<div class="error">Failed to clear buffer: ` + err.Error() + `</div>`))
		return
	}

	h.logger.Info().Msg("Successfully cleared all tickets from buffer")

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<div class="metric">
		<div class="metric-header">Buffer Cleared</div>
		<p>All stored Jira tickets have been successfully removed from the database.</p>
	</div>`))
}
