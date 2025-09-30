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
	Port        int    `toml:"port"`
	WebPort     int    `toml:"-"` // Deprecated: use Port instead
}

type JiraConfig struct {
	Method        []string      `toml:"method"`
	BaseURL       string        `toml:"base_url"`
	Timeout       int           `toml:"timeout_seconds"`
	APIConfig     APIConfig     `toml:"api"`
	ScraperConfig ScraperConfig `toml:"scraper"`
}

type APIConfig struct {
	Username string `toml:"username"`
	APIToken string `toml:"api_token"`
}

type ScraperConfig struct {
	UseExistingBrowser bool   `toml:"use_existing_browser"`
	RemoteDebugPort    int    `toml:"remote_debug_port"`
	BrowserPath        string `toml:"browser_path"`
	UserDataDir        string `toml:"user_data_dir"`
	Headless           bool   `toml:"headless"`
	WaitBeforeScrape   int    `toml:"wait_before_scrape_ms"`
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
			Name:        execName,
			Environment: "development",
			SendLimit:   100,
			Port:        8080,
		},
		Jira: JiraConfig{
			Method:  []string{"api"},
			BaseURL: "https://your-company.atlassian.net",
			Timeout: 30,
			APIConfig: APIConfig{
				Username: "your-email@company.com",
				APIToken: "your-api-token",
			},
			ScraperConfig: ScraperConfig{
				UseExistingBrowser: false,
				RemoteDebugPort:    9222,
				BrowserPath:        "",
				UserDataDir:        "",
				Headless:           true,
				WaitBeforeScrape:   1000,
			},
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
	if jiraUsername := os.Getenv("JIRA_API_USERNAME"); jiraUsername != "" {
		config.Jira.APIConfig.Username = jiraUsername
	}
	if jiraToken := os.Getenv("JIRA_API_TOKEN"); jiraToken != "" {
		config.Jira.APIConfig.APIToken = jiraToken
	}
	if timeoutStr := os.Getenv("JIRA_TIMEOUT"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil {
			config.Jira.Timeout = timeout
		}
	}
	if useExisting := os.Getenv("JIRA_SCRAPER_USE_EXISTING"); useExisting != "" {
		config.Jira.ScraperConfig.UseExistingBrowser = useExisting == "true" || useExisting == "1"
	}
	if browserPath := os.Getenv("JIRA_SCRAPER_BROWSER_PATH"); browserPath != "" {
		config.Jira.ScraperConfig.BrowserPath = browserPath
	}
	if userDataDir := os.Getenv("JIRA_SCRAPER_USER_DATA_DIR"); userDataDir != "" {
		config.Jira.ScraperConfig.UserDataDir = userDataDir
	}
	if headless := os.Getenv("JIRA_SCRAPER_HEADLESS"); headless != "" {
		config.Jira.ScraperConfig.Headless = headless == "true" || headless == "1"
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
	if len(c.Jira.Method) == 0 {
		c.Jira.Method = []string{"api"}
	}

	for _, method := range c.Jira.Method {
		if method != "api" && method != "scraper" {
			return fmt.Errorf("jira method must be 'api' or 'scraper', got '%s'", method)
		}
	}

	if c.Jira.BaseURL == "" {
		return fmt.Errorf("jira base_url is required")
	}

	hasAPI := false
	for _, method := range c.Jira.Method {
		if method == "api" {
			hasAPI = true
			break
		}
	}

	if hasAPI {
		if c.Jira.APIConfig.Username == "" {
			return fmt.Errorf("jira.api.username is required when method includes 'api'")
		}
		if c.Jira.APIConfig.APIToken == "" {
			return fmt.Errorf("jira.api.api_token is required when method includes 'api'")
		}
	}

	// No additional validation needed for scraper when using existing browser
	// Remote debugging doesn't require user_data_dir

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

	// Support legacy web_port field and migrate to port
	if c.Collector.WebPort > 0 && c.Collector.Port == 0 {
		c.Collector.Port = c.Collector.WebPort
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

func (c *Config) UsesAPI() bool {
	for _, method := range c.Jira.Method {
		if method == "api" {
			return true
		}
	}
	return false
}

func (c *Config) UsesScraper() bool {
	for _, method := range c.Jira.Method {
		if method == "scraper" {
			return true
		}
	}
	return false
}

func (c *Config) GetPrimaryMethod() string {
	if len(c.Jira.Method) > 0 {
		return c.Jira.Method[0]
	}
	return "api"
}
