package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Collector      CollectorConfig          `toml:"collector"`
	Jira           JiraConfig               `toml:"jira"`
	ProjectsList   ProjectsList             `toml:"projects"`
	ProjectConfigs map[string]ProjectConfig `toml:",inline"`
	Projects       []ProjectConfig          `toml:"-"`
	Storage        StorageConfig            `toml:"storage"`
	Logging        LoggingConfig            `toml:"logging"`
}

type CollectorConfig struct {
	Name        string `toml:"name"`
	Environment string `toml:"environment"`
	SendLimit   int    `toml:"send_limit"`
	WebPort     int    `toml:"web_port"`
}

type JiraConfig struct {
	BaseURL  string `toml:"base_url"`
	Username string `toml:"username"`
	APIToken string `toml:"api_token"`
	Timeout  int    `toml:"timeout_seconds"`
}

type ProjectsList struct {
	Projects []string `toml:"projects"`
}

type ProjectConfig struct {
	Key            string   `toml:"-"`
	Name           string   `toml:"name"`
	IssueTypes     []string `toml:"issue_types"`
	Statuses       []string `toml:"statuses"`
	MaxResults     int      `toml:"max_results"`
	IncludeHistory bool     `toml:"include_history"`
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
			Name:        "aktis-collector-jira",
			Environment: "development",
			SendLimit:   100,
			WebPort:     8080,
		},
		Jira: JiraConfig{
			BaseURL:  "https://your-company.atlassian.net",
			Username: "your-email@company.com",
			APIToken: "your-api-token",
			Timeout:  30,
		},
		Projects: []ProjectConfig{
			{
				Key:            "PROJ",
				Name:           "Main Project",
				IssueTypes:     []string{"Bug", "Story", "Task", "Epic"},
				Statuses:       []string{"To Do", "In Progress", "In Review", "Done"},
				MaxResults:     1000,
				IncludeHistory: true,
			},
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
		return config, nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	var rawConfig map[string]interface{}
	if err := toml.Unmarshal(data, &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := toml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if len(config.ProjectsList.Projects) > 0 {
		config.Projects = make([]ProjectConfig, 0, len(config.ProjectsList.Projects))
		for _, projectKey := range config.ProjectsList.Projects {
			if projectData, ok := rawConfig[projectKey]; ok {
				var projectConfig ProjectConfig
				projectBytes, err := toml.Marshal(projectData)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal project %s: %w", projectKey, err)
				}
				if err := toml.Unmarshal(projectBytes, &projectConfig); err != nil {
					return nil, fmt.Errorf("failed to parse project %s: %w", projectKey, err)
				}
				projectConfig.Key = projectKey
				config.Projects = append(config.Projects, projectConfig)
			} else {
				return nil, fmt.Errorf("project %s listed but configuration not found", projectKey)
			}
		}
	}

	applyEnvOverrides(config)

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

func applyEnvOverrides(config *Config) {
	if jiraURL := os.Getenv("JIRA_BASE_URL"); jiraURL != "" {
		config.Jira.BaseURL = jiraURL
	}
	if jiraUsername := os.Getenv("JIRA_USERNAME"); jiraUsername != "" {
		config.Jira.Username = jiraUsername
	}
	if jiraToken := os.Getenv("JIRA_API_TOKEN"); jiraToken != "" {
		config.Jira.APIToken = jiraToken
	}
	if timeoutStr := os.Getenv("JIRA_TIMEOUT"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil {
			config.Jira.Timeout = timeout
		}
	}

	if dbPath := os.Getenv("DATABASE_PATH"); dbPath != "" {
		config.Storage.DatabasePath = dbPath
	}
	if backupDir := os.Getenv("BACKUP_DIR"); backupDir != "" {
		config.Storage.BackupDir = backupDir
	}
	if sendLimit := os.Getenv("SEND_LIMIT"); sendLimit != "" {
		if limit, err := strconv.Atoi(sendLimit); err == nil {
			config.Collector.SendLimit = limit
		}
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
}

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
	if c.Storage.DatabasePath == "" {
		return fmt.Errorf("storage database_path is required")
	}
	if c.Collector.SendLimit <= 0 {
		c.Collector.SendLimit = 100
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
