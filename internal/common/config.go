package common

import (
	"fmt"
	"os"
	"strconv"

	"aktis-collector-jira/internal/collector"
)

// AppConfig holds the complete application configuration
type AppConfig struct {
	Collector *collector.Config `json:"collector"`
	Logging   LoggingConfig     `json:"logging"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	Output     string `json:"output"`
	MaxSize    int    `json:"max_size"`
	MaxBackups int    `json:"max_backups"`
}

// DefaultAppConfig returns a configuration with sensible defaults
func DefaultAppConfig() *AppConfig {
	return &AppConfig{
		Collector: getDefaultCollectorConfig(),
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "text",
			Output:     "both",
			MaxSize:    100, // MB
			MaxBackups: 3,
		},
	}
}

func getDefaultCollectorConfig() *collector.Config {
	return collector.DefaultConfig()
}

// LoadFromFile loads configuration with priority: defaults -> JSON -> environment -> command line
func LoadFromFile(filename string) (*AppConfig, error) {
	return loadConfigWithPriority(filename)
}

// LoadConfig loads configuration with priority: defaults -> JSON -> environment -> command line
func LoadConfig(filename string) (*AppConfig, error) {
	return loadConfigWithPriority(filename)
}

// loadConfigWithPriority implements the priority loading pattern
func loadConfigWithPriority(filename string) (*AppConfig, error) {
	// 1. Start with defaults
	config := DefaultAppConfig()

	// 2. Load collector configuration if file exists (JSON format for now)
	if filename != "" {
		collectorConfig, err := collector.LoadConfig(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to load collector config: %w", err)
		}
		config.Collector = collectorConfig
	}

	// 3. Apply environment variable overrides
	applyEnvOverrides(config)

	// 4. Command line overrides would be applied by the caller (main.go)

	// 5. Validate final configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// applyEnvOverrides applies environment variable overrides to configuration
func applyEnvOverrides(config *AppConfig) {
	// Jira configuration overrides
	if jiraURL := os.Getenv("JIRA_BASE_URL"); jiraURL != "" {
		config.Collector.Jira.BaseURL = jiraURL
	}
	if jiraUsername := os.Getenv("JIRA_USERNAME"); jiraUsername != "" {
		config.Collector.Jira.Username = jiraUsername
	}
	if jiraToken := os.Getenv("JIRA_API_TOKEN"); jiraToken != "" {
		config.Collector.Jira.APIToken = jiraToken
	}
	if timeoutStr := os.Getenv("JIRA_TIMEOUT"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil {
			config.Collector.Jira.Timeout = timeout
		}
	}

	// Storage configuration overrides
	if dbPath := os.Getenv("DATABASE_PATH"); dbPath != "" {
		config.Collector.Storage.DatabasePath = dbPath
	}
	if backupDir := os.Getenv("BACKUP_DIR"); backupDir != "" {
		config.Collector.Storage.BackupDir = backupDir
	}
	if sendLimit := os.Getenv("SEND_LIMIT"); sendLimit != "" {
		if limit, err := strconv.Atoi(sendLimit); err == nil {
			config.Collector.Collector.SendLimit = limit
		}
	}

	// Logging configuration overrides
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		config.Logging.Level = logLevel
	}
	if logFormat := os.Getenv("LOG_FORMAT"); logFormat != "" {
		config.Logging.Format = logFormat
	}
	if logOutput := os.Getenv("LOG_OUTPUT"); logOutput != "" {
		config.Logging.Output = logOutput
	}
}

// Validate checks if the configuration is valid
func (c *AppConfig) Validate() error {
	// Validate collector config
	if err := c.Collector.Validate(); err != nil {
		return fmt.Errorf("collector config validation failed: %w", err)
	}

	// Validate logging config
	validLogLevels := []string{"debug", "info", "warn", "error", "fatal", "panic"}
	validLevel := false
	for _, level := range validLogLevels {
		if c.Logging.Level == level {
			validLevel = true
			break
		}
	}
	if !validLevel {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}

	validOutputs := []string{"console", "file", "both"}
	validOutput := false
	for _, output := range validOutputs {
		if c.Logging.Output == output {
			validOutput = true
			break
		}
	}
	if !validOutput {
		return fmt.Errorf("invalid log output: %s", c.Logging.Output)
	}

	return nil
}

// GetCollectorConfig returns the collector configuration
func (c *AppConfig) GetCollectorConfig() *collector.Config {
	return c.Collector
}

// IsProduction returns true if running in production mode
func (c *AppConfig) IsProduction() bool {
	// Check if log level indicates production (warn, error, fatal)
	return c.Logging.Level == "warn" || c.Logging.Level == "error" || c.Logging.Level == "fatal"
}
