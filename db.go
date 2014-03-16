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
	accounts []*Account
}

// Open opens and initializes the database.
func (db *DB) Open(path string, mode os.FileMode) error {
	if err := db.DB.Open(path, mode); err != nil {
		return err
	}

	// Initialize all the required buckets.
	return db.Do(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists("repositories")
		return nil
	})
}

// Repository retrieves a repository by ID.
func (db *DB) Repository(id string) (*Repository, error) {
	r := new(Repository)
	err := db.With(func(tx *bolt.Tx) error {
		value := tx.Bucket("repositories").Get([]byte(id))
		return json.Unmarshal(value, &r)
	})
	return r, err
}

// PutRepository inserts a repository.
func (db *DB) PutRepository(r *Repository) error {
	value, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return db.Put("repositories", []byte(r.ID), value)
}

// AddMessage inserts a message for an existing repository.
func (db *DB) AddMessage(repositoryID string, m *Message) error {
	return db.Do(func(tx *bolt.Tx) error {
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
	})
}

// ForEachRepository calls a given function for every repository.
func (db *DB) ForEachRepository(fn func(*Repository) error) error {
	return db.With(func(tx *bolt.Tx) error {
		return tx.Bucket("repositories").ForEach(func(k, v []byte) error {
			var r Repository
			if err := json.Unmarshal(v, &r); err != nil {
				return err
			}
			return fn(&r)
		})
	})
}
