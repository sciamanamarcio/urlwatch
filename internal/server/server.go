// Package server exposes the checker over HTTP.
package server

import (
	"encoding/json"
	"net/http"

	"github.com/sciamanamarcio/urlwatch/internal/checker"
)

const (
	maxURLsPerRequest = 50
	maxBodyBytes      = 1 << 20
)

type checkRequest struct {
	URLs []string `json:"urls"`
}

// New returns the HTTP handler. opts is shared by every request.
func New(opts checker.Options) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /check", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

		var req checkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if len(req.URLs) == 0 {
			writeError(w, http.StatusBadRequest, "urls must not be empty")
			return
		}
		if len(req.URLs) > maxURLsPerRequest {
			writeError(w, http.StatusBadRequest, "too many urls (max 50)")
			return
		}

		results := checker.CheckAll(r.Context(), req.URLs, opts)
		writeJSON(w, http.StatusOK, results)
	})

	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
