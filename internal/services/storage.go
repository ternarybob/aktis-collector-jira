package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "aktis-collector-jira/internal/common"
	. "aktis-collector-jira/internal/interfaces"

	bolt "go.etcd.io/bbolt"
)

const (
	ticketsBucket   = "tickets"
	metadataBucket  = "metadata"
	processedBucket = "processed"
	lastUpdateKey   = "last_update"
	sendCountKey    = "send_count"
	refreshCountKey = "refresh_count"
)

type storage struct {
	db     *bolt.DB
	config *StorageConfig
}

func NewStorage(config *StorageConfig) (Storage, error) {
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

	return &storage{
		db:     db,
		config: config,
	}, nil
}

func (s *storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *storage) SaveTickets(projectKey string, tickets map[string]*TicketData) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(ticketsBucket))
		now := time.Now()

		for _, ticket := range tickets {
			key := []byte(fmt.Sprintf("%s:%s", projectKey, ticket.Key))
			existing := bucket.Get(key)

			if existing == nil {
				ticket.Created = now.Format(time.RFC3339)
			}

			ticket.Updated = now.Format(time.RFC3339)

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

func (s *storage) LoadTickets(projectKey string) (map[string]*TicketData, error) {
	tickets := make(map[string]*TicketData)

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(ticketsBucket))
		prefix := []byte(fmt.Sprintf("%s:", projectKey))

		c := bucket.Cursor()
		for k, v := c.Seek(prefix); k != nil && len(k) >= len(prefix) && string(k[:len(prefix)]) == string(prefix); k, v = c.Next() {
			var ticket TicketData
			if err := json.Unmarshal(v, &ticket); err != nil {
				continue
			}
			tickets[ticket.Key] = &ticket
		}

		return nil
	})

	return tickets, err
}

func (s *storage) LoadAllTickets() (map[string]*TicketData, error) {
	tickets := make(map[string]*TicketData)

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(ticketsBucket))

		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var ticket TicketData
			if err := json.Unmarshal(v, &ticket); err != nil {
				continue
			}
			tickets[ticket.Key] = &ticket
		}

		return nil
	})

	return tickets, err
}

func (s *storage) GetLastUpdate(projectKey string) (string, error) {
	var lastUpdate time.Time

	err := s.db.View(func(tx *bolt.Tx) error {
		metaBucket := tx.Bucket([]byte(metadataBucket))
		key := []byte(fmt.Sprintf("%s:%s", projectKey, lastUpdateKey))
		data := metaBucket.Get(key)

		if data == nil {
			return nil
		}

		return lastUpdate.UnmarshalBinary(data)
	})

	if err != nil {
		return "", err
	}

	if lastUpdate.IsZero() {
		return "", nil
	}

	return lastUpdate.Format("2006-01-02 15:04"), nil
}
