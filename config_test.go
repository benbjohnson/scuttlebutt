package scuttlebutt

import (
	"testing"

	"github.com/burntsushi/toml"
	"github.com/stretchr/testify/assert"
)

// Ensure that a configuration file can be decoded.
func TestConfig(t *testing.T) {
	str := `
[[account]]
name = "github_js"
language = "javascript"
consumer_key = "ABC"
consumer_secret = "123"

[[account]]
name = "github_go"
language = "go"
consumer_key = "DEF"
consumer_secret = "456"
`
	c := &Config{}
	_, err := toml.Decode(str, &c)
	assert.NoError(t, err)
	if assert.Equal(t, 2, len(c.Accounts)) {
		assert.Equal(t, "github_js", c.Accounts[0].Name)
		assert.Equal(t, "javascript", c.Accounts[0].Language)
		assert.Equal(t, "ABC", c.Accounts[0].ConsumerKey)
		assert.Equal(t, "123", c.Accounts[0].ConsumerSecret)

		assert.Equal(t, "github_go", c.Accounts[1].Name)
		assert.Equal(t, "go", c.Accounts[1].Language)
		assert.Equal(t, "DEF", c.Accounts[1].ConsumerKey)
		assert.Equal(t, "456", c.Accounts[1].ConsumerSecret)
	}
}
