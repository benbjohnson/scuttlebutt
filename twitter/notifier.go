package twitter

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/benbjohnson/scuttlebutt"
	"github.com/kurrik/twittergo"
)

// Notifier represents a client to post messages to the Twitter API.
type Notifier struct {
	Client interface {
		SendRequest(*http.Request) (*twittergo.APIResponse, error)
	}
}

// NewNotifier creates a new instance of Client authorized to a user.
func NewNotifier() *Notifier {
	return &Notifier{}
}

// userConfig := oauth1a.NewAuthorizedConfig(key, secret)

// return &Client{
// 	Transport: twittergo.NewClient(&oauth1a.ClientConfig{
// 		ConsumerKey:    consumerKey,
// 		ConsumerSecret: consumerSecret,
// 	}, userConfig),
// }


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

	// Parse the response.
	var tweet twittergo.Tweet
	if err := resp.Parse(&tweet); err != nil {
		return nil, fmt.Errorf("parse: %s", err)
	}

	return &scuttlebutt.Message{ID: tweet.Id(), Text: text, RepositoryID: r.ID}, nil
}

// NotifyText returns a tweet sized message for a repository.
func NotifyText(r *scuttlebutt.Repository) string {
	const maxLength = 140
	const shortUrlLength = 23
	const padding = 2
	const format = "%s - %s %s"

	name, url := r.Name(), r.URL()

	// Calculate the remaining characters without the description.
	remaining := maxLength - len(fmt.Sprintf(format, name, "", strings.Repeat(" ", shortUrlLength))) - padding

	// Shorten the description, if necessary.
	var description = strings.TrimSpace(r.Description)
	if len(description) > remaining {
		description = strings.TrimSpace(description[:remaining-3]) + "..."
	}

	return fmt.Sprintf(format, name, description, url)
}

