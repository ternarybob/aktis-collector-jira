package collector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	ticketsBucket    = "tickets"
	metadataBucket   = "metadata"
	processedBucket  = "processed"
	lastUpdateKey    = "last_update"
	sendCountKey     = "send_count"
	refreshCountKey  = "refresh_count"
)

type Storage struct {
	db     *bolt.DB
	config *StorageConfig
}

type TicketData struct {
	Key       string                 `json:"key"`
	Project   string                 `json:"project"`
	Data      map[string]interface{} `json:"data"`
	Collected time.Time              `json:"collected"`
	Updated   time.Time              `json:"updated"`
	Version   int                    `json:"version"`
	Hash      string                 `json:"hash"`
	Processed bool                   `json:"processed"`
	Sent      bool                   `json:"sent"`
	SentAt    *time.Time             `json:"sent_at,omitempty"`
}

type ProjectDataset struct {
	ProjectKey string                 `json:"project_key"`
	LastUpdate time.Time              `json:"last_update"`
	TotalCount int                    `json:"total_count"`
	Tickets    map[string]*TicketData `json:"tickets"`
}

func NewStorage(config *StorageConfig) (*Storage, error) {
	dbDir := filepath.Dir(config.DatabasePath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	if config.BackupDir != "" {
		if err := os.MkdirAll(config.BackupDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create backup directory: %w", err)
		}
	}

	db, err := bolt.Open(config.DatabasePath, 0600, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(ticketsBucket)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(metadataBucket)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(processedBucket)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create buckets: %w", err)
	}

	return &Storage{
		db:     db,
		config: config,
	}, nil
}

func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Storage) SaveTickets(projectKey string, tickets []TicketData) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(ticketsBucket))
		now := time.Now()

		for _, ticket := range tickets {
			ticket.Project = projectKey

			key := []byte(fmt.Sprintf("%s:%s", projectKey, ticket.Key))
			existing := bucket.Get(key)

			if existing != nil {
				var existingTicket TicketData
				if err := json.Unmarshal(existing, &existingTicket); err == nil {
					ticket.Version = existingTicket.Version + 1
					ticket.Collected = existingTicket.Collected
					ticket.Processed = existingTicket.Processed
					ticket.Sent = existingTicket.Sent
					ticket.SentAt = existingTicket.SentAt
				}
			} else {
				ticket.Version = 1
				ticket.Collected = now
			}

			ticket.Updated = now

			data, err := json.Marshal(ticket)
			if err != nil {
				return fmt.Errorf("failed to marshal ticket %s: %w", ticket.Key, err)
			}

			if err := bucket.Put(key, data); err != nil {
				return fmt.Errorf("failed to save ticket %s: %w", ticket.Key, err)
			}
		}

		metaBucket := tx.Bucket([]byte(metadataBucket))
		lastUpdateKey := []byte(fmt.Sprintf("%s:%s", projectKey, lastUpdateKey))
		lastUpdateData, _ := now.MarshalBinary()
		return metaBucket.Put(lastUpdateKey, lastUpdateData)
	})
}

func (s *Storage) LoadProjectDataset(projectKey string) (*ProjectDataset, error) {
	dataset := &ProjectDataset{
		ProjectKey: projectKey,
		Tickets:    make(map[string]*TicketData),
	}

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(ticketsBucket))
		prefix := []byte(fmt.Sprintf("%s:", projectKey))

		c := bucket.Cursor()
		for k, v := c.Seek(prefix); k != nil && len(k) >= len(prefix) && string(k[:len(prefix)]) == string(prefix); k, v = c.Next() {
			var ticket TicketData
			if err := json.Unmarshal(v, &ticket); err != nil {
				continue
			}
			dataset.Tickets[ticket.Key] = &ticket
			dataset.TotalCount++
		}

		metaBucket := tx.Bucket([]byte(metadataBucket))
		lastUpdateKey := []byte(fmt.Sprintf("%s:%s", projectKey, lastUpdateKey))
		if lastUpdateData := metaBucket.Get(lastUpdateKey); lastUpdateData != nil {
			dataset.LastUpdate.UnmarshalBinary(lastUpdateData)
		}

		return nil
	})

	return dataset, err
}

func (s *Storage) GetLastUpdateTime(projectKey string) (time.Time, error) {
	var lastUpdate time.Time

	err := s.db.View(func(tx *bolt.Tx) error {
		metaBucket := tx.Bucket([]byte(metadataBucket))
		key := []byte(fmt.Sprintf("%s:%s", projectKey, lastUpdateKey))
		data := metaBucket.Get(key)

		if data == nil {
			return fmt.Errorf("no last update time found for project %s", projectKey)
		}

		return lastUpdate.UnmarshalBinary(data)
	})

	return lastUpdate, err
}

