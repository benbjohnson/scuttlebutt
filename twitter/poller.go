package twitter

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/benbjohnson/scuttlebutt"
	"github.com/kurrik/twittergo"
)

// Poller represents polling client for the Twitter API.
type Poller struct {
	Client interface {
		SendRequest(*http.Request) (*twittergo.APIResponse, error)
	}
}

// NewPoller creates a new instance of Poller.
func NewPoller() *Poller {
	return &Poller{}
}

// Poll returns new messages since a given message ID.
func (p *Poller) Poll(sinceID uint64) ([]*scuttlebutt.Message, error) {
	// Send request.
	resp, err := p.Client.SendRequest(NewSearchRequest(sinceID))
	if err != nil {
		return nil, fmt.Errorf("send request: %s", err)
	}
	defer resp.Body.Close()

	// Convert to search results.
	var res twittergo.SearchResults
	if err := resp.Parse(&res); err != nil {
		return nil, fmt.Errorf("twitter search results error: %s", err)
	}

	// Convert search results to messages.
	var messages []*scuttlebutt.Message
	for _, tweet := range res.Statuses() {
		m := encodeTweet(tweet)
		if m.RepositoryID == "" {
			continue
		}
		messages = append(messages, m)
	}

	return messages, nil
}

func encodeTweet(tweet twittergo.Tweet) *scuttlebutt.Message {
	m := &scuttlebutt.Message{
		ID:   uint64(tweet["id"].(int64)),
		Text: tweet["text"].(string),
	}

	// Extract entities.
	if entities, ok := tweet["entities"].(map[string]interface{}); ok {
		if urls, ok := entities["urls"].([]interface{}); ok {
		loop:
			for _, u := range urls {
				if u, ok := u.(map[string]interface{}); ok {
					expandedURL, _ := u["expanded_url"].(string)

					// Convert to URL.
					u, err := url.Parse(strings.ToLower(expandedURL))
					if err != nil {
						continue
					}

					// Only keep the first two parts of the path.
					segments := strings.Split(u.Path, "/")
					if len(segments) != 3 {
						continue
					}

					m.RepositoryID = "github.com/" + segments[1] + "/" + segments[2]
					break loop
				}
			}
		}
	}

	return m
}

// NewSearchRequest returns a new HTTP request.
func NewSearchRequest(sinceID uint64) *http.Request {
	// Build query string.
	q := url.Values{"q": {"github.com"}}
	if sinceID > 0 {
		q.Set("since_id", strconv.FormatUint(sinceID, 10))
	}

	// Build URL object.
	u := &url.URL{Path: "/1.1/search/tweets.json", RawQuery: q.Encode()}

	// Build the request object. This really shouldn't error.
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		panic(err)
	}
	return req
}
