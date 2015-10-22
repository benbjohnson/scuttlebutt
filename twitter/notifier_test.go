package twitter_test

import (
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/benbjohnson/scuttlebutt"
	"github.com/benbjohnson/scuttlebutt/twitter"
	"github.com/davecgh/go-spew/spew"
	"github.com/kurrik/twittergo"
)

// Ensure the notifier can update the status for an account.
func TestNotifier_UpdateStatus(t *testing.T) {
	n := NewNotifier()

	// Mock transport to return a successful update.
	n.Client.SendRequestFn = func(r *http.Request) (*twittergo.APIResponse, error) {
		switch r.URL.Path {
		case "/1.1/statuses/user_timeline.json":
			return &twittergo.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(`[{"created_at": "Wed Aug 29 17:12:58 +0000 2012"}]`)),
			}, nil

		case "/1.1/statuses/update.json":
			return &twittergo.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(`{"id_str":"123","text":"hello!","created_at": "Wed Aug 29 17:12:58 +0000 2012"}`)),
			}, nil
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
			return nil, nil
		}
	}

	// Update account's status and check the response.
	if m, err := n.Notify(&scuttlebutt.Repository{
		ID:          "github.com/benbjohnson/proj",
		Description: "my awesome project",
	}); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(m, &scuttlebutt.Message{
		ID:           123,
		Text:         "proj - my awesome project https://github.com/benbjohnson/proj",
		RepositoryID: "github.com/benbjohnson/proj",
	}) {
		t.Fatalf("unexpected message: %s", spew.Sdump(m))
	}
}

// Notifier represents a test wrapper for twitter.Notifier.
type Notifier struct {
	*twitter.Notifier
	Client NotifierClient
}

// NewNotifier returns a new instance of Notifier.
func NewNotifier() *Notifier {
	n := &Notifier{Notifier: twitter.NewNotifier()}
	n.Notifier.Client = &n.Client
	return n
}

// NotifierClient represents a mock implementing Notifier.Client.
type NotifierClient struct {
	SendRequestFn func(*http.Request) (*twittergo.APIResponse, error)
}

func (c *NotifierClient) SendRequest(r *http.Request) (*twittergo.APIResponse, error) {
	return c.SendRequestFn(r)
}
