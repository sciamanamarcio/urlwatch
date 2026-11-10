package server

import (
"encoding/json"
"net/http"
"net/http/httptest"
"strings"
"testing"

"github.com/sciamanamarcio/urlwatch/internal/checker"
)

func TestHealthz(t *testing.T) {
rec := httptest.NewRecorder()

New(checker.Options{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

if rec.Code != http.StatusOK {
t.Fatalf("status = %d, want 200", rec.Code)
}
}

func TestCheckReturnsResults(t *testing.T) {
target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
w.WriteHeader(http.StatusOK)
}))
defer target.Close()

body := `{"urls":["` + target.URL + `"]}`
rec := httptest.NewRecorder()
req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))

New(checker.Options{}).ServeHTTP(rec, req)

if rec.Code != http.StatusOK {
t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
}
var results []checker.Result
if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
t.Fatal(err)
}
if len(results) != 1 || !results[0].OK {
t.Fatalf("unexpected results: %+v", results)
}
}

func TestCheckRejectsBadRequests(t *testing.T) {
manyURLs := `{"urls":["` + strings.Repeat(`a.test","`, 60) + `a.test"]}`
cases := map[string]string{
"invalid json": `not json`,
"empty list":   `{"urls":[]}`,
"too many":     manyURLs,
}

for name, body := range cases {
t.Run(name, func(t *testing.T) {
rec := httptest.NewRecorder()
req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))

New(checker.Options{}).ServeHTTP(rec, req)

if rec.Code != http.StatusBadRequest {
t.Fatalf("status = %d, want 400", rec.Code)
}
})
}
}