func (s *Storage) GetTicketKeys(projectKey string) ([]string, error) {
	var keys []string

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(ticketsBucket))
		prefix := []byte(fmt.Sprintf("%s:", projectKey))

		c := bucket.Cursor()
		for k, v := c.Seek(prefix); k != nil && len(k) >= len(prefix) && string(k[:len(prefix)]) == string(prefix); k, v = c.Next() {
			var ticket TicketData
			if err := json.Unmarshal(v, &ticket); err != nil {
				continue
			}
			keys = append(keys, ticket.Key)
		}

		return nil
	})

	return keys, err
}

func (s *Storage) GetTicketsUpdatedSince(projectKey string, since time.Time) ([]*TicketData, error) {
	var tickets []*TicketData

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(ticketsBucket))
		prefix := []byte(fmt.Sprintf("%s:", projectKey))

		c := bucket.Cursor()
		for k, v := c.Seek(prefix); k != nil && len(k) >= len(prefix) && string(k[:len(prefix)]) == string(prefix); k, v = c.Next() {
			var ticket TicketData
			if err := json.Unmarshal(v, &ticket); err != nil {
				continue
			}
			if ticket.Updated.After(since) {
				tickets = append(tickets, &ticket)
			}
		}

		return nil
	})

	return tickets, err
}

func (s *Storage) GetUnsentTickets(projectKey string, limit int) ([]*TicketData, error) {
	var tickets []*TicketData

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(ticketsBucket))
		prefix := []byte(fmt.Sprintf("%s:", projectKey))

		c := bucket.Cursor()
		count := 0
		for k, v := c.Seek(prefix); k != nil && len(k) >= len(prefix) && string(k[:len(prefix)]) == string(prefix); k, v = c.Next() {
			if limit > 0 && count >= limit {
				break
			}

			var ticket TicketData
			if err := json.Unmarshal(v, &ticket); err != nil {
				continue
			}

			if !ticket.Sent {
				tickets = append(tickets, &ticket)
				count++
			}
		}

		return nil
	})

	return tickets, err
}

func (s *Storage) MarkTicketsAsSent(ticketKeys []string, projectKey string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(ticketsBucket))
		now := time.Now()

		for _, ticketKey := range ticketKeys {
			key := []byte(fmt.Sprintf("%s:%s", projectKey, ticketKey))
			data := bucket.Get(key)

			if data == nil {
				continue
			}

			var ticket TicketData
			if err := json.Unmarshal(data, &ticket); err != nil {
				continue
			}

			ticket.Sent = true
			ticket.SentAt = &now

			updatedData, err := json.Marshal(ticket)
			if err != nil {
				return fmt.Errorf("failed to marshal ticket %s: %w", ticketKey, err)
			}

			if err := bucket.Put(key, updatedData); err != nil {
				return fmt.Errorf("failed to update ticket %s: %w", ticketKey, err)
			}
		}

		metaBucket := tx.Bucket([]byte(metadataBucket))
		sendCountKey := []byte(fmt.Sprintf("%s:%s", projectKey, sendCountKey))

		var sendCount int
		if countData := metaBucket.Get(sendCountKey); countData != nil {
			json.Unmarshal(countData, &sendCount)
		}
		sendCount += len(ticketKeys)

		countData, _ := json.Marshal(sendCount)
		return metaBucket.Put(sendCountKey, countData)
	})
}

func (s *Storage) CleanupOldData() error {
	if s.config.RetentionDays <= 0 {
		return nil
	}

	cutoffDate := time.Now().AddDate(0, 0, -s.config.RetentionDays)

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(ticketsBucket))
		c := bucket.Cursor()

		var keysToDelete [][]byte

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var ticket TicketData
			if err := json.Unmarshal(v, &ticket); err != nil {
				continue
			}

			if ticket.Updated.Before(cutoffDate) && ticket.Sent {
				keysToDelete = append(keysToDelete, k)
			}
		}

		for _, key := range keysToDelete {
			if err := bucket.Delete(key); err != nil {
				return fmt.Errorf("failed to delete old ticket: %w", err)
			}
		}

		return nil
	})
}

func (s *Storage) Backup() error {
	if s.config.BackupDir == "" {
		return nil
	}

	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(s.config.BackupDir, fmt.Sprintf("collector_%s.db", timestamp))

	return s.db.View(func(tx *bolt.Tx) error {
		return tx.CopyFile(backupPath, 0600)
	})
}

func (s *Storage) SaveProjectDataset(dataset *ProjectDataset) error {
	tickets := make([]TicketData, 0, len(dataset.Tickets))
	for _, ticket := range dataset.Tickets {
		tickets = append(tickets, *ticket)
	}
	return s.SaveTickets(dataset.ProjectKey, tickets)
}