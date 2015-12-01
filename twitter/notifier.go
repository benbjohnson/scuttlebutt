package twitter

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/benbjohnson/scuttlebutt"
	"github.com/kurrik/twittergo"
)

// Notifier represents a client to post messages to the Twitter API.
type Notifier struct {
	lastTweetTime time.Time

	Username string
	Language string

	Client interface {
		SendRequest(*http.Request) (*twittergo.APIResponse, error)
	}
}

// NewNotifier creates a new instance of Client authorized to a user.
func NewNotifier() *Notifier {
	return &Notifier{}
}

// Notify updates the authorized user's status. Returns the tweet ID on success.
func (n *Notifier) Notify(r *scuttlebutt.Repository) (*scuttlebutt.Message, error) {
	text := NotifyText(r)

	// Construct request.
	req, err := http.NewRequest("POST", "/1.1/statuses/update.json", strings.NewReader((url.Values{"status": {text}}).Encode()))
	if err != nil {
		return nil, fmt.Errorf("notify request: %s", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request.
	resp, err := n.Client.SendRequest(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %s", err)
	}
	defer resp.Body.Close()

	// Parse the response.
	var tweet twittergo.Tweet
	if err := resp.Parse(&tweet); err != nil {
		return nil, fmt.Errorf("parse: %s", err)
	}

	// Update last tweet time cache.
	n.lastTweetTime = tweet.CreatedAt()

	return &scuttlebutt.Message{ID: tweet.Id(), Text: text, RepositoryID: r.ID}, nil
}

// LastTweetTime returns the timestamp of the last tweet.
// Returns a cached version, if possible. Otherwise retrieves from Twitter.
func (n *Notifier) LastTweetTime() (time.Time, error) {
	// Return cached time, if available.
	if !n.lastTweetTime.IsZero() {
		return n.lastTweetTime, nil
	}

	// Construct request.
	q := make(url.Values)
	q.Set("screen_name", n.Username)
	q.Set("count", strconv.Itoa(1))
	req, err := http.NewRequest("GET", "/1.1/statuses/user_timeline.json", strings.NewReader(q.Encode()))
	if err != nil {
		return time.Time{}, fmt.Errorf("new request: %s", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request.
	resp, err := n.Client.SendRequest(req)
	if err != nil {
		return time.Time{}, fmt.Errorf("send request: %s", err)
	}
	defer resp.Body.Close()

	// Parse the response.
	var tweets twittergo.Timeline
	if err := resp.Parse(&tweets); err != nil {
		return time.Time{}, fmt.Errorf("parse: %s", err)
	}

	// If there's no tweets then return a zero ID with no error.
	if len(tweets) == 0 {
		return time.Time{}, nil
	}

	return tweets[0].CreatedAt(), nil
}

// NotifyText returns a tweet sized message for a repository.
func NotifyText(r *scuttlebutt.Repository) string {
	const maxLength = 140
	const format = "%s - %s %s"

	name, url := r.Name(), r.URL()

	// Calculate the remaining characters without the description.
	remaining := maxLength - len(fmt.Sprintf(format, name, "", url))

	// Shorten the description, if necessary.
	var description = strings.TrimSpace(r.Description)
	if remaining < 3 {
		description = ""
	} else if len(description) > remaining {
		description = strings.TrimSpace(description[:remaining-3]) + "..."
	}

	return fmt.Sprintf(format, name, description, url)
}
