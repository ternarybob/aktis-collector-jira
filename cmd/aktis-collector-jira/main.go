// -----------------------------------------------------------------------
// Last Modified: Friday, 26th September 2025 4:58:22 pm
// Modified By: Bob McAllan
// -----------------------------------------------------------------------

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"aktis-collector-jira/internal/common"
	"aktis-collector-jira/internal/interfaces"
	"aktis-collector-jira/internal/services"
	"github.com/ternarybob/arbor"
)

const (
	pluginName    = "aktis-collector-jira"
	pluginVersion = "1.0.0"
)

func main() {
	// Parse command line flags
	var (
		configPath     = flag.String("config", "", "Path to configuration file")
		mode           = flag.String("mode", "dev", "Environment mode: 'dev', 'development', 'prod', or 'production'")
		quiet          = flag.Bool("quiet", false, "Suppress banner output")
		version        = flag.Bool("version", false, "Show version information")
		help           = flag.Bool("help", false, "Show help message")
		validateConfig = flag.Bool("validate", false, "Validate configuration file and exit")
	)
	flag.Parse()

	// Handle version flag
	if *version {
		fmt.Printf("%s v%s (build: %s)\n", pluginName, common.GetVersion(), common.GetBuild())
		os.Exit(0)
	}

	// Handle help flag
	if *help {
		showHelp()
		os.Exit(0)
	}

	// Parse environment from mode
	environment := parseMode(*mode)

	// Load configuration with priority: defaults -> TOML
	cfg, err := common.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Update environment from command line
	cfg.Collector.Environment = environment

	// Handle validate flag
	if *validateConfig {
		fmt.Println("Configuration is valid")
		os.Exit(0)
	}

	// Initialize logger
	loggingConfig := &common.LoggingConfig{
		Level:  "info",
		Format: "text",
		Output: "both",
	}
	if err := common.InitLogger(loggingConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Now get the configured logger
	logger := common.GetLogger()

	// Log startup information first to ensure log file is created
	logger.Info().
		Str("version", common.GetVersion()).
		Str("build", common.GetBuild()).
		Str("environment", environment).
		Msg("Starting Aktis Collector Jira Service")

	logger.Info().
		Str("config_path", *configPath).
		Msg("Configuration loaded")

	// Display startup banner after initial log messages (to ensure log file exists)
	if !*quiet {
		logFilePath := common.GetLogFilePath()
		common.PrintBanner(pluginName, environment, "Server", logFilePath)
	}

	// Initialize services
	logger.Info().Msg("Initializing services...")

	// Create storage
	storage, err := services.NewStorage(&cfg.Storage)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to initialize storage")
		os.Exit(1)
	}
	defer storage.Close()

	logger.Info().Msg("Services initialized successfully")

	// Server mode - start web server and run continuously
	runServerMode(cfg, storage, logger, environment)

	logger.Info().Msg("Aktis Collector Jira Service shutdown complete")
}

func runServerMode(cfg *common.Config, storage interfaces.Storage, logger arbor.ILogger, environment string) {
	logger.Info().Msg("Starting in server mode")

	// Create web server
	webServer, err := services.NewWebServer(cfg, storage, logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create web server")
		return
	}

	// Start web server
	ctx := context.Background()
	if err := webServer.Start(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to start web server")
		return
	}

	logger.Info().
		Int("port", cfg.Collector.Port).
		Msg("Web server started successfully")

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info().Msg("Server running - press Ctrl+C to stop")

	// Wait for shutdown signal
	<-sigChan
	logger.Info().Msg("Shutdown signal received")

	// Stop web server
	if err := webServer.Stop(); err != nil {
		logger.Error().Err(err).Msg("Error stopping web server")
	}

	logger.Info().Msg("Server mode shutdown complete")
}

func parseMode(mode string) string {
	mode = strings.ToLower(mode)
	switch mode {
	case "prod", "production":
		return "production"
	case "dev", "development":
		return "development"
	default:
		return "development"
	}
}

func showHelp() {
	fmt.Printf("%s v%s - Jira Ticket Collector\n\n", pluginName, pluginVersion)
	fmt.Println("Usage:")
	fmt.Printf("  %s [flags]\n\n", os.Args[0])
	fmt.Println("Flags:")
	fmt.Println("  -mode string        Environment mode: 'dev', 'development', 'prod', or 'production' (default \"dev\")")
	fmt.Println("  -config string      Configuration file path")
	fmt.Println("  -quiet              Suppress banner output")
	fmt.Println("  -version            Show version information")
	fmt.Println("  -help               Show help message")
	fmt.Println("  -validate           Validate configuration file and exit")
	fmt.Println("\nExamples:")
	fmt.Printf("  %s                                  # Run in server mode\n", os.Args[0])
	fmt.Printf("  %s -mode prod                       # Run server in production mode\n", os.Args[0])
	fmt.Printf("  %s -config /path/to/config.toml     # Use custom config file\n", os.Args[0])
	fmt.Println("\nNote: Data collection is performed via the Chrome extension, not the collector binary.")
}
