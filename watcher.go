package scuttlebutt

import (
	"time"

	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
)

// DefaultWatcherInterval is the default time between Twitter searches.
const DefaultWatcherInterval = 5 * time.Second

// Watcher represents a Twitter search watcher that retrieves new mentions of Github projects.
type Watcher struct {
	Interval time.Duration
	client   *twittergo.Client
}

// NewWatcher creates a new Watcher instance.
func NewWatcher(key, secret string) *Watcher {
	config := &oauth1a.ClientConfig{
		ConsumerKey:    key,
		ConsumerSecret: secret,
	}
	return &Watcher{client: twittergo.NewClient(config, nil)}
}

// Watch watches for new Github project mentions and executes the provided functions once for each message.
func (w *Watcher) Watch(fn func(m *Message)) {

}
