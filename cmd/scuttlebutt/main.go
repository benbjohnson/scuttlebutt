package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/benbjohnson/scuttlebutt"
	"github.com/burntsushi/toml"
	// "github.com/kurrik/oauth1a"
	// "github.com/kurrik/twittergo"
)

var (
	dataDir    = flag.String("data-dir", "", "data directory")
	configPath = flag.String("config", "", "config path")
)

func main() {
	log.SetFlags(0)
	flag.Parse()
	if *dataDir == "" {
		log.Fatal("data directory required: -data-dir")
	} else if *configPath == "" {
		log.Fatal("config path required: -config")
	}

	// Read configuration.
	config := new(scuttlebutt.Config)
	if _, err := toml.DecodeFile(*configPath, &config); err != nil {
		log.Fatal("config error: " + err.Error())
	}

	// Ensure data directory exists.
	if err := os.MkdirAll(*dataDir, 0700); err != nil {
		log.Fatal("data dir error: " + err.Error())
	}

	// Open database.
	db := new(scuttlebutt.DB)
	if err := db.Open(filepath.Join(*dataDir, "db"), 0600); err != nil {
		log.Fatal("db error: " + err.Error())
	}

	// Start goroutines.
	go watch(db, config.AppKey, config.AppSecret)
	go notify(db, config.Accounts, time.Duration(config.Interval))
	select {}
}

func watch(db *scuttlebutt.DB, key string, secret string) {
	w := scuttlebutt.NewWatcher(key, secret)
	w.Watch(func(repositoryID string, m *scuttlebutt.Message) {
		// TODO: Create repo if not exists.
		// TODO: Add message.
	})
}

func notify(db *scuttlebutt.DB, accounts []*scuttlebutt.Account, interval time.Duration) {
	for {
		var pendingAccounts []*scuttlebutt.Account
		for _, account := range accounts {
			// If the last notification is less than the interval then skip this account.
			lastNotifyTime, err := db.LastNotifyTime(account.Username)
			if err != nil {
				log.Print("last notify time error: " + err.Error())
				continue
			} else if time.Now().Sub(lastNotifyTime) < interval {
				continue
			}

			// Otherwise add the account to the list of pending accounts.
			pendingAccounts = append(pendingAccounts, account)
		}

		// If we have no pending accounts then start over.
		if len(pendingAccounts) == 0 {
			continue
		}

		// Retrieve top repos.
		repositories, err := db.TopRepositoriesByLanguage()
		if err != nil {
			log.Print("top repositories error: ", err)
		}
	}
}
