package collector

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// Storage handles data persistence and retrievaltype Storage struct {
	config *StorageConfig
}

// TicketData represents stored ticket information with metadata
type TicketData struct {
	Key       string                 `json:"key"`
	Project   string                 `json:"project"`
	Data      map[string]interface{} `json:"data"`
	Collected time.Time              `json:"collected"`
	Updated   time.Time              `json:"updated"`
	Version   int                    `json:"version"`
	Hash      string                 `json:"hash"`
}

// ProjectDataset represents a collection of tickets for a project
type ProjectDataset struct {
	ProjectKey string                  `json:"project_key"`
	LastUpdate time.Time               `json:"last_update"`
	TotalCount int                     `json:"total_count"`
	Tickets    map[string]*TicketData  `json:"tickets"`
}

// NewStorage creates a new storage instancefunc NewStorage(config *StorageConfig) (*Storage, error) {
	storage := &Storage{config: config}
	
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	
	// Create backup directory if specified
	if config.BackupDir != "" {
		if err := os.MkdirAll(config.BackupDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create backup directory: %w", err)
		}
	}
	
	return storage, nil
}

// SaveTickets saves a batch of tickets to storagefunc (s *Storage) SaveTickets(projectKey string, tickets []TicketData) error {
	dataset, err := s.LoadProjectDataset(projectKey)
	if err != nil {
		dataset = &ProjectDataset{
			ProjectKey: projectKey,
			Tickets:    make(map[string]*TicketData),
		}
	}
	
	now := time.Now()
	for _, ticket := range tickets {
		// Update existing ticket or add new one
		if existing, exists := dataset.Tickets[ticket.Key]; exists {
			// Update existing ticket
			existing.Data = ticket.Data
			existing.Updated = now
			existing.Version++
		} else {
			// Add new ticket
			ticket.Collected = now
			ticket.Updated = now
			ticket.Version = 1
			dataset.Tickets[ticket.Key] = &ticket
			dataset.TotalCount++
		}
	}
	
	dataset.LastUpdate = now
	
	return s.SaveProjectDataset(dataset)
}

// LoadProjectDataset loads the dataset for a specific projectfunc (s *Storage) LoadProjectDataset(projectKey string) (*ProjectDataset, error) {
	filePath := s.getProjectFilePath(projectKey)
	
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &ProjectDataset{
				ProjectKey: projectKey,
				Tickets:    make(map[string]*TicketData),
			}, nil
		}
		return nil, fmt.Errorf("failed to read dataset file: %w", err)
	}
	
	var dataset ProjectDataset
	if err := json.Unmarshal(data, &dataset); err != nil {
		return nil, fmt.Errorf("failed to parse dataset: %w", err)
	}
	
	// Initialize Tickets map if nil
	if dataset.Tickets == nil {
		dataset.Tickets = make(map[string]*TicketData)
	}
	
	return &dataset, nil
}

// SaveProjectDataset saves the complete dataset for a projectfunc (s *Storage) SaveProjectDataset(dataset *ProjectDataset) error {
	filePath := s.getProjectFilePath(dataset.ProjectKey)
	
	// Create backup before saving
	if err := s.createBackup(filePath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	
	data, err := json.MarshalIndent(dataset, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dataset: %w", err)
	}
	
	// Write to temporary file first
	tempFile := filePath + ".tmp"
	if err := ioutil.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}
	
	// Atomic rename
	if err := os.Rename(tempFile, filePath); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}
	
	return nil
}

// GetLastUpdateTime returns the last update time for a projectfunc (s *Storage) GetLastUpdateTime(projectKey string) (time.Time, error) {
	dataset, err := s.LoadProjectDataset(projectKey)
	if err != nil {
		return time.Time{}, err
	}
	return dataset.LastUpdate, nil
}

// GetTicketKeys returns all ticket keys for a projectfunc (s *Storage) GetTicketKeys(projectKey string) ([]string, error) {
	dataset, err := s.LoadProjectDataset(projectKey)
	if err != nil {
		return nil, err
	}
	
	keys := make([]string, 0, len(dataset.Tickets))
	for key := range dataset.Tickets {
		keys = append(keys, key)
	}
	
	return keys, nil
}

// GetTicketsUpdatedSince returns tickets updated since a specific timefunc (s *Storage) GetTicketsUpdatedSince(projectKey string, since time.Time) ([]*TicketData, error) {
	dataset, err := s.LoadProjectDataset(projectKey)
	if err != nil {
		return nil, err
	}
	
	var tickets []*TicketData
	for _, ticket := range dataset.Tickets {
		if ticket.Updated.After(since) {
			tickets = append(tickets, ticket)
		}
	}
	
	return tickets, nil
}

// CleanupOldData removes old data based on retention policyfunc (s *Storage) CleanupOldData() error {
	if s.config.RetentionDays <= 0 {
		return nil
	}
	
	cutoffDate := time.Now().AddDate(0, 0, -s.config.RetentionDays)
	
	files, err := ioutil.ReadDir(s.config.DataDir)
	if err != nil {
		return fmt.Errorf("failed to read data directory: %w", err)
	}
	
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			filePath := filepath.Join(s.config.DataDir, file.Name())
			if file.ModTime().Before(cutoffDate) {
				if err := os.Remove(filePath); err != nil {
					return fmt.Errorf("failed to remove old file %s: %w", file.Name(), err)
				}
			}
		}
	}
	
	return nil
}

func (s *Storage) getProjectFilePath(projectKey string) string {
	return filepath.Join(s.config.DataDir, fmt.Sprintf("%s_tickets.json", strings.ToLower(projectKey)))
}

func (s *Storage) createBackup(filePath string) error {
	if s.config.BackupDir == "" {
		return nil
	}
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}
	
	// Read original file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	
	// Create backup filename with timestamp
	filename := filepath.Base(filePath)
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(s.config.BackupDir, fmt.Sprintf("%s.%s.bak", filename, timestamp))
	
	return ioutil.WriteFile(backupFile, data, 0644)
}