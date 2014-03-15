package scuttlebutt

// Config represents the configuration used by Scuttlebutt.
type Config struct {
	Accounts []*Account `toml:"account"`
}
