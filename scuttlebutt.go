package scuttlebutt

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// Repository represents a code repository.
type Repository struct {
	ID          string
	Description string
	Language    string
	Notified    bool
	Messages    []*Message
}

// Name returns the name of the repository.
func (r *Repository) Name() string { return path.Base(r.ID) }

// URL returns the URL for the repository.
func (r *Repository) URL() string { return "https://" + r.ID }

// Repositories represents a sortable list of repositories.
type Repositories []*Repository

func (p Repositories) Len() int           { return len(p) }
func (p Repositories) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p Repositories) Less(i, j int) bool { return p[i].ID < p[j].ID }

// Message represents a message associated with a project and language.
type Message struct {
	ID           uint64
	Text         string
	RepositoryID string
}

// Extracts the repository identifier from a given URL.
func ExtractRepositoryID(u *url.URL) (string, error) {
	sections := strings.Split(path.Clean(u.Path), "/")
	if len(sections) != 3 {
		return "", fmt.Errorf("invalid section count: %d", len(sections))
	}
	host, username, repositoryName := u.Host, sections[1], sections[2]

	// Validate host & username.
	switch host {
	case "github.com", "www.github.com":
	default:
		return "", fmt.Errorf("invalid host: %s", host)
	}
	switch username {
	case "blog", "explore":
		return "", fmt.Errorf("invalid username: %s", username)
	}

	// Rejoin sections and return.
	return path.Join(host, username, repositoryName), nil
}
