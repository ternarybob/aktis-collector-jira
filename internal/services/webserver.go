package services

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"aktis-collector-jira/internal/common"
	"aktis-collector-jira/internal/handlers"
	"aktis-collector-jira/internal/interfaces"
	"aktis-collector-jira/internal/middleware"

	"github.com/ternarybob/arbor"
)

// webServer provides HTTP endpoints for monitoring and status
type webServer struct {
	config      *common.Config
	storage     interfaces.Storage
	server      *http.Server
	logger      arbor.ILogger
	apiHandlers *handlers.APIHandlers
	uiHandlers  *handlers.UIHandlers
	wsHub       *handlers.WebSocketHub
	running     bool
	startTime   time.Time
}

// NewWebServer creates a new web server instance
func NewWebServer(cfg *common.Config, storage interfaces.Storage, logger arbor.ILogger) (interfaces.WebService, error) {
	mux := http.NewServeMux()

	// Create page assessor service
	assessor := NewPageAssessor(logger)

	// Create WebSocket hub first (needed by API handlers)
	wsHub := handlers.NewWebSocketHub(logger)

	// Create API handlers with assessor and WebSocket hub
	apiHandlers := handlers.NewAPIHandlers(cfg, storage, logger, assessor, wsHub)

	// Find pages directory - check both relative to working dir and binary location
	pagesDir := "pages"
	if _, err := os.Stat(pagesDir); os.IsNotExist(err) {
		// Try relative to binary location
		pagesDir = filepath.Join(".", "pages")
		if _, err := os.Stat(pagesDir); os.IsNotExist(err) {
			logger.Warn().Msg("Pages directory not found, UI will not be available")
		}
	}

	// Create UI handlers
	uiHandlers, err := handlers.NewUIHandlers(cfg, storage, logger, pagesDir)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to initialize UI handlers, only API endpoints will be available")
	}

	ws := &webServer{
		config:      cfg,
		storage:     storage,
		logger:      logger,
		apiHandlers: apiHandlers,
		uiHandlers:  uiHandlers,
		wsHub:       wsHub,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Collector.Port),
			Handler: mux,
		},
	}

	// Create middleware chain
	logMiddleware := middleware.Logging(logger)
	corsMiddleware := middleware.CORS

	// Register API endpoints with middleware
	mux.HandleFunc("/health", logMiddleware(corsMiddleware(apiHandlers.HealthHandler)))
	mux.HandleFunc("/version", logMiddleware(corsMiddleware(apiHandlers.VersionHandler)))
	mux.HandleFunc("/status", logMiddleware(corsMiddleware(apiHandlers.StatusHandler)))
	mux.HandleFunc("/projects", logMiddleware(corsMiddleware(apiHandlers.ProjectsHandler)))
	mux.HandleFunc("/database", logMiddleware(corsMiddleware(apiHandlers.DatabaseHandler)))
	mux.HandleFunc("/config", logMiddleware(corsMiddleware(apiHandlers.ConfigHandler)))
	mux.HandleFunc("/assess", logMiddleware(corsMiddleware(apiHandlers.AssessHandler)))
	mux.HandleFunc("/receiver", logMiddleware(corsMiddleware(apiHandlers.ReceiverHandler)))

	// Register WebSocket endpoint
	mux.HandleFunc("/ws", corsMiddleware(wsHub.WebSocketHandler))

	// Register UI endpoints if available
	if uiHandlers != nil {
		mux.HandleFunc("/", logMiddleware(uiHandlers.IndexHandler))
		mux.HandleFunc("/database/data", logMiddleware(uiHandlers.BufferDataHandler))
	}

	return ws, nil
}

// Start starts the web server
func (ws *webServer) Start(ctx context.Context) error {
	ws.running = true
	ws.startTime = time.Now()

	go func() {
		ws.logger.Info().Int("port", ws.config.Collector.Port).Msg("Starting web server")
		if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ws.logger.Error().Err(err).Msg("Web server error")
		}
	}()
	return nil
}

// Stop stops the web server
func (ws *webServer) Stop() error {
	ws.running = false

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ws.logger.Info().Msg("Shutting down web server")
	return ws.server.Shutdown(ctx)
}

// IsRunning returns true if the web server is running
func (ws *webServer) IsRunning() bool {
	return ws.running
}
