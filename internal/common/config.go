package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Collector CollectorConfig `toml:"collector"`
	Storage   StorageConfig   `toml:"storage"`
	Logging   LoggingConfig   `toml:"logging"`
}

type CollectorConfig struct {
	Name        string `toml:"name"`
	Environment string `toml:"environment"`
	Port        int    `toml:"port"`
}

type StorageConfig struct {
	DatabasePath  string `toml:"database_path"`
	BackupDir     string `toml:"backup_dir"`
	RetentionDays int    `toml:"retention_days"`
}

type LoggingConfig struct {
	Level      string `toml:"level"`
	Format     string `toml:"format"`
	Output     string `toml:"output"`
	MaxSize    int    `toml:"max_size"`
	MaxBackups int    `toml:"max_backups"`
}

func DefaultConfig() *Config {
	execPath, _ := os.Executable()
	execDir := filepath.Dir(execPath)
	execName := filepath.Base(execPath)
	execName = execName[:len(execName)-len(filepath.Ext(execName))]

	defaultDBPath := filepath.Join(execDir, "data", execName+".db")

	return &Config{
		Collector: CollectorConfig{
			Name:        execName,
			Environment: "development",
			Port:        8080,
		},
		Storage: StorageConfig{
			DatabasePath:  defaultDBPath,
			BackupDir:     "./backups",
			RetentionDays: 90,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "text",
			Output:     "both",
			MaxSize:    100,
			MaxBackups: 3,
		},
	}
}

func LoadConfig(configFile string) (*Config, error) {
	config := DefaultConfig()

	if configFile == "" {
		// Auto-detect config file
		execPath, _ := os.Executable()
		execDir := filepath.Dir(execPath)
		execName := filepath.Base(execPath)
		execName = execName[:len(execName)-len(filepath.Ext(execName))]

		possiblePaths := []string{
			filepath.Join(execDir, execName+".toml"),
			filepath.Join(execDir, "config.toml"),
			"config.toml",
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				configFile = path
				break
			}
		}

		if configFile == "" {
			return config, nil
		}
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	if err := toml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	applyEnvOverrides(config)

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

func applyEnvOverrides(config *Config) {
	if dbPath := os.Getenv("DATABASE_PATH"); dbPath != "" {
		config.Storage.DatabasePath = dbPath
	}
	if backupDir := os.Getenv("BACKUP_DIR"); backupDir != "" {
		config.Storage.BackupDir = backupDir
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		config.Logging.Level = logLevel
	}
	if logFormat := os.Getenv("LOG_FORMAT"); logFormat != "" {
		config.Logging.Format = logFormat
	}
	if logOutput := os.Getenv("LOG_OUTPUT"); logOutput != "" {
		config.Logging.Output = logOutput
	}

	if port := os.Getenv("SERVER_PORT"); port != "" {
		if portNum, err := strconv.Atoi(port); err == nil {
			config.Collector.Port = portNum
		}
	}
}

func (c *Config) Validate() error {
	if c.Storage.DatabasePath == "" {
		return fmt.Errorf("storage database_path is required")
	}

	if c.Collector.Port <= 0 {
		c.Collector.Port = 8080
	}

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

func (c *Config) IsProduction() bool {
	return c.Logging.Level == "warn" || c.Logging.Level == "error" || c.Logging.Level == "fatal"
}
