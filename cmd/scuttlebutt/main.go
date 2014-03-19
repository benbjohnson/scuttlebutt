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

// DefaultSearchInterval is the default time between Twitter searches.
const DefaultSearchInterval = 5 * time.Second

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
	s := scuttlebutt.NewSearcher(db, key, secret)
	for {
		err := s.Search(func(repositoryID string, m *scuttlebutt.Message) {
			// TODO: Create repo if not exists.
			// TODO: Add message.
		})
		if err != nil {
			log.Print("[watch] ", err)
		}
		time.Sleep(DefaultSearchInterval)
	}
}

func notify(db *scuttlebutt.DB, accounts []*scuttlebutt.Account, interval time.Duration) {
	for {
		time.Sleep(time.Second)

		db.With(func(tx *scuttlebutt.Tx) error {
			// Retrieve list of accounts ready for notification.
			notifiable, err := tx.NotifiableAccounts(accounts, interval)
			if err != nil {
				log.Print("[notify] error: ", err)
				return nil
			} else if len(notifiable) == 0 {
				return nil
			}

			log.Print("[notify] Notifiable accounts: ", len(notifiable))

			// Retrieve top repositories.
			repositories, err := tx.TopRepositoriesByLanguage()
			if err != nil {
				log.Print("[notify] top repo error: ", err)
				return nil
			}

			// Notify each account that has an available repository.
			for _, account := range notifiable {
				r := repositories[account.Language]
				if r == nil {
					log.Print("[notify] No repo available: ", account.Username)
					continue
				}

				log.Print("[notify] Sending: ", account.Username, r.ID)

				if err := account.Notify(r); err != nil {
					log.Print("[notify] account notify error: ", err)
					continue
				}
				// TODO(benbjohnson): Update notify time.
				// TODO(benbjohnson): Update account status.
			}

			return nil
		})
	}
}
