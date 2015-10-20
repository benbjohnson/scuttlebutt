package main_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	main "github.com/benbjohnson/scuttlebutt/cmd/scuttlebuttd"
	"github.com/burntsushi/toml"
	"github.com/davecgh/go-spew/spew"
)

// Ensure that a configuration file can be decoded.
func TestConfig(t *testing.T) {
	str := `
[twitter]
key = "XXX"
secret = "YYY"

[github]
token = "ZZZ"

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
	c := &main.Config{}
	_, err := toml.Decode(str, &c)
	if err != nil {
		t.Fatal(err)
	}

	// Verify top-level properties.
	if c.Twitter.Key != "XXX" {
		t.Fatalf("unexpected twitter key: %s", c.Twitter.Key)
	} else if c.Twitter.Secret != "YYY" {
		t.Fatalf("unexpected twitter secret: %s", c.Twitter.Secret)
	} else if c.GitHub.Token != "ZZZ" {
		t.Fatalf("unexpected github token: %s", c.GitHub.Token)
	} else if len(c.Accounts) != 2 {
		t.Fatalf("unexpected account count: %d", len(c.Accounts))
	}

	// Verify first account.
	if !reflect.DeepEqual(c.Accounts[0], &main.Account{
		Username: "github_js",
		Language: "javascript",
		Key:      "ABC",
		Secret:   "123",
	}) {
		t.Fatalf("unexpected account(0): %s", spew.Sdump(c.Accounts[0]))
	}

	// Verify second account.
	if !reflect.DeepEqual(c.Accounts[1], &main.Account{
		Username: "github_go",
		Language: "go",
		Key:      "DEF",
		Secret:   "456",
	}) {
		t.Fatalf("unexpected account(1): %s", spew.Sdump(c.Accounts[1]))
	}
}

// Ensure the program can parse command line flags.
func TestMain_ParseFlags(t *testing.T) {
	// Create temporary path for config.
	f, _ := ioutil.TempFile("", "scuttlebuttd-")
	f.Close()
	defer os.Remove(f.Name())

	// Write config file.
	if err := ioutil.WriteFile(f.Name(), []byte(`
[twitter]
key = "XXX"
secret = "YYY"

[github]
token = "ZZZ"

[[account]]
username = "github_js"
`), 0666); err != nil {
		t.Fatal(err)
	}

	// Parse flags and config.
	m := NewMain()
	err := m.ParseFlags([]string{"-d", "/my/data", "-c", f.Name(), "-addr", ":1000"})
	if err != nil {
		t.Fatal(err)
	} else if m.DataDir != "/my/data" {
		t.Fatalf("unexpected path: %s", m.DataDir)
	} else if m.ConfigPath != f.Name() {
		t.Fatalf("unexpected config path: %s", m.ConfigPath)
	} else if m.Config.Twitter.Key != "XXX" {
		t.Fatalf("unexpected twitter key: %s", m.Config.Twitter.Key)
	}
}

type Main struct {
	*main.Main

	Stdin  bytes.Buffer
	Stdout bytes.Buffer
	Stderr bytes.Buffer
}

func NewMain() *Main {
	m := &Main{Main: main.NewMain()}
	m.Main.Stdin = &m.Stdin
	m.Main.Stdout = &m.Stderr
	m.Main.Stderr = &m.Stderr

	if testing.Verbose() {
		m.Main.Stdout = io.MultiWriter(os.Stderr, m.Main.Stdout)
		m.Main.Stderr = io.MultiWriter(os.Stderr, m.Main.Stderr)
	}

	return m
}
