package scuttlebutt_test

import (
	"errors"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/benbjohnson/scuttlebutt"
	"github.com/davecgh/go-spew/spew"
)

// Ensure that duplicate messages are only recorded once.
func TestStore_AddMessage_Duplicate(t *testing.T) {
	s := OpenStore()
	defer s.Close()

	// Mock remote store.
	s.RemoteStore.RepositoryFn = func(id string) (*scuttlebutt.Repository, error) {
		return &scuttlebutt.Repository{ID: id}, nil
	}

	// Add duplicate messages.
	if err := s.AddMessage(&scuttlebutt.Message{ID: 1, Text: "A", RepositoryID: "github.com/user/repo"}); err != nil {
		t.Fatal(err)
	} else if err := s.AddMessage(&scuttlebutt.Message{ID: 1, Text: "A", RepositoryID: "github.com/user/repo"}); err != nil {
		t.Fatal(err)
	}

	// Verify that only one message was added.
	if r, err := s.Repository("github.com/user/repo"); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(r, &scuttlebutt.Repository{
		ID:       "github.com/user/repo",
		Messages: []*scuttlebutt.Message{{ID: 1, Text: "A"}},
	}) {
		t.Fatalf("unexpected repository: %s", spew.Sdump(r))
	}
}

// Ensure that an error on the remote store is passed back.
func TestStore_AddMessage_ErrRemoteStore(t *testing.T) {
	s := OpenStore()
	defer s.Close()

	// Mock remote store.
	s.RemoteStore.RepositoryFn = func(id string) (*scuttlebutt.Repository, error) {
		return nil, errors.New("marker")
	}

	// Add messages.
	err := s.AddMessage(&scuttlebutt.Message{ID: 1, Text: "A", RepositoryID: "github.com/benbjohnson/go1"})
	if err == nil || err.Error() != `remote: marker` {
		t.Fatalf("unexpected error: %s", err)
	}
}

// Ensure that a non-existent repository is ignored.
func TestStore_AddMessage_ErrRepositoryNotFound(t *testing.T) {
	s := OpenStore()
	defer s.Close()

	// Mock remote store.
	s.RemoteStore.RepositoryFn = func(id string) (*scuttlebutt.Repository, error) {
		return nil, nil
	}

	// Add message to
	err := s.AddMessage(&scuttlebutt.Message{ID: 1, Text: "A", RepositoryID: "github.com/benbjohnson/no-such-repo"})
	if err != scuttlebutt.ErrRepositoryNotFound {
		t.Fatalf("unexpected error: %s", err)
	}
}

// Ensure that a repository can be marked as notified.
func TestStore_MarkNotified(t *testing.T) {
	s := OpenStore()
	defer s.Close()

	// Mock remote store.
	s.RemoteStore.RepositoryFn = func(id string) (*scuttlebutt.Repository, error) {
		return &scuttlebutt.Repository{ID: id}, nil
	}

	// Add message to pull in repository from remote store.
	if err := s.AddMessage(&scuttlebutt.Message{ID: 1, Text: "A", RepositoryID: "github.com/user/repo"}); err != nil {
		t.Fatal(err)
	}

	// Mark repository as notified.
	if err := s.MarkNotified("github.com/user/repo"); err != nil {
		t.Fatal(err)
	}

	// Verify that repository has been updated.
	if r, err := s.Repository("github.com/user/repo"); err != nil {
		t.Fatal(err)
	} else if !r.Notified {
		t.Fatal("expected notified")
	}

}

// Ensure that messages can be added and then top repositories computed.
func TestStore_TopRepositories(t *testing.T) {
	s := OpenStore()
	defer s.Close()

	// Mock remote store.
	s.RemoteStore.RepositoryFn = func(id string) (*scuttlebutt.Repository, error) {
		r := &scuttlebutt.Repository{
			ID:          id,
			Description: "lorem ipsum",
		}

		switch id {
		case "github.com/benbjohnson/go1":
			r.Language = "go"
		case "github.com/benbjohnson/go2":
			r.Language = "go"
		case "github.com/benbjohnson/js1":
			r.Language = "javascript"
		}

		return r, nil
	}

	// Add messages.
	if err := s.AddMessage(&scuttlebutt.Message{ID: 1, Text: "A", RepositoryID: "github.com/benbjohnson/go1"}); err != nil {
		t.Fatal(err)
	} else if err := s.AddMessage(&scuttlebutt.Message{ID: 2, Text: "B", RepositoryID: "github.com/benbjohnson/go2"}); err != nil {
		t.Fatal(err)
	} else if err := s.AddMessage(&scuttlebutt.Message{ID: 3, Text: "C", RepositoryID: "github.com/benbjohnson/go2"}); err != nil {
		t.Fatal(err)
	} else if err := s.AddMessage(&scuttlebutt.Message{ID: 4, Text: "D", RepositoryID: "github.com/benbjohnson/js1"}); err != nil {
		t.Fatal(err)
	}

	// Compute top repositories by language.
	m, err := s.TopRepositories()
	if err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(m, map[string]*scuttlebutt.Repository{
		"go": &scuttlebutt.Repository{
			ID:          "github.com/benbjohnson/go2",
			Description: "lorem ipsum",
			Language:    "go",
			Messages: []*scuttlebutt.Message{
				{ID: 2, Text: "B"},
				{ID: 3, Text: "C"},
			},
		},
		"javascript": &scuttlebutt.Repository{
			ID:          "github.com/benbjohnson/js1",
			Description: "lorem ipsum",
			Language:    "javascript",
			Messages: []*scuttlebutt.Message{
				{ID: 4, Text: "D"},
			},
		},
	}) {
		t.Fatalf("unexpected repositories: %s", spew.Sdump(m))
	}
}

// Store represents a test wrapper for scuttlebutt.Store.
type Store struct {
	*scuttlebutt.Store
	RemoteStore RemoteStore
}

// NewStore returns a new instance of Store at a temporary location.
func NewStore() *Store {
	// Create temporary path.
	f, _ := ioutil.TempFile("", "scuttlebutt-")
	f.Close()
	os.Remove(f.Name())

	// Create test store.
	s := &Store{
		Store: scuttlebutt.NewStore(f.Name()),
	}
	s.Store.RemoteStore = &s.RemoteStore
	return s
}

// OpenStore returns a new, open instance of Store.
func OpenStore() *Store {
	s := NewStore()
	if err := s.Open(); err != nil {
		panic(err)
	}
	return s
}

// Close closes the store and removes the underlying data.
func (s *Store) Close() error {
	defer os.RemoveAll(s.Store.Path())
	return s.Store.Close()
}

// RemoteStore represents a mock implementation of Store.RemoteStore.
type RemoteStore struct {
	RepositoryFn func(id string) (*scuttlebutt.Repository, error)
}

func (s *RemoteStore) Repository(id string) (*scuttlebutt.Repository, error) {
	return s.RepositoryFn(id)
}
