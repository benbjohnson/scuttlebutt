package scuttlebutt

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/benbjohnson/scuttlebutt/internal"
	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
)

//go:generate protoc --gogo_out=. internal/internal.proto

var (
	// ErrRepositoryNotFound is returned when operating on a non-existent repo.
	ErrRepositoryNotFound = errors.New("repository not found")
)

// Store represents the data storage for storing messages received and sent.
// The store acts as a cache to the backing remote store for repository info.
type Store struct {
	path string
	db   *bolt.DB

	// The remote backing store.
	RemoteStore interface {
		Repository(id string) (*Repository, error)
	}
}

// NewStore returns a new instance of Store.
func NewStore(path string) *Store {
	return &Store{
		path: path,
	}
}

// Path returns the data path.
func (s *Store) Path() string { return s.path }

// Open opens and initializes the database.
func (s *Store) Open() error {
	// Open underlying data store.
	db, err := bolt.Open(s.path, 0666, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}
	s.db = db

	// Initialize all the required buckets.
	if err := s.db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("repositories"))
		tx.CreateBucketIfNotExists([]byte("meta"))
		return nil
	}); err != nil {
		s.Close()
		return err
	}

	return nil
}

// Close closes the store.
func (s *Store) Close() error {
	if s.db != nil {
		s.db.Close()
	}
	return nil
}

// AddMessage adds a message related to a repository.
// Retrieves repository data from the remote store, if needed.
func (s *Store) AddMessage(m *Message) error {
	if err := s.db.Update(func(tx *bolt.Tx) error {
		// Retrieve repository.
		r, err := s.repository(tx, m.RepositoryID)
		if err != nil {
			return err
		}

		// If repository is not in local store then fetch it remotely.
		if r == nil {
			repo, err := s.RemoteStore.Repository(m.RepositoryID)
			if err != nil {
				return fmt.Errorf("remote: %s", err)
			} else if repo == nil {
				return ErrRepositoryNotFound
			}

			// Convert to internal format.
			r = encodeRepository(repo)
		}

		// Ensure message doesn't already exist.
		for _, msg := range r.GetMessages() {
			if msg.GetID() == m.ID {
				return errDuplicateMessage
			}
		}

		// Append message.
		r.Messages = append(r.Messages, encodeMessage(m))

		// Update repository.
		if err := s.saveRepository(tx, r); err != nil {
			return err
		}
		return nil
	}); err == errDuplicateMessage {
		return nil // ignore duplicates
	} else if err != nil {
		return err
	}
	return nil
}

// Repository returns a repository by id.
func (s *Store) Repository(id string) (r *Repository, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		// Retrieve encoded entry.
		buf := tx.Bucket([]byte("repositories")).Get([]byte(id))
		if buf == nil {
			return nil
		}

		// Decode repository.
		var pb internal.Repository
		if err := proto.Unmarshal(buf, &pb); err != nil {
			return err
		}
		r = decodeRepository(&pb)

		return nil
	})
	return
}

// Repositories returns all repositories.
func (s *Store) Repositories() (a []*Repository, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("repositories")).Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var pb internal.Repository
			if err := proto.Unmarshal(v, &pb); err != nil {
				return err
			}
			a = append(a, decodeRepository(&pb))
		}

		return nil
	})
	return
}

// TopRepositories returns the most mentioned repositories by language.
func (s *Store) TopRepositories() (m map[string]*Repository, err error) {
	m = make(map[string]*Repository)

	err = s.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("repositories")).Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			// Decode repository.
			var r internal.Repository
			if err := proto.Unmarshal(v, &r); err != nil {
				return err
			}

			// Retrieve repository language.
			lang := r.GetLanguage()

			// Ignore marked repositories or repositories that have a lower message count.
			if r.GetNotified() {
				continue
			} else if m[lang] != nil && len(r.GetMessages()) <= len(m[lang].Messages) {
				continue
			}

			// Override repo.
			m[lang] = decodeRepository(&r)
		}
		return nil
	})
	return
}

// MarkNotified flags a repository as notified.
func (s *Store) MarkNotified(repositoryID string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		// Retrieve repository.
		r, err := s.repository(tx, repositoryID)
		if err != nil {
			return err
		} else if r == nil {
			return ErrRepositoryNotFound
		}

		// Update the notified flag.
		r.Notified = proto.Bool(true)

		// Perist repository.
		if err := s.saveRepository(tx, r); err != nil {
			return err
		}
		return nil
	})
}

// WriteTo writes the length and contents of the engine to w.
func (s *Store) WriteTo(w io.Writer) (n int64, err error) {
	tx, err := s.db.Begin(false)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Set content length header, if an HTTP response writer.
	if w, ok := w.(http.ResponseWriter); ok {
		w.Header().Set("Content-Length", strconv.FormatInt(tx.Size(), 10))
	}

	// Write data.
	return tx.WriteTo(w)
}

// repository returns a repository by ID.
func (s *Store) repository(tx *bolt.Tx, id string) (*internal.Repository, error) {
	v := tx.Bucket([]byte("repositories")).Get([]byte(id))
	if v == nil {
		return nil, nil
	}

	r := &internal.Repository{}
	if err := proto.Unmarshal(v, r); err != nil {
		return nil, err
	}
	return r, nil
}

// saveRepository saves a repository in the store.
func (s *Store) saveRepository(tx *bolt.Tx, r *internal.Repository) error {
	buf, err := proto.Marshal(r)
	if err != nil {
		return err
	}
	return tx.Bucket([]byte("repositories")).Put([]byte(r.GetID()), buf)
}

// encodeRepository encodes r into the internal format.
func encodeRepository(r *Repository) *internal.Repository {
	pb := &internal.Repository{
		ID:          proto.String(r.ID),
		Description: proto.String(r.Description),
		Language:    proto.String(r.Language),
		Notified:    proto.Bool(r.Notified),
		Messages:    make([]*internal.Message, len(r.Messages)),
	}

	for i, m := range r.Messages {
		pb.Messages[i] = encodeMessage(m)
	}

	return pb
}

// decodeRepository decodes pb into an application type.
func decodeRepository(pb *internal.Repository) *Repository {
	r := &Repository{
		ID:          pb.GetID(),
		Description: pb.GetDescription(),
		Language:    pb.GetLanguage(),
		Notified:    pb.GetNotified(),
		Messages:    make([]*Message, len(pb.Messages)),
	}

	for i, m := range pb.GetMessages() {
		r.Messages[i] = decodeMessage(m)
	}

	return r
}

// encodeMessage encodes m into the internal format.
func encodeMessage(m *Message) *internal.Message {
	return &internal.Message{
		ID:   proto.Uint64(m.ID),
		Text: proto.String(m.Text),
	}
}

// decodeMessage decodes pb into an application type.
func decodeMessage(pb *internal.Message) *Message {
	return &Message{
		ID:   pb.GetID(),
		Text: pb.GetText(),
	}
}

// errDuplicateMessage is a marker error.
var errDuplicateMessage = errors.New("duplicate message")
