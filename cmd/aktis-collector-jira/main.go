// -----------------------------------------------------------------------
// Last Modified: Friday, 26th September 2025 4:58:22 pm
// Modified By: Bob McAllan
// -----------------------------------------------------------------------

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"aktis-collector-jira/internal/common"
	"aktis-collector-jira/internal/interfaces"
	"aktis-collector-jira/internal/services"
	plugin "github.com/ternarybob/aktis-plugin-sdk"
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
		update         = flag.Bool("update", false, "Run in update mode (fetch only latest changes)")
		batchSize      = flag.Int("batch-size", 50, "Number of tickets to process in each batch")
		validateConfig = flag.Bool("validate", false, "Validate configuration file and exit")
		server         = flag.Bool("server", false, "Run in server mode with web interface")
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
		collectionMode := "Collection"
		if *update {
			collectionMode = "Update"
		}
		logFilePath := common.GetLogFilePath()
		common.PrintBanner(pluginName, environment, collectionMode, logFilePath)
	}

	startTime := time.Now()

	// Initialize services
	logger.Info().Msg("Initializing services...")

	// Create Jira client
	jiraClient := services.NewJiraClient(&cfg.Jira)

	// Create storage
	storage, err := services.NewStorage(&cfg.Storage)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to initialize storage")
		handleError(err, *quiet, environment, startTime)
		return
	}
	defer storage.Close()

	// Create collector
	collector := services.NewCollector(cfg, jiraClient, storage)
	defer collector.Close()

	logger.Info().Msg("Services initialized successfully")

	if *server {
		// Server mode - start web server and run continuously
		runServerMode(cfg, collector, storage, logger, environment)
	} else {
		// Single run mode - collect data and exit
		runCollectionMode(collector, *update, *batchSize, *quiet, environment, startTime, logger)
	}

	logger.Info().Msg("Aktis Collector Jira Service shutdown complete")
}

func runServerMode(cfg *common.Config, collector interfaces.Collector, storage interfaces.Storage, logger arbor.ILogger, environment string) {
	logger.Info().Msg("Starting in server mode")

	// Create web server
	webServer, err := services.NewWebServer(cfg, collector, storage, logger)
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
		Int("port", cfg.Collector.WebPort).
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

func runCollectionMode(collector interfaces.Collector, update bool, batchSize int, quiet bool, environment string, startTime time.Time, logger arbor.ILogger) {
	// Collect data based on mode
	logger.Info().
		Str("mode", fmt.Sprintf("update=%t", update)).
		Str("batch_size", fmt.Sprintf("%d", batchSize)).
		Msg("Starting data collection")

	var payloads []plugin.Payload
	payloads, err := collector.CollectAllTickets(batchSize)

	if err != nil {
		logger.Error().Err(err).Msg("Data collection failed")
		handleError(err, quiet, environment, startTime)
		return
	}

	logger.Info().
		Str("payload_count", fmt.Sprintf("%d", len(payloads))).
		Str("duration", time.Since(startTime).String()).
		Msg("Data collection completed successfully")

	// Build successful output
	output := plugin.CollectorOutput{
		Success:   true,
		Timestamp: time.Now(),
		Payloads:  payloads,
		Collector: plugin.CollectorInfo{
			Name:        pluginName,
			Type:        plugin.CollectorTypeData,
			Version:     pluginVersion,
			Environment: environment,
		},
		Stats: plugin.CollectorStats{
			Duration:     time.Since(startTime).String(),
			PayloadCount: len(payloads),
		},
	}

	if quiet {
		// JSON output for aktis-collector
		json.NewEncoder(os.Stdout).Encode(output)
	} else {
		// Human-readable CLI output
		displayResults(output, update)
	}
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

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
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
	fmt.Println("  -update             Run in update mode (fetch only latest changes)")
	fmt.Println("  -batch-size int     Number of tickets to process in each batch (default 50)")
	fmt.Println("  -validate           Validate configuration file and exit")
	fmt.Println("  -server             Run in server mode with web interface")
	fmt.Println("\nExamples:")
	fmt.Printf("  %s                                  # Run in development mode\n", os.Args[0])
	fmt.Printf("  %s -mode prod                       # Run in production mode\n", os.Args[0])
	fmt.Printf("  %s -update                          # Update existing tickets\n", os.Args[0])
	fmt.Printf("  %s -config /path/to/config.json     # Use custom config file\n", os.Args[0])
	fmt.Printf("  %s -server                          # Run with web interface on port 8080\n", os.Args[0])
	fmt.Printf("  %s -quiet                           # JSON output for aktis-collector\n", os.Args[0])
}

func handleError(err error, quiet bool, environment string, startTime time.Time) {
	if quiet {
		// JSON error output for aktis-collector
		output := plugin.CollectorOutput{
			Success:   false,
			Timestamp: time.Now(),
			Error:     err.Error(),
			Collector: plugin.CollectorInfo{
				Name:        pluginName,
				Type:        plugin.CollectorTypeData,
				Version:     pluginVersion,
				Environment: environment,
			},
			Stats: plugin.CollectorStats{
				Duration: time.Since(startTime).String(),
			},
		}
		json.NewEncoder(os.Stdout).Encode(output)
	} else {
		fmt.Printf("\n❌ Error: %v\n", err)
		os.Exit(1)
	}
}

func displayResults(output plugin.CollectorOutput, updateMode bool) {
	fmt.Printf("\n📊 Collection Summary\n")
	fmt.Printf("===================\n")

	mode := "Collected"
	if updateMode {
		mode = "Updated"
	}

	fmt.Printf("%s: %d tickets\n", mode, output.Stats.PayloadCount)
	fmt.Printf("Duration: %s\n", output.Stats.Duration)
	fmt.Printf("Timestamp: %s\n", output.Timestamp.Format("2006-01-02 15:04:05"))

	if output.Stats.PayloadCount > 0 {
		fmt.Printf("\n🎟️  Ticket Types:\n")
		typeCount := make(map[string]int)
		for _, payload := range output.Payloads {
			typeCount[payload.Type]++
		}
		for ticketType, count := range typeCount {
			fmt.Printf("  • %s: %d\n", ticketType, count)
		}
	}

	fmt.Printf("\n✅ Collection completed successfully!\n")
}
