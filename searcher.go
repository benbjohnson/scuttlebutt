package scuttlebutt

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
)

// Searcher represents a Twitter searcher that retrieves new mentions of Github projects.
type Searcher struct {
	client *twittergo.Client
}

// NewSearcher creates a new Searcher instance.
func NewSearcher(key, secret string) *Searcher {
	config := &oauth1a.ClientConfig{
		ConsumerKey:    key,
		ConsumerSecret: secret,
	}
	return &Searcher{
		client: twittergo.NewClient(config, nil),
	}
}

// SearchURL returns the URL used for a search.
func (s *Searcher) SearchURL(sinceID int) *url.URL {
	var q = url.Values{"q": {"github.com"}}
	if sinceID > 0 {
		q.Set("since_id", strconv.Itoa(sinceID))
	}
	return &url.URL{Path: "/1.1/search/tweets.json", RawQuery: q.Encode()}
}

// Search finds new Github project mentions and executes the provided functions once for each message.
func (s *Searcher) Search(sinceID int) (*SearchResults, error) {
	var results = new(SearchResults)

	// Build search request.
	req, err := http.NewRequest("GET", s.SearchURL(sinceID).String(), nil)
	if err != nil {
		return nil, fmt.Errorf("request parse error: %s", err)
	}

	// Send request.
	resp, err := s.client.SendRequest(req)
	if err != nil {
		return nil, fmt.Errorf("twitter search error: %s", err)
	}

	// Convert to search results.
	var res twittergo.SearchResults
	if err := resp.Parse(&res); err != nil {
		return nil, fmt.Errorf("twitter search results error: %s", err)
	}

	// Iterate over each result.
	for _, tweet := range res.Statuses() {
		var result = new(SearchResult)

		// Extract ID & text.
		id, _ := tweet["id"].(int64)
		result.ID = int(id)
		if result.ID == 0 {
			log.Println("[search] invalid tweet id: ", marshalJSON(tweet))
			continue
		}
		result.Text, _ = tweet["text"].(string)
		result.Text = strings.Join(strings.Fields(result.Text), " ")

		// Extract entities.
		if entities, ok := tweet["entities"].(map[string]interface{}); ok {
			if urls, ok := entities["urls"].([]interface{}); ok {
				for _, u := range urls {
					if u, ok := u.(map[string]interface{}); ok {
						str, _ := u["expanded_url"].(string)
						url, err := url.Parse(strings.ToLower(str))
						if err != nil {
							continue
						}
						result.URLs = append(result.URLs, url)
					}
				}
			}
		}

		results.Results = append(results.Results, result)
	}

	// Log out rate limit.
	if resp.HasRateLimit() {
		results.RateLimit = int(resp.RateLimit())
		results.RateLimitRemaining = int(resp.RateLimitRemaining())
		results.RateLimitReset = resp.RateLimitReset()
	}

	return results, nil
}

type SearchResults struct {
	Results            []*SearchResult
	RateLimit          int
	RateLimitRemaining int
	RateLimitReset     time.Time
}

type SearchResult struct {
	ID   int
	Text string
	URLs []*url.URL
}
