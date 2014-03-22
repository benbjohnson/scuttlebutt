package scuttlebutt

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"
)

// Config represents the configuration used by Scuttlebutt.
type Config struct {
	AppKey    string     `toml:"app_key"`
	AppSecret string     `toml:"app_secret"`
	Interval  Duration   `toml:"interval"`
	Accounts  []*Account `toml:"account"`
}

// Account represents a Twitter account that tweets occassional trending repos.
type Account struct {
	Username string `toml:"username"`
	Language string `toml:"language"`
	Key      string `toml:"key"`
	Secret   string `toml:"secret"`
}

// Notify sends a tweet for an account about a given repository.
func (a *Account) Notify(r *Repository) error {
	// TODO(benbjohnson): Create client.
	// TODO(benbjohnson): Construct tweet text.
	// TODO(benbjohnson): Send tweet.
	panic("NOT IMPLEMENTED: Account.Notify()")
	return nil
}

// AccountStatus represents status information about a given account.
type AccountStatus struct {
	NotifyTime time.Time `json:"notifyTime"`
}

// Repository represents a code repository.
type Repository struct {
	ID          string     `json:"id"`
	URL         string     `json:"url"`
	Description string     `json:"description"`
	Language    string     `json:"language"`
	Messages    []*Message `json:"messages"`
}

// Message represents a message associated with a project and language.
type Message struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}

// Duration is a helper type for unmarshaling durations in TOML.
type Duration time.Duration

func (d *Duration) UnmarshalText(text []byte) error {
	duration, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = Duration(duration)
	return nil
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

func splitRepositoryID(id string) (string, string, string) {
	s := strings.Split(id, "/")
	if len(s) != 3 {
		return "", "", ""
	}
	return s[0], s[1], s[2]
}

func marshalJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
