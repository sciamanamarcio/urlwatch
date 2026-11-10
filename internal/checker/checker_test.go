package checker

import (
"context"
"net/http"
"net/http/httptest"
"sync/atomic"
"testing"
"time"
)

func newServer(t *testing.T) *httptest.Server {
t.Helper()
mux := http.NewServeMux()
mux.HandleFunc("/ok", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
mux.HandleFunc("/missing", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })
mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
select {
case <-time.After(2 * time.Second):
case <-r.Context().Done():
}
})
mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
http.Redirect(w, r, "/ok", http.StatusFound)
})
srv := httptest.NewServer(mux)
t.Cleanup(srv.Close)
return srv
}

func TestCheckAllClassifiesResultsAndKeepsOrder(t *testing.T) {
srv := newServer(t)
urls := []string{srv.URL + "/ok", srv.URL + "/missing", srv.URL + "/redirect"}

results := CheckAll(context.Background(), urls, Options{Concurrency: 2})

if len(results) != 3 {
t.Fatalf("expected 3 results, got %d", len(results))
}
want := []struct {
status int
ok     bool
}{{200, true}, {404, false}, {200, true}}
for i, w := range want {
if results[i].URL != urls[i] {
t.Errorf("result %d: url = %s, want %s", i, results[i].URL, urls[i])
}
if results[i].Status != w.status || results[i].OK != w.ok {
t.Errorf("result %d: got status=%d ok=%v, want status=%d ok=%v",
i, results[i].Status, results[i].OK, w.status, w.ok)
}
}
}

func TestCheckAllReportsTimeout(t *testing.T) {
srv := newServer(t)

results := CheckAll(context.Background(), []string{srv.URL + "/slow"},
Options{Timeout: 100 * time.Millisecond})

if results[0].OK || results[0].Error != "timeout" {
t.Fatalf("expected a timeout error, got %+v", results[0])
}
}

func TestCheckAllReportsInvalidURL(t *testing.T) {
results := CheckAll(context.Background(), []string{"ftp://example.com"}, Options{})

if results[0].OK || results[0].Error == "" {
t.Fatalf("expected an error for an invalid URL, got %+v", results[0])
}
}

func TestCheckAllRespectsConcurrencyLimit(t *testing.T) {
var inFlight, peak atomic.Int32
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
n := inFlight.Add(1)
for {
p := peak.Load()
if n <= p || peak.CompareAndSwap(p, n) {
break
}
}
time.Sleep(50 * time.Millisecond)
inFlight.Add(-1)
w.WriteHeader(http.StatusOK)
}))
defer srv.Close()

urls := make([]string, 12)
for i := range urls {
urls[i] = srv.URL
}

CheckAll(context.Background(), urls, Options{Concurrency: 3})

if peak.Load() > 3 {
t.Fatalf("peak concurrency %d exceeded the limit of 3", peak.Load())
}
if peak.Load() < 2 {
t.Fatalf("expected requests to run in parallel, peak was %d", peak.Load())
}
}

func TestCheckAllHandlesEmptyInput(t *testing.T) {
if got := CheckAll(context.Background(), nil, Options{}); len(got) != 0 {
t.Fatalf("expected no results, got %d", len(got))
}
}

func TestNormalize(t *testing.T) {
cases := []struct {
in      string
want    string
wantErr bool
}{
{"example.com", "https://example.com", false},
{"http://example.com/path", "http://example.com/path", false},
{"  https://example.com  ", "https://example.com", false},
{"ftp://example.com", "", true},
{"https://", "", true},
}
for _, c := range cases {
got, err := Normalize(c.in)
if (err != nil) != c.wantErr || got != c.want {
t.Errorf("Normalize(%q) = %q, %v; want %q, err=%v", c.in, got, err, c.want, c.wantErr)
}
}
}
