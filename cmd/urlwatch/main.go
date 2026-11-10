// Command urlwatch checks the health of many URLs concurrently.
//
// Usage:
//
//urlwatch [flags] [url ...]
//urlwatch serve [flags]
package main

import (
"context"
"errors"
"flag"
"fmt"
"io"
"net/http"
"os"
"os/signal"
"syscall"
"time"

"github.com/sciamanamarcio/urlwatch/internal/checker"
"github.com/sciamanamarcio/urlwatch/internal/input"
"github.com/sciamanamarcio/urlwatch/internal/report"
"github.com/sciamanamarcio/urlwatch/internal/server"
)

func main() {
os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run is separate from main so it can be tested. It returns the exit code:
// 0 all URLs healthy, 1 at least one failed, 2 usage or runtime error.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
if len(args) > 0 && args[0] == "serve" {
return runServe(args[1:], stderr)
}
return runCheck(args, stdin, stdout, stderr)
}

func runCheck(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
fs := flag.NewFlagSet("urlwatch", flag.ContinueOnError)
fs.SetOutput(stderr)
file := fs.String("f", "", "file with one URL per line ('-' reads stdin)")
concurrency := fs.Int("c", 10, "maximum concurrent requests")
timeout := fs.Duration("t", 5*time.Second, "timeout per request")
asJSON := fs.Bool("json", false, "print results as JSON")
fs.Usage = func() {
fmt.Fprintln(stderr, "Usage: urlwatch [flags] [url ...]")
fmt.Fprintln(stderr, "       urlwatch serve [flags]")
fmt.Fprintln(stderr, "\nFlags (must come before the URLs):")
fs.PrintDefaults()
}

if err := fs.Parse(args); err != nil {
if errors.Is(err, flag.ErrHelp) {
return 0
}
return 2
}

urls := fs.Args()
if *file != "" {
fromFile, err := readURLs(*file, stdin)
if err != nil {
fmt.Fprintln(stderr, "error:", err)
return 2
}
urls = append(urls, fromFile...)
}
if len(urls) == 0 {
fs.Usage()
return 2
}

ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

results := checker.CheckAll(ctx, urls, checker.Options{Concurrency: *concurrency, Timeout: *timeout})

var err error
if *asJSON {
err = report.WriteJSON(stdout, results)
} else {
err = report.WriteTable(stdout, results)
}
if err != nil {
fmt.Fprintln(stderr, "error:", err)
return 2
}

if _, failed := report.Summary(results); failed > 0 {
return 1
}
return 0
}

func readURLs(path string, stdin io.Reader) ([]string, error) {
if path == "-" {
return input.ParseLines(stdin)
}
f, err := os.Open(path)
if err != nil {
return nil, err
}
defer f.Close()
return input.ParseLines(f)
}

func runServe(args []string, stderr io.Writer) int {
fs := flag.NewFlagSet("urlwatch serve", flag.ContinueOnError)
fs.SetOutput(stderr)
addr := fs.String("addr", ":8080", "address to listen on")
concurrency := fs.Int("c", 10, "maximum concurrent requests per call")
timeout := fs.Duration("t", 5*time.Second, "timeout per checked URL")
if err := fs.Parse(args); err != nil {
if errors.Is(err, flag.ErrHelp) {
return 0
}
return 2
}

srv := &http.Server{
Addr:              *addr,
Handler:           server.New(checker.Options{Concurrency: *concurrency, Timeout: *timeout}),
ReadHeaderTimeout: 5 * time.Second,
WriteTimeout:      60 * time.Second,
}

ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

errCh := make(chan error, 1)
go func() { errCh <- srv.ListenAndServe() }()
fmt.Fprintf(stderr, "urlwatch listening on %s\n", *addr)

select {
case err := <-errCh:
fmt.Fprintln(stderr, "error:", err)
return 2
case <-ctx.Done():
}

shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
if err := srv.Shutdown(shutdownCtx); err != nil {
fmt.Fprintln(stderr, "shutdown error:", err)
return 2
}
return 0
}
