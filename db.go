package scuttlebutt

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/boltdb/bolt"
)

// DB represents the data storage for storing messages received and sent.
type DB struct {
	bolt.DB
}

// Open opens and initializes the database.
func (db *DB) Open(path string, mode os.FileMode) error {
	if err := db.DB.Open(path, mode); err != nil {
		return err
	}

	// Initialize all the required buckets.
	return db.Do(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists("blacklist")
		tx.CreateBucketIfNotExists("repositories")
		return nil
	})
}

// Tx represents a transaction.
type Tx bolt.Tx

// Repository retrieves a repository by ID.
func (tx *Tx) Repository(id string) (*Repository, error) {
	r := new(Repository)
	value := tx.Bucket("repositories").Get([]byte(id))
	return json.Unmarshal(value, &r)
}

// PutRepository inserts a repository.
func (tx *Tx) PutRepository(r *Repository) error {
	value, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return tx.Put("repositories", []byte(r.ID), value)
}

// AddMessage inserts a message for an existing repository.
func (tx *Tx) AddMessage(repositoryID string, m *Message) error {
	var r Repository
	var bucket = tx.Bucket("repositories")
	value := bucket.Get([]byte(repositoryID))
	if len(value) == 0 {
		return fmt.Errorf("repository not found: %s", repositoryID)
	}
	if err := json.Unmarshal(value, &r); err != nil {
		return err
	}

	// Append message.
	r.Messages = append(r.Messages, m)

	// Reinsert back into bucket.
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(repositoryID), b)
}

// ForEachRepository calls a given function for every repository.
func (tx *Tx) ForEachRepository(fn func(*Repository) error) error {
	// Create blacklist first.
	blacklist := make(map[string]bool)
	err := tx.Bucket("blacklist").ForEach(func(k, v []byte) error {
		blacklist[string(k)] = true
		return nil
	})
	if err != nil {
		return err
	}

	// Iterate over all repositories not on the blacklist.
	return tx.Bucket("repositories").ForEach(func(k, v []byte) error {
		if blacklist[string(k)] {
			return nil
		}

		var r Repository
		if err := json.Unmarshal(v, &r); err != nil {
			return err
		}
		return fn(&r)
	})
}

// TopRepositoriesByLanguage returns a map of top mentioned repositories by language.
// Only languages not on the blacklist are included.
func (tx *Tx) TopRepositoriesByLanguage() (map[string]*Repository, error) {
	m := make(map[string]*Repository)
	err := tx.ForEachRepository(func(r *Repository) error {
		current := m[r.Language]
		if current == nil || len(current.Messages) < len(r.Messages) {
			m[r.Language] = r
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return m, nil
}
