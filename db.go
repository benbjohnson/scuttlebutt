package scuttlebutt

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/boltdb/bolt"
	"github.com/google/go-github/github"
)

// DB represents the data storage for storing messages received and sent.
type DB struct {
	*bolt.DB
}

// Open opens and initializes the database.
func (db *DB) Open(path string, mode os.FileMode) error {
	var err error
	db.DB, err = bolt.Open(path, mode)
	if err != nil {
		return err
	}

	// Initialize all the required buckets.
	return db.Update(func(tx *Tx) error {
		tx.CreateBucketIfNotExists([]byte("blacklist"))
		tx.CreateBucketIfNotExists([]byte("repositories"))
		tx.CreateBucketIfNotExists([]byte("meta"))
		tx.CreateBucketIfNotExists([]byte("status"))
		return nil
	})
}

// View executes a function in the context of a read-only transaction.
func (db *DB) View(fn func(*Tx) error) error {
	return db.DB.View(func(tx *bolt.Tx) error {
		return fn(&Tx{tx})
	})
}

// Update executes a function in the context of a writable transaction.
func (db *DB) Update(fn func(*Tx) error) error {
	return db.DB.Update(func(tx *bolt.Tx) error {
		return fn(&Tx{tx})
	})
}

// Tx represents a transaction.
type Tx struct {
	*bolt.Tx
}

// Meta retrieves a meta field by name.
func (tx *Tx) Meta(key string) string {
	return string(tx.Bucket([]byte("meta")).Get([]byte(key)))
}

// SetMeta sets the value of a meta field by name.
func (tx *Tx) SetMeta(key, value string) error {
	return tx.Bucket([]byte("meta")).Put([]byte(key), []byte(value))
}

// AccountStatus retrieves the status for an account by username.
func (tx *Tx) AccountStatus(username string) (*AccountStatus, error) {
	var status AccountStatus
	value := tx.Bucket([]byte("status")).Get([]byte(username))
	if len(value) > 0 {
		if err := json.Unmarshal(value, &status); err != nil {
			return nil, err
		}
	}
	return &status, nil
}

// SetAccountStatus updates the status of an account.
func (tx *Tx) SetAccountStatus(username string, status *AccountStatus) error {
	value, err := json.Marshal(status)
	if err != nil {
		return err
	}
	return tx.Bucket([]byte("status")).Put([]byte(username), value)
}

// NotifiableAccounts filters a list of Accounts by which have not notified in a given duration.
func (tx *Tx) NotifiableAccounts(accounts []*Account, duration time.Duration) ([]*Account, error) {
	var ret []*Account
	for _, account := range accounts {
		// If the last notification is less than the interval then skip this account.
		status, err := tx.AccountStatus(account.Username)
		if err != nil {
			return nil, fmt.Errorf("notifiable accounts error: %s", err)
		} else if status.NotifyTime.IsZero() || time.Now().Sub(status.NotifyTime) > duration {
			ret = append(ret, account)
		}
	}
	return ret, nil
}

// Repository retrieves a repository by ID.
func (tx *Tx) Repository(id string) (*Repository, error) {
	r := new(Repository)
	value := tx.Bucket([]byte("repositories")).Get([]byte(id))
	if value == nil {
		return nil, nil
	} else if err := json.Unmarshal(value, &r); err != nil {
		return nil, err
	}
	return r, nil
}

// PutRepository inserts a repository.
func (tx *Tx) PutRepository(r *Repository) error {
	value, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return tx.Bucket([]byte("repositories")).Put([]byte(r.ID), value)
}

// FindOrCreateRepository finds or creates the repository from GitHub and creates it locally.
func (tx *Tx) FindOrCreateRepository(id string) (*Repository, error) {
	// Ignore if repo already exists.
	r, err := tx.Repository(id)
	if err != nil {
		return nil, err
	} else if r != nil {
		return r, nil
	}

	// Look up repository and insert it.
	client := github.NewClient(nil)
	_, username, repositoryName := splitRepositoryID(id)
	repo, _, err := client.Repositories.Get(username, repositoryName)
	if err != nil {
		return nil, fmt.Errorf("create repo error: %s", err)
	}

	// Create repository.
	r = &Repository{ID: id}
	if repo.HTMLURL != nil {
		r.URL = *repo.HTMLURL
	}
	if repo.Language != nil {
		r.Language = *repo.Language
	}
	if repo.Description != nil {
		r.Description = *repo.Description
	}

	if err := tx.PutRepository(r); err != nil {
		return nil, fmt.Errorf("put repo error: %s", err)
	}
	return r, nil
}

// AddMessage inserts a message for an existing repository.
func (tx *Tx) AddMessage(repositoryID string, m *Message) error {
	var r Repository
	var bucket = tx.Bucket([]byte("repositories"))
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
	err := tx.Bucket([]byte("blacklist")).ForEach(func(k, _ []byte) error {
		blacklist[string(k)] = true
		return nil
	})
	if err != nil {
		return err
	}

	// Iterate over all repositories not on the blacklist.
	return tx.Bucket([]byte("repositories")).ForEach(func(k, v []byte) error {
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

// Blacklist retrieves a list of repository ids on the blacklist.
func (tx *Tx) Blacklist() []string {
	blacklist := make([]string, 0)
	tx.Bucket([]byte("blacklist")).ForEach(func(k, _ []byte) error {
		blacklist = append(blacklist, string(k))
		return nil
	})
	return blacklist
}

// AddToBlacklist adds a repository id to the blacklist.
func (tx *Tx) AddToBlacklist(repositoryID string) error {
	return tx.Bucket([]byte("blacklist")).Put([]byte(repositoryID), []byte{})
}

// RemoveFromBlacklist removes a repository id from the blacklist.
func (tx *Tx) RemoveFromBlacklist(repositoryID string) error {
	return tx.Bucket([]byte("blacklist")).Delete([]byte(repositoryID))
}
