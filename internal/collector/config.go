package collector

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the Jira collector configuration
type Config struct {
	Jira     JiraConfig      `json:"jira"`
	Projects []ProjectConfig `json:"projects"`
	Storage  StorageConfig   `json:"storage"`
}

// JiraConfig contains Jira connection settings
type JiraConfig struct {
	BaseURL  string `json:"base_url"`
	Username string `json:"username"`
	APIToken string `json:"api_token"`
	Timeout  int    `json:"timeout_seconds,omitempty"`
}

// ProjectConfig represents a Jira project to collect tickets from
type ProjectConfig struct {
	Key            string   `json:"key"`
	Name           string   `json:"name"`
	IssueTypes     []string `json:"issue_types,omitempty"`
	Statuses       []string `json:"statuses,omitempty"`
	MaxResults     int      `json:"max_results,omitempty"`
	IncludeHistory bool     `json:"include_history,omitempty"`
}

// StorageConfig contains data storage settings
type StorageConfig struct {
	DataDir       string `json:"data_dir"`
	BackupDir     string `json:"backup_dir,omitempty"`
	MaxFileSize   int64  `json:"max_file_size_mb,omitempty"`
	RetentionDays int    `json:"retention_days,omitempty"`
}

// LoadConfig loads configuration from file or returns default config
func LoadConfig(configFile string) (*Config, error) {
	if configFile != "" {
		return loadConfigFromFile(configFile)
	}
	return getDefaultConfig(), nil
}

func loadConfigFromFile(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func getDefaultConfig() *Config {
	return &Config{
		Jira: JiraConfig{
			BaseURL:  "https://your-company.atlassian.net",
			Username: "your-email@company.com",
			APIToken: "your-api-token",
			Timeout:  30,
		},
		Projects: []ProjectConfig{
			{
				Key:        "PROJ",
				Name:       "Main Project",
				IssueTypes: []string{"Bug", "Story", "Task", "Epic"},
				Statuses:   []string{"To Do", "In Progress", "In Review", "Done"},
				MaxResults: 1000,
			},
		},
		Storage: StorageConfig{
			DataDir:       "./data",
			BackupDir:     "./backups",
			MaxFileSize:   100,
			RetentionDays: 30,
		},
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Jira.BaseURL == "" {
		return fmt.Errorf("jira base_url is required")
	}
	if c.Jira.Username == "" {
		return fmt.Errorf("jira username is required")
	}
	if c.Jira.APIToken == "" {
		return fmt.Errorf("jira api_token is required")
	}
	if len(c.Projects) == 0 {
		return fmt.Errorf("at least one project must be configured")
	}
	for _, project := range c.Projects {
		if project.Key == "" {
			return fmt.Errorf("project key is required")
		}
	}
	if c.Storage.DataDir == "" {
		return fmt.Errorf("storage data_dir is required")
	}
	return nil
}
