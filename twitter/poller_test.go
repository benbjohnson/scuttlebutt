package twitter_test

import (
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/benbjohnson/scuttlebutt/twitter"
	"github.com/benbjohnson/scuttlebutt"
	"github.com/davecgh/go-spew/spew"
	"github.com/kurrik/twittergo"
)

// Ensure the poller can retrieve new messages.
func TestPoller_Poll(t *testing.T) {
	p := NewPoller()

	// Mock transport to return a successful update.
	p.Client.SendRequestFn = func(*http.Request) (*twittergo.APIResponse, error) {
		return &twittergo.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(`{"statuses":[{"id":123,"text":"hello!","entities":{"urls":[{"expanded_url":"https://github.com/benbjohnson/proj"}]}}]}`)),
		}, nil
	}

	// Search for statuses and check the response.
	if messages, err := p.Poll(0); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(messages, []*scuttlebutt.Message{
		{ID: 123, Text: "hello!", RepositoryID: "github.com/benbjohnson/proj"},
	}) {
		t.Fatalf("unexpected statues: %s", spew.Sdump(messages))
	}
}

// Poller represents a test wrapper for twitter.Poller.
type Poller struct {
	*twitter.Poller
	Client PollerClient
}

// NewPoller returns a new instance of Poller.
func NewPoller() *Poller {
	p := &Poller{Poller: twitter.NewPoller()}
	p.Poller.Client = &p.Client
	return p
}

// PollerClient represents a mock implementing Poller.Client.
type PollerClient struct {
	SendRequestFn func(*http.Request) (*twittergo.APIResponse, error)
}

func (c *PollerClient) SendRequest(r *http.Request) (*twittergo.APIResponse, error) {
	return c.SendRequestFn(r)
}
