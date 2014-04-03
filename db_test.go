package scuttlebutt

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Ensure that repositories and messages can be added and retrieved.
func TestDB(t *testing.T) {
	withDB(func(db *DB) {
		db.Update(func(tx *Tx) error {
			assert.NoError(t, tx.PutRepository(&Repository{"//github.com/foo/bar", "https://github.com/foo/bar", "", "go", nil}))
			assert.NoError(t, tx.PutRepository(&Repository{"//github.com/rails/rails", "https://github.com/rails/rails", "", "ruby", nil}))

			assert.NoError(t, tx.AddMessage("//github.com/rails/rails", &Message{123, "XXX"}))
			assert.NoError(t, tx.AddMessage("//github.com/foo/bar", &Message{456, "YYY"}))
			assert.NoError(t, tx.AddMessage("//github.com/rails/rails", &Message{789, "ZZZ"}))
			return nil
		})

		var index int
		db.View(func(tx *Tx) error {
			return tx.ForEachRepository(func(r *Repository) error {
				switch index {
				case 0:
					assert.Equal(t, "//github.com/foo/bar", r.ID)
					assert.Equal(t, "go", r.Language)
					if assert.Equal(t, 1, len(r.Messages)) {
						assert.Equal(t, 456, r.Messages[0].ID)
						assert.Equal(t, "YYY", r.Messages[0].Text)
					}
				case 1:
					assert.Equal(t, "//github.com/rails/rails", r.ID)
					assert.Equal(t, "ruby", r.Language)
					if assert.Equal(t, 2, len(r.Messages)) {
						assert.Equal(t, 123, r.Messages[0].ID)
						assert.Equal(t, "XXX", r.Messages[0].Text)
						assert.Equal(t, 789, r.Messages[1].ID)
						assert.Equal(t, "ZZZ", r.Messages[1].Text)
					}
				default:
					panic("invalid index")
				}
				index++
				return nil
			})
		})
		assert.Equal(t, 2, index)
	})
}

// withDB executes a function with an open database reference.
func withDB(fn func(*DB)) {
	f, _ := ioutil.TempFile("", "scuttlebutt-")
	path := f.Name()
	f.Close()
	os.Remove(path)
	defer os.RemoveAll(path)

	var db DB
	if err := db.Open(path, 0644); err != nil {
		panic("cannot open db: " + err.Error())
	}
	defer db.Close()
	fn(&db)
}
