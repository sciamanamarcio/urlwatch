// Package checker probes URLs concurrently using a bounded worker pool.
package checker

import (
"context"
"errors"
"io"
"net/http"
"net/url"
"strings"
"sync"
"time"
)

// Result is the outcome of checking one URL.
type Result struct {
URL       string `json:"url"`
Status    int    `json:"status,omitempty"`
OK        bool   `json:"ok"`
LatencyMS int64  `json:"latency_ms"`
Error     string `json:"error,omitempty"`
}

// Options configures CheckAll.
type Options struct {
Concurrency int           // maximum number of requests in flight (default 10)
Timeout     time.Duration // per-request timeout (default 5s)
Client      *http.Client  // optional, mainly for tests
}

func (o Options) withDefaults() Options {
if o.Concurrency <= 0 {
o.Concurrency = 10
}
if o.Timeout <= 0 {
o.Timeout = 5 * time.Second
}
if o.Client == nil {
o.Client = &http.Client{}
}
return o
}

// Normalize adds https:// when the scheme is missing and validates the result.
func Normalize(raw string) (string, error) {
raw = strings.TrimSpace(raw)
if !strings.Contains(raw, "://") {
raw = "https://" + raw
}
u, err := url.Parse(raw)
if err != nil {
return "", err
}
if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
return "", errors.New("must be an http(s) URL with a host")
}
return u.String(), nil
}

// CheckAll checks every URL with at most Options.Concurrency requests in
// flight. Results keep the same order as the input.
func CheckAll(ctx context.Context, urls []string, opts Options) []Result {
opts = opts.withDefaults()
results := make([]Result, len(urls))

jobs := make(chan int)
var wg sync.WaitGroup

workers := min(opts.Concurrency, len(urls))
for w := 0; w < workers; w++ {
wg.Add(1)
go func() {
defer wg.Done()
for i := range jobs {
results[i] = checkOne(ctx, urls[i], opts)
}
}()
}

for i := range urls {
jobs <- i
}
close(jobs)
wg.Wait()

return results
}

func checkOne(ctx context.Context, raw string, opts Options) Result {
target, err := Normalize(raw)
if err != nil {
return Result{URL: raw, Error: err.Error()}
}

ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
defer cancel()

req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
if err != nil {
return Result{URL: target, Error: err.Error()}
}
req.Header.Set("User-Agent", "urlwatch/1.0")

start := time.Now()
resp, err := opts.Client.Do(req)
latency := time.Since(start).Milliseconds()
if err != nil {
return Result{URL: target, LatencyMS: latency, Error: simplifyError(err)}
}
defer resp.Body.Close()
_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))

return Result{
URL:       target,
Status:    resp.StatusCode,
OK:        resp.StatusCode < 400,
LatencyMS: latency,
}
}

func simplifyError(err error) string {
if ue, ok := err.(*url.Error); ok {
if ue.Timeout() {
return "timeout"
}
return ue.Err.Error()
}
return err.Error()
}
