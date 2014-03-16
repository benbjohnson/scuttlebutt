package scuttlebutt

// Config represents the configuration used by Scuttlebutt.
type Config struct {
	AppKey    string     `toml:"app_key"`
	AppSecret string     `toml:"app_secret"`
	Accounts  []*Account `toml:"account"`
}

// Account represents a Twitter account that tweets occassional trending repos.
type Account struct {
	Username string `toml:"username"`
	Language string `toml:"language"`
	Key      string `toml:"key"`
	Secret   string `toml:"secret"`
}

// Repository represents a code repository.
type Repository struct {
	ID       string     `json:"id"`
	Language string     `json:"language"`
	Messages []*Message `json:"messages"`
}

// Message represents a message associated with a project and language.
type Message struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type tweet struct {
	ID       int64                   `json:"id"`
	Text     string                  `json:"text"`
	Entities map[string]*tweetEntity `json:"entities"`
}

type tweetEntity struct {
	URLs []*tweetURLEntity `json:"urls"`
}

type tweetURLEntity struct {
	URL         string `json:"url"`
	ExpandedURL string `json:"expanded_url"`
}
