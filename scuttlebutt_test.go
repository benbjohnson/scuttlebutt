package scuttlebutt

import (
	"testing"
	"time"

	"github.com/burntsushi/toml"
	"github.com/stretchr/testify/assert"
)

// Ensure that a configuration file can be decoded.
func TestConfig(t *testing.T) {
	str := `
app_key = "XXX"
app_secret = "000"
interval = "1m30s"

[[account]]
username = "github_js"
language = "javascript"
key = "ABC"
secret = "123"

[[account]]
username = "github_go"
language = "go"
key = "DEF"
secret = "456"
`
	c := &Config{}
	_, err := toml.Decode(str, &c)
	assert.NoError(t, err)
	assert.Equal(t, "XXX", c.AppKey)
	assert.Equal(t, "000", c.AppSecret)
	assert.Equal(t, 90*time.Second, c.Interval)
	if assert.Equal(t, 2, len(c.Accounts)) {
		assert.Equal(t, "github_js", c.Accounts[0].Username)
		assert.Equal(t, "javascript", c.Accounts[0].Language)
		assert.Equal(t, "ABC", c.Accounts[0].Key)
		assert.Equal(t, "123", c.Accounts[0].Secret)

		assert.Equal(t, "github_go", c.Accounts[1].Username)
		assert.Equal(t, "go", c.Accounts[1].Language)
		assert.Equal(t, "DEF", c.Accounts[1].Key)
		assert.Equal(t, "456", c.Accounts[1].Secret)
	}
}

// Ensure that a repository notify text is appropriately formatted.
func TestRepositoryNotifyText(t *testing.T) {
	r := &Repository{ID: "github.com/rails/rails", URL: "https://github.com/rails/rails", Description: "Ruby on Rails"}
	assert.Equal(t, "rails - Ruby on Rails https://github.com/rails/rails", r.NotifyText())
}

// Ensure that a repository notify text is appropriately formatted with a long description.
func TestRepositoryNotifyTextDescriptionTooLong(t *testing.T) {
	r := &Repository{ID: "github.com/browserstate/history.js", URL: "https://github.com/browserstate/history.js", Description: "History.js gracefully supports the HTML5 History/State APIs (pushState, replaceState, onPopState) in all browsers. Including continued support for data, titles, replaceState. Supports jQuery, MooTools and Prototype. "}
	assert.Equal(t, "history.js - History.js gracefully supports the HTML5 History/State APIs (pushState, replaceState, onPopState)... https://github.com/browserstate/history.js", r.NotifyText())
}
