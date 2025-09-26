package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	plugin "github.com/ternarybob/aktis-plugin-sdk"
	"github.com/ternarybob/jira-collector/internal/collector"
)

const (
	pluginName    = "jira-collector"
	pluginVersion = "1.0.0"
)

func main() {
	// Standard command line flags
	mode := flag.String("mode", "dev", "Environment mode: 'dev', 'development', 'prod', or 'production'")
	configFile := flag.String("config", "", "Configuration file path")
	quiet := flag.Bool("quiet", false, "Suppress banner output")
	version := flag.Bool("version", false, "Show version information")
	help := flag.Bool("help", false, "Show help message")
	update := flag.Bool("update", false, "Run in update mode (fetch only latest changes)")
	batchSize := flag.Int("batch-size", 50, "Number of tickets to process in each batch")
	flag.Parse()

	// Handle version and help
	if *version {
		fmt.Printf("%s v%s\n", pluginName, pluginVersion)
		return
	}
	if *help {
		showHelp()
		return
	}

	// Parse environment from mode
	environment := parseMode(*mode)

	// Load configuration
	cfg, err := collector.LoadConfig(*configFile)
	if err != nil {
		handleError(err, *quiet, environment, time.Now())
		return
	}

	// Show banner unless quiet
	if !*quiet {
		showBanner(environment, *update)
	}

	startTime := time.Now()

	// Initialize Jira collector
	jiraCollector, err := collector.NewJiraCollector(cfg)
	if err != nil {
		handleError(err, *quiet, environment, startTime)
		return
	}

	// Collect data based on mode
	var payloads []plugin.Payload
	if *update {
		payloads, err = jiraCollector.UpdateTickets(*batchSize)
	} else {
		payloads, err = jiraCollector.CollectAllTickets(*batchSize)
	}

	if err != nil {
		handleError(err, *quiet, environment, startTime)
		return
	}

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

	if *quiet {
		// JSON output for aktis-collector
		json.NewEncoder(os.Stdout).Encode(output)
	} else {
		// Human-readable CLI output
		displayResults(output, *update)
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

func showBanner(environment string, updateMode bool) {
	mode := "Collection"
	if updateMode {
		mode = "Update"
	}
	fmt.Printf("🎯 %s v%s (%s Mode)\n", pluginName, pluginVersion, mode)
	fmt.Printf("📍 Environment: %s\n", environment)
	fmt.Printf("⏰ Started: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
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
	fmt.Println("\nExamples:")
	fmt.Printf("  %s                                  # Run in development mode\n", os.Args[0])
	fmt.Printf("  %s -mode prod                       # Run in production mode\n", os.Args[0])
	fmt.Printf("  %s -update                          # Update existing tickets\n", os.Args[0])
	fmt.Printf("  %s -config /path/to/config.json     # Use custom config file\n", os.Args[0])
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
