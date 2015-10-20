package github

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"code.google.com/p/goauth2/oauth"
	"github.com/benbjohnson/scuttlebutt"
	"github.com/google/go-github/github"
)

var (
	// ErrInvalidRepositoryID is returned when the repository ID does not conform
	// to a 3-segment github username/repository path.
	ErrInvalidRepositoryID = errors.New("invalid repository id")
)

// Store represents GitHub as a data store.
type Store struct {
	client *github.Client
}

// NewStore returns a new instance of Store.
func NewStore(token string) *Store {
	return &Store{
		client: github.NewClient(
			(&oauth.Transport{
				Token: &oauth.Token{AccessToken: token},
			}).Client(),
		),
	}
}

// Repository returns a repository by ID.
func (s *Store) Repository(id string) (*scuttlebutt.Repository, error) {
	// Parse repository ID.
	segments := strings.Split(id, "/")
	if len(segments) != 3 {
		return nil, ErrInvalidRepositoryID
	}
	username, name := segments[1], segments[2]

	// Retrieve repository data from GitHub.
	repo, _, err := s.client.Repositories.Get(username, name)
	if e, ok := err.(*github.ErrorResponse); ok && e.Response.StatusCode == http.StatusNotFound {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("get repository: %s", err)
	}

	// Create repository.
	r := &scuttlebutt.Repository{ID: id}
	if repo.Language != nil {
		r.Language = *repo.Language
	}
	if repo.Description != nil {
		r.Description = *repo.Description
	}

	return r, nil
}
