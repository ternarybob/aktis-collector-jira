package services

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	. "aktis-collector-jira/internal/common"
	"aktis-collector-jira/internal/handlers"
	. "aktis-collector-jira/internal/interfaces"
	"github.com/ternarybob/arbor"
)

// webServer provides HTTP endpoints for monitoring and status
type webServer struct {
	config      *Config
	collector   Collector
	storage     Storage
	server      *http.Server
	logger      arbor.ILogger
	apiHandlers *handlers.APIHandlers
	uiHandlers  *handlers.UIHandlers
	running     bool
	startTime   time.Time
}

// NewWebServer creates a new web server instance
func NewWebServer(cfg *Config, collector Collector, storage Storage, logger arbor.ILogger) (WebService, error) {
	mux := http.NewServeMux()

	// Create API handlers
	apiHandlers := handlers.NewAPIHandlers(cfg, collector, storage, logger)

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
		collector:   collector,
		storage:     storage,
		logger:      logger,
		apiHandlers: apiHandlers,
		uiHandlers:  uiHandlers,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Collector.WebPort),
			Handler: mux,
		},
	}

	// Register API endpoints with logging middleware
	mux.HandleFunc("/health", ws.loggingMiddleware(apiHandlers.HealthHandler))
	mux.HandleFunc("/status", ws.loggingMiddleware(apiHandlers.StatusHandler))
	mux.HandleFunc("/database", ws.loggingMiddleware(apiHandlers.DatabaseHandler))
	mux.HandleFunc("/config", ws.loggingMiddleware(apiHandlers.ConfigHandler))

	// Register UI endpoints if available
	if uiHandlers != nil {
		mux.HandleFunc("/", ws.loggingMiddleware(uiHandlers.IndexHandler))
		mux.HandleFunc("/database/data", ws.loggingMiddleware(uiHandlers.BufferDataHandler))
	}

	return ws, nil
}

// Start starts the web server
func (ws *webServer) Start(ctx context.Context) error {
	ws.running = true
	ws.startTime = time.Now()

	go func() {
		ws.logger.Info().Int("port", ws.config.Collector.WebPort).Msg("Starting web server")
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

func (ws *webServer) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next(lrw, r)

		duration := time.Since(start)

		ws.logger.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote_addr", r.RemoteAddr).
			Int("status", lrw.statusCode).
			Dur("duration", duration).
			Msg("HTTP request")
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
