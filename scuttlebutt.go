package scuttlebutt

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
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
func (a *Account) Notify(c *twittergo.Client, r *Repository) (uint64, error) {
	// Construct tweet text.
	msg := r.NotifyText()

	// Construct request.
	req, err := http.NewRequest("POST", "/1.1/statuses/update.json", strings.NewReader((url.Values{"status": {msg}}).Encode()))
	if err != nil {
		return 0, fmt.Errorf("notify request: %s", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request.
	resp, err := c.SendRequest(req)
	if err != nil {
		return 0, fmt.Errorf("notify send: %s", err)
	}

	// Parse the response.
	var tweet twittergo.Tweet
	if err := resp.Parse(&tweet); err != nil {
		return 0, fmt.Errorf("notify: %s", err)
	}

	return tweet.Id(), nil
}

// Client creates a new Twitter client for the account.
func (a *Account) Client(key, secret string) *twittergo.Client {
	return twittergo.NewClient(
		&oauth1a.ClientConfig{ConsumerKey: key, ConsumerSecret: secret},
		oauth1a.NewAuthorizedConfig(a.Key, a.Secret),
	)
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

// Name returns the name of the repository.
func (r *Repository) Name() string {
	return path.Base(r.ID)
}

// NotifyText returns a tweet sized message containing the repository name,
// description, and url.
func (r *Repository) NotifyText() string {
	var maxLength = 140
	var shortUrlLength = 23
	var padding = 2
	name, url := r.Name(), r.URL
	format := "%s - %s %s"

	// Calculate the remaining characters without the description.
	remaining := maxLength - len(fmt.Sprintf(format, name, "", strings.Repeat(" ", shortUrlLength))) - padding

	// Shorten the description, if necessary.
	var description = strings.TrimSpace(r.Description)
	if len(description) > remaining {
		description = strings.TrimSpace(description[:remaining-3]) + "..."
	}

	return fmt.Sprintf(format, name, description, url)
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
