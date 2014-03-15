package scuttlebutt

import (
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
		tx.CreateBucketIfNotExists("incoming")
		tx.CreateBucketIfNotExists("outgoing")
		return nil
	})
}
