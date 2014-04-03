package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/benbjohnson/scuttlebutt"
	"github.com/burntsushi/toml"
)

// DefaultSearchInterval is the default time between Twitter searches.
const DefaultSearchInterval = 30 * time.Second

var (
	dataDir    = flag.String("data-dir", "", "data directory")
	configPath = flag.String("config", "", "config path")
	addr       = flag.String("addr", ":5050", "HTTP port")
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
	go notify(db, config.AppKey, config.AppSecret, config.Accounts, time.Duration(config.Interval))

	// Start HTTP server.
	h := &scuttlebutt.Handler{db}
	http.HandleFunc("/top", h.TopHandleFunc)
	http.HandleFunc("/repositories", h.RepositoriesHandleFunc)
	log.Printf("Listening on http://localhost%s", *addr)
	log.SetFlags(log.LstdFlags)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func watch(db *scuttlebutt.DB, key, secret string) {
	logger := log.New(os.Stdout, "[watch] ", log.LstdFlags)
	s := scuttlebutt.NewSearcher(key, secret)
	for {
		err := db.Update(func(tx *scuttlebutt.Tx) error {
			sinceID, _ := strconv.Atoi(tx.Meta("LastTweetID"))
			logger.Println(s.SearchURL(sinceID).String())
			results, err := s.Search(sinceID)
			if err != nil {
				return err
			}
			logger.Printf("rate limit: %v / %v / %v\n", results.RateLimit, results.RateLimitRemaining, results.RateLimitReset)

			// Process each result.
			for _, result := range results.Results {
				logger.Printf("https://twitter.com/_/status/%d", result.ID)

				// Update the last tweet id.
				if result.ID > sinceID {
					sinceID = result.ID
				}

				// Find relevant repository.
				var repositoryID string
				for _, u := range result.URLs {
					repositoryID, err = scuttlebutt.ExtractRepositoryID(u)
					if err != nil {
						u.Scheme = ""
						u.RawQuery = ""
						break
					}
				}
				if repositoryID == "" {
					continue
				}

				// Create message from result.
				m := &scuttlebutt.Message{ID: result.ID, Text: result.Text}

				// Find or create the repository and add the message.
				r, err := tx.FindOrCreateRepository(repositoryID)
				if err != nil {
					if !strings.Contains(err.Error(), "404 Not Found") {
						logger.Println("find or create repo error:", err)
					}
					continue
				}

				// Add message to repo.
				r.Messages = append(r.Messages, m)

				// Update repository.
				if err := tx.PutRepository(r); err != nil {
					logger.Println("update repo error:", err)
					continue
				}

				logger.Printf("OK: %s %s (%d)", r.Language, r.ID, len(r.Messages))
			}

			// Update highwater mark.
			if err := tx.SetMeta("LastTweetID", strconv.Itoa(sinceID)); err != nil {
				return fmt.Errorf("set last tweet id error: %s", err)
			}

			return nil
		})
		if err != nil {
			logger.Println(err)
		}
		logger.Println(strings.Repeat("=", 40))
		time.Sleep(DefaultSearchInterval)
	}
}

func notify(db *scuttlebutt.DB, key, secret string, accounts []*scuttlebutt.Account, interval time.Duration) {
	logger := log.New(os.Stdout, "[notify] ", log.LstdFlags)
	for {
		time.Sleep(time.Second)

		db.Update(func(tx *scuttlebutt.Tx) error {
			// Retrieve list of accounts ready for notification.
			notifiable, err := tx.NotifiableAccounts(accounts, interval)
			if err != nil {
				logger.Print("error: ", err)
				return nil
			} else if len(notifiable) == 0 {
				return nil
			}

			logger.Print("notifiable accounts: ", len(notifiable))

			// Retrieve top repositories.
			repositories, err := tx.TopRepositoriesByLanguage()
			if err != nil {
				logger.Print("top repo error: ", err)
				return nil
			}

			// Notify each account that has an available repository.
			for _, account := range notifiable {
				r := repositories[account.Language]
				if r == nil {
					logger.Print("No repo available: ", account.Username)
					continue
				}

				logger.Println("sending: ", account.Username, r.ID)
				tweetID, err := account.Notify(account.Client(key, secret), r)
				if err != nil {
					logger.Print("account notify: ", err)
					continue
				}
				logger.Printf("sent: https://twitter.com/_/status/%d", tweetID)

				// Update notify status.
				if err := tx.SetAccountStatus(account.Username, &scuttlebutt.AccountStatus{time.Now()}); err != nil {
					logger.Print("account status: ", err)
					continue
				}

				// Add repository to the blacklist.
				if err := tx.AddToBlacklist(r.ID); err != nil {
					logger.Print("add to blacklist: ", err)
					continue
				}
			}

			return nil
		})
	}
}

func marshalJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
