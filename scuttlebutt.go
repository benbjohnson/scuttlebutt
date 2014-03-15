package scuttlebutt

// Account represents a Twitter account that tweets occassional trending repos.
type Account struct {
	Name           string `toml:"name"`
	Language       string `toml:"language"`
	ConsumerKey    string `toml:"consumer_key"`
	ConsumerSecret string `toml:"consumer_secret"`
}

// Message represents a message associated with a project and language.
type Message struct {
	URL      string `json:"url"`
	Language string `json:"language"`
	Body     string `json:"body"`
}
