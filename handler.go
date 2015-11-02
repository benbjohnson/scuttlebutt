package scuttlebutt

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Handler represents an HTTP interface to the store.
type Handler struct {
	Store *Store
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/debug/pprof") {
		switch r.URL.Path {
		case "/debug/pprof/cmdline":
			pprof.Cmdline(w, r)
		case "/debug/pprof/profile":
			pprof.Profile(w, r)
		case "/debug/pprof/symbol":
			pprof.Symbol(w, r)
		default:
			pprof.Index(w, r)
		}
		return
	}

	switch r.URL.Path {
	case "/":
		h.serveRoot(w, r)
	case "/top":
		h.serveTop(w, r)
	case "/top/stats":
		h.serveTopStats(w, r)
	case "/repositories":
		h.serveRepositories(w, r)
	case "/backup":
		h.serveBackup(w, r)
	case "/debug/vars":
		h.serveExpvars(w, r)
	default:
		http.NotFound(w, r)
	}
}

// serveRoot serves the home page.
func (h *Handler) serveRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, `<h1>scuttlebutt</h1>`)
	fmt.Fprintln(w, `<p><a href="/top">Top Repositories by Language</a></p>`)
	fmt.Fprintln(w, `<p><a href="/repositories">All Repositories</a></p>`)
}

// serveTop prints a list of the top repository for each language.
func (h *Handler) serveTop(w http.ResponseWriter, r *http.Request) {
	// Retrieve the top repositories.
	m, err := h.Store.TopRepositories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Sort keys.
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	w.Header().Set("content-type", "text/plain")

	// Print results.
	for _, k := range keys {
		r := m[k]
		fmt.Fprintf(w, "%s: %s - %s\n", k, r.Name(), r.Description)
	}
}

// serveTopStats prints timing stats for calculating top repos.
func (h *Handler) serveTopStats(w http.ResponseWriter, r *http.Request) {
	// Retrieve the top repositories.
	topStartTime := time.Now()
	_, err := h.Store.TopRepositories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	topDuration := time.Since(topStartTime)

	// Retrieve repository count.
	nStartTime := time.Now()
	repositoryN, err := h.Store.RepositoryN()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	nDuration := time.Since(nStartTime)

	w.Header().Set("content-type", "text/plain")
	fmt.Fprintf(w, "repositories: %d\n", repositoryN)
	fmt.Fprintf(w, "top time: %s (%s per repo)\n", topDuration, topDuration/time.Duration(repositoryN))
	fmt.Fprintf(w, "count time: %s\n", nDuration)
}

// serveRepositories prints a list of all repositories.
func (h *Handler) serveRepositories(w http.ResponseWriter, r *http.Request) {
	// Retrieve all repositories.
	repos, err := h.Store.Repositories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Sort by ID.
	sort.Sort(Repositories(repos))

	// Initialize CSV writer.
	w.Header().Set("Content-Type", "text/plain")
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"id", "description", "language", "notified", "messages"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write each row.
	for _, r := range repos {
		notified := strconv.FormatBool(r.Notified)
		messageN := strconv.Itoa(len(r.Messages))

		if err := cw.Write([]string{r.ID, r.Description, r.Language, notified, messageN}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Flush the writer out.
	cw.Flush()
}

// serveBackup writes the store to the response writer.
func (h *Handler) serveBackup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "binary/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=db")
	if _, err := h.Store.WriteTo(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// serveExpvars handles /debug/vars requests.
func (h *Handler) serveExpvars(w http.ResponseWriter, r *http.Request) {
	// Copied from $GOROOT/src/expvar/expvar.go

	// Generate JSON.
	data := expvarJSON()

	// Pretty print JSON.
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write to response.
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	buf.WriteTo(w)
}

// expvarJSON returns expvars as a JSON-formatted byte slice.
func expvarJSON() []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "{\n")
	first := true
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprintf(&buf, ",\n")
		}
		first = false
		fmt.Fprintf(&buf, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(&buf, "\n}\n")
	return buf.Bytes()
}
