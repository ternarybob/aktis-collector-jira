package common

import (
	"fmt"
	"strings"

	"github.com/ternarybob/banner"
)

// PrintBanner displays the application startup banner
func PrintBanner(serviceName, environment, mode, logFile string) {
	version := GetVersion()
	build := GetBuild()

	// Create banner with custom styling
	b := banner.New().
		SetStyle(banner.StyleDouble).
		SetBorderColor(banner.ColorPurple).
		SetTextColor(banner.ColorWhite).
		SetBold(true).
		SetWidth(80)

	fmt.Printf("\n")

	// Print banner header
	b.PrintTopLine()
	b.PrintCenteredText("AKTIS COLLECTOR - JIRA")
	b.PrintCenteredText("Jira Data Collection Service")
	b.PrintSeparatorLine()

	// Print version and runtime information
	b.PrintKeyValue("Version", version, 15)
	b.PrintKeyValue("Build", build, 15)
	b.PrintKeyValue("Environment", environment, 15)
	b.PrintKeyValue("Mode", mode, 15)
	b.PrintKeyValue("Plugin SDK", "Aktis Plugin SDK v0.1.2", 15)
	b.PrintBottomLine()

	fmt.Printf("\n")

	// Print configuration details
	fmt.Printf("📋 Configuration:\n")
	fmt.Printf("   • Config File: config.json\n")

	// Show log file if provided
	if logFile != "" {
		pattern := strings.Replace(logFile, ".log", ".{YYYY-MM-DDTHH-MM-SS}.log", 1)
		fmt.Printf("   • Log File: %s\n", pattern)
	}
	fmt.Printf("\n")

	// Print collector information
	printCollectorInfo()
	fmt.Printf("\n")
}

// printCollectorInfo displays the collector capabilities
func printCollectorInfo() {
	fmt.Printf("🎯 Collector Capabilities:\n")
	fmt.Printf("   • Full Collection - Collect all tickets from configured projects\n")
	fmt.Printf("   • Update Mode - Collect only recently updated tickets\n")
	fmt.Printf("   • Batch Processing - Configurable batch sizes for optimal performance\n")
	fmt.Printf("   • Data Export - JSON output compatible with aktis-receiver\n")
	fmt.Printf("   • Web Interface - Real-time monitoring and control\n")
}

// PrintShutdownBanner displays the application shutdown banner
func PrintShutdownBanner(serviceName string) {
	b := banner.New().
		SetStyle(banner.StyleDouble).
		SetBorderColor(banner.ColorPurple).
		SetTextColor(banner.ColorWhite).
		SetBold(true).
		SetWidth(42)

	b.PrintTopLine()
	b.PrintCenteredText("SHUTTING DOWN")
	b.PrintCenteredText(serviceName)
	b.PrintBottomLine()
	fmt.Println()
}

// PrintColorizedMessage prints a message with specified color
func PrintColorizedMessage(color, message string) {
	fmt.Printf("%s%s%s\n", color, message, banner.ColorReset)
}

// PrintSuccess prints a success message in green
func PrintSuccess(message string) {
	PrintColorizedMessage(banner.ColorGreen, fmt.Sprintf("✓ %s", message))
}

// PrintError prints an error message in red
func PrintError(message string) {
	PrintColorizedMessage(banner.ColorRed, fmt.Sprintf("✗ %s", message))
}

// PrintWarning prints a warning message in yellow
func PrintWarning(message string) {
	PrintColorizedMessage(banner.ColorYellow, fmt.Sprintf("⚠ %s", message))
}

// PrintInfo prints an info message in cyan
func PrintInfo(message string) {
	PrintColorizedMessage(banner.ColorCyan, fmt.Sprintf("ℹ %s", message))
}