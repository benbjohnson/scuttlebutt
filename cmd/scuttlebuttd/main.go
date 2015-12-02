package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/benbjohnson/scuttlebutt"
	"github.com/benbjohnson/scuttlebutt/github"
	"github.com/benbjohnson/scuttlebutt/twitter"
	"github.com/burntsushi/toml"
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
)

const (
	// DefaultPollInterval is the default time between Twitter polling.
	DefaultPollInterval = 30 * time.Second

	// DefaultNotifyInterval is the default time between individual account notifications.
	DefaultNotifyInterval = 4 * time.Hour

	// DefaultNotifyCheckInterval is the default time between notification checks.
	DefaultNotifyCheckInterval = 30 * time.Minute

	// DefaultAddr is the default HTTP bind address.
	DefaultAddr = ":5050"
)

func main() {
	m := NewMain()

	// Parse command line flags.
	if err := m.ParseFlags(os.Args[1:]); err != nil {
		fmt.Fprintln(m.Stderr, err)
		os.Exit(1)
	}

	// Execute program.
	if err := m.Run(); err != nil {
		fmt.Fprintln(m.Stderr, err)
		os.Exit(1)
	}

	// Wait indefinitely.
	<-(chan struct{})(nil)
}

// Main represents the main program execution.
type Main struct {
	// Data store
	store     *scuttlebutt.Store
	poller    *twitter.Poller
	notifiers []*twitter.Notifier

	// HTTP interface
	Listener net.Listener
	Handler  http.Handler

	// Close management
	wg      sync.WaitGroup
	closing chan struct{}

	// HTTP bind address
	Addr string

	// Parsed config
	Config *Config

	// Data and configuration paths.
	DataDir    string
	ConfigPath string

	// Duration between polling for mentions.
	PollInterval time.Duration

	// Time between individual account messages.
	NotifyInterval time.Duration

	// Time between checking if notification interval has passed.
	NotifyCheckInterval time.Duration

	// Input/output streams
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// NewMain returns a new instance of Main.
func NewMain() *Main {
	return &Main{
		PollInterval:        DefaultPollInterval,
		NotifyInterval:      DefaultNotifyInterval,
		NotifyCheckInterval: DefaultNotifyCheckInterval,

		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// Run executes the program.
func (m *Main) Run() error {
	logger := log.New(m.Stderr, "", log.LstdFlags)

	// Validate options.
	if m.DataDir == "" {
		return errors.New("data directory required")
	}

	// Create base directory, if not exists.
	if err := os.MkdirAll(m.DataDir, 0777); err != nil {
		return err
	}

	// Open data store.
	m.store = scuttlebutt.NewStore(filepath.Join(m.DataDir, "db"))
	m.store.RemoteStore = github.NewStore(m.Config.GitHub.Token)
	if err := m.store.Open(); err != nil {
		return fmt.Errorf("open store: %s", err)
	}

	// Initialize poller.
	m.poller = twitter.NewPoller()
	m.poller.Client = twittergo.NewClient(&oauth1a.ClientConfig{
		ConsumerKey:    m.Config.Twitter.Key,
		ConsumerSecret: m.Config.Twitter.Secret,
	}, nil)

	// Initialize notifiers for each account
	for _, acc := range m.Config.Accounts {
		client := twittergo.NewClient(
			&oauth1a.ClientConfig{
				ConsumerKey:    m.Config.Twitter.Key,
				ConsumerSecret: m.Config.Twitter.Secret,
			},
			oauth1a.NewAuthorizedConfig(acc.Key, acc.Secret),
		)

		n := twitter.NewNotifier()
		n.Username = acc.Username
		n.Language = acc.Language
		n.Client = client

		m.notifiers = append(m.notifiers, n)
	}

	// Open HTTP listener.
	ln, err := net.Listen("tcp", m.Addr)
	if err != nil {
		return err
	}
	m.Listener = ln
	m.Handler = &scuttlebutt.Handler{Store: m.store}

	// Run HTTP server is separate goroutine.
	logger.Printf("Listening on http://localhost%s", m.Addr)
	go http.Serve(m.Listener, m.Handler)

	// Create a poller & notify monitor.
	m.wg.Add(2)
	go m.runPoller()
	go m.runNotifier()

	return nil
}

// Close shuts down the program and all goroutines.
// Calling close twice will cause a panic.
func (m *Main) Close() error {
	// Close HTTP listener.
	if m.Listener != nil {
		m.Listener.Close()
		m.Listener = nil
	}

	// Notify goroutines of closing.
	close(m.closing)
	m.wg.Wait()

	return nil
}

// ParseFlags parses the command line flags.
func (m *Main) ParseFlags(args []string) error {
	// Parse command line options.
	fs := flag.NewFlagSet("scuttlebuttd", flag.ContinueOnError)
	fs.StringVar(&m.DataDir, "d", "", "data directory")
	fs.StringVar(&m.ConfigPath, "c", "", "config path")
	fs.StringVar(&m.Addr, "addr", ":5050", "HTTP port")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate options.
	if m.ConfigPath == "" {
		return errors.New("config path required")
	}

	// Read configuration.
	c, err := ParseConfigFile(m.ConfigPath)
	if err != nil {
		return fmt.Errorf("parse config file: %s", err)
	}

	// Copy config to program.
	m.Config = c

	return nil
}

// runPoller periodically searches for messages mentioning repositories.
func (m *Main) runPoller() {
	defer m.wg.Done()

	// Setup logging.
	logger := log.New(m.Stderr, "[poller] ", log.LstdFlags)

	var sinceID uint64
	for {
		if err := m.poll(&sinceID); err != nil {
			logger.Printf("poll error: %s", err)
		}

		// Wait for next interval or for shutdown signal.
		select {
		case <-time.After(m.PollInterval):
		case <-m.closing:
			return
		}
	}
}

// poll retrieves messages since a given ID.
// The sinceID is updated if any messages are retrieved.
func (m *Main) poll(sinceID *uint64) error {
	// Retrieve messages from twitter.
	messages, err := m.poller.Poll(*sinceID)
	if err != nil {
		return fmt.Errorf("poll: %s", err)
	}

	// Save messages to store.
	for _, message := range messages {
		if err := m.store.AddMessage(message); err == scuttlebutt.ErrRepositoryNotFound {
			// nop
		} else if err != nil {
			return fmt.Errorf("add message: %s", err)
		}

		// Update the highest "since id".
		if message.ID > *sinceID {
			*sinceID = message.ID
		}
	}

	return nil
}

// runNotifier periodically searches for messages mentioning repositories.
func (m *Main) runNotifier() {
	defer m.wg.Done()

	// Setup logging.
	logger := log.New(m.Stderr, "[notifier] ", log.LstdFlags)

	for {
		// Attempt to notify accounts with new repos!
		if err := m.notify(); err != nil {
			logger.Printf("notify error: %s", err)
		}

		// Wait for next interval or for shutdown signal.
		select {
		case <-time.After(m.NotifyCheckInterval):
		case <-m.closing:
			return
		}
	}
}

// notify sends a message to each account if enough time has elapsed.
func (m *Main) notify() error {
	// Setup logging.
	logger := log.New(m.Stderr, "[notifier] ", log.LstdFlags)

	// Retrieve top repositories by language.
	repos, err := m.store.TopRepositories()
	if err != nil {
		return fmt.Errorf("top repositories: %s", err)
	}

	// Iterate over each account.
	for _, n := range m.notifiers {
		// Retrieve last tweet time.
		lastTweetTime, err := n.LastTweetTime()
		if err != nil {
			logger.Printf("last tweet time error: username=%s, err=%s", n.Username, err)
			continue
		}

		// Skip notifier if last tweet time is within interval.
		if !lastTweetTime.IsZero() && time.Since(lastTweetTime) < DefaultNotifyInterval {
			continue
		}

		// Retrieve top language for the repository.
		r := repos[n.Language]
		if r == nil {
			continue
		}

		// Attempt to send message to account.
		if _, err := n.Notify(r); err != nil {
			logger.Printf("notify error: username=%s, repo=%s, text=%q, err=%s", n.Username, r.ID, twitter.NotifyText(r), err)
			continue
		}
		// logger.Printf("NOTIFY: username=%s, repo=%s", n.Username, r.ID)

		// Mark repository as notified.
		if err := m.store.MarkNotified(r.ID); err != nil {
			logger.Printf("mark notified error: username=%s, repo=%s, err=%s", r.ID, err)
			continue
		}
	}

	return nil
}

// Config represents the configuration.
type Config struct {
	Twitter struct {
		Key    string `toml:"key"`
		Secret string `toml:"secret"`
	} `toml:"twitter"`

	GitHub struct {
		Token string `toml:"token"`
	} `toml:"github"`

	Accounts []*Account `toml:"account"`
}

// ParseConfigFile parses the contents of path into a Config.
func ParseConfigFile(path string) (*Config, error) {
	c := &Config{}
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return nil, err
	}
	return c, nil
}

// Account represents a Twitter account that tweets occassional trending repos.
type Account struct {
	Username string `toml:"username"`
	Language string `toml:"language"`
	Key      string `toml:"key"`
	Secret   string `toml:"secret"`

	Client *twittergo.Client `toml:"-"`
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
