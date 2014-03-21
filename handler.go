package scuttlebutt

import (
	"encoding/json"
	"net/http"
)

// Handler represents the primary HTTP handler.
type Handler struct {
	DB *DB
}

// RepositoriesHandleFunc writes out all repositories data as JSON.
func (h *Handler) RepositoriesHandleFunc(w http.ResponseWriter, r *http.Request) {
	h.DB.With(func(tx *Tx) error {
		return tx.Bucket("repositories").ForEach(func(k, v []byte) error {
			w.Write(v)
			w.Write([]byte("\n"))
			return nil
		})
	})
}

// TopHandleFunc writes out all repositories data as JSON.
func (h *Handler) TopHandleFunc(w http.ResponseWriter, r *http.Request) {
	h.DB.With(func(tx *Tx) error {
		m, err := tx.TopRepositoriesByLanguage()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return nil
		}
		json.NewEncoder(w).Encode(m)
		return nil
	})
}
