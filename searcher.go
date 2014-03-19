package scuttlebutt

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"regexp"

	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
)

// Matches a repository identifier in a URL.
var repositoryIDRegexp = regexp.MustCompile(`(github\.com)\/([^\/]+)\/([^\/]+)\/?$`)

// Searcher represents a Twitter searcher that retrieves new mentions of Github projects.
type Searcher struct {
	db     *DB
	client *twittergo.Client
}

// NewSearcher creates a new Searcher instance.
func NewSearcher(db *DB, key, secret string) *Searcher {
	config := &oauth1a.ClientConfig{
		ConsumerKey:    key,
		ConsumerSecret: secret,
	}
	return &Searcher{
		db:     db,
		client: twittergo.NewClient(config, nil),
	}
}

// Search finds new Github project mentions and executes the provided functions once for each message.
func (s *Searcher) Search(fn func(repositoryID string, m *Message)) error {
	return s.db.Do(func(tx *Tx) error {
		sinceID := tx.Meta("LastTweetID")

		// Build search request.
		var u = url.URL{
			Path:     "/1.1/search/tweets.json",
			RawQuery: (url.Values{"q": {"github.com"}, "since_id": {sinceID}}).Encode(),
		}
		log.Println("[search]", u.String())
		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			return fmt.Errorf("request parse error: %s", err)
		}

		// Send request.
		resp, err := s.client.SendRequest(req)
		if err != nil {
			return fmt.Errorf("twitter search error: %s", err)
		}

		// Convert to search results.
		var results twittergo.SearchResults
		if err := resp.Parse(&results); err != nil {
			return fmt.Errorf("twitter search results error: %s", err)
		}

		// Iterate over each result.
		for _, tweet := range results.Statuses() {
			var m Message

			// Extract ID.
			m.ID, _ = tweet["id_str"].(string)
			if m.ID == "" {
				return fmt.Errorf("invalid tweet id: %q", tweet["id_str"])
			}

			// Update the last tweet id.
			sinceID = m.ID

			// Extract entities.
			if entities, ok := tweet["entities"].(map[string]interface{}); ok {
				if urls, ok := entities["urls"].([]interface{}); ok {
					for _, u := range urls {
						if u, ok := u.(map[string]interface{}); ok {
							str, _ := u["expanded_url"].(string)

							// Extract repository identifier from URL.
							repositoryID := s.extractRepositoryID(str)
							if repositoryID == "" {
								continue
							}

							// Create the repository and add the message.
							r, err := tx.CreateRepositoryIfNotExists(repositoryID)
							if err != nil {
								log.Println("create repo error:", err)
								continue
							}

							// Add message to repo.
							r.Messages = append(r.Messages, &m)

							// Update repository.
							if err := tx.PutRepository(r); err != nil {
								return fmt.Errorf("update repo error: %s", err)
							}
						}
					}
				}
			}
			// TODO: Call function.
		}

		// Update the last tweet id.
		log.Println("set last:", sinceID)
		if err := tx.SetMeta("LastTweetID", sinceID); err != nil {
			return fmt.Errorf("set last tweet id error: %s", err)
		}

		// Log out rate limit.
		if resp.HasRateLimit() {
			log.Printf("[rate limit] %v / %v / %v\n", resp.RateLimit(), resp.RateLimitRemaining(), resp.RateLimitReset())
		}
		return nil
	})
}

// Extracts the repository identifier from a given URL.
func (s *Searcher) extractRepositoryID(str string) string {
	m := repositoryIDRegexp.FindStringSubmatch(str)
	if m == nil {
		return ""
	}

	// Extract parts of the repository id.
	host, username, project := m[1], m[2], m[3]

	// Ignore certain usernames.
	switch username {
	case "blog", "explore":
		return ""
	}

	// Rejoin sections and return.
	return path.Join(host, username, project)
}
