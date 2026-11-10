# urlwatch

![CI](https://github.com/sciamanamarcio/urlwatch/actions/workflows/ci.yml/badge.svg)

A fast URL health checker written in **Go**. It probes many URLs concurrently with a bounded worker pool, and works both as a **CLI** and as a small **HTTP service**. It uses only the Go standard library.

## Features

- Concurrent checks with a configurable limit on requests in flight
- Per-request timeout and cancellation through `context`
- Results always keep the same order as the input
- Table or JSON output, with exit codes that are easy to use in scripts and CI
- Reads URLs from arguments, a file or stdin; blank lines and `#` comments are ignored
- `serve` mode exposing the same checker over HTTP, with graceful shutdown
- Tested with `httptest`, including a test that proves the concurrency limit is respected
- Multi-stage Docker image (static binary, non-root user) and GitHub Actions CI with the race detector

## Install and run

Requires Go 1.22 or later.

```bash
git clone https://github.com/sciamanamarcio/urlwatch.git
cd urlwatch
make build
./bin/urlwatch github.com go.dev example.com
```

## CLI

```
urlwatch [flags] [url ...]
urlwatch serve [flags]
```

Flags must come before the URLs.

| Flag | Default | Description |
|------|---------|-------------|
| `-f` | | File with one URL per line (`-` reads stdin) |
| `-c` | `10` | Maximum concurrent requests |
| `-t` | `5s` | Timeout per request |
| `-json` | `false` | Print results as JSON |

URLs without a scheme are checked over `https://`. A URL is healthy when the final status code (after redirects) is below 400.

```bash
$ urlwatch -t 3s github.com go.dev does-not-exist.invalid
STATUS  CODE  LATENCY  URL
OK      200   182ms    https://github.com
OK      200   95ms     https://go.dev
FAIL    -     41ms     https://does-not-exist.invalid  (no such host)

3 checked, 2 ok, 1 failed
```

| Exit code | Meaning |
|-----------|---------|
| `0` | All URLs are healthy |
| `1` | At least one URL failed |
| `2` | Usage or runtime error |

```bash
# in a script or pipeline
urlwatch -f sites.txt || echo "something is down"
```

## HTTP service

```bash
urlwatch serve -addr :8080
```

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/healthz` | Health check |
| POST | `/check` | Check up to 50 URLs, returns the results as JSON |

```bash
curl -X POST localhost:8080/check -d '{"urls":["github.com","go.dev"]}'
```

## Docker

```bash
docker compose up --build
# service on http://localhost:8080

# one-off CLI run with the same image
docker compose run --rm --entrypoint urlwatch urlwatch github.com go.dev
```

## Development

```bash
make lint   # go vet and gofmt check
make test   # tests with the race detector
```

## Architecture

```
cmd/urlwatch/      # CLI entry point and serve command
internal/checker/  # concurrent worker pool and URL normalization
internal/input/    # URL list parsing
internal/report/   # table and JSON output
internal/server/   # HTTP handlers
```

## Design decisions

- **Bounded worker pool** instead of one goroutine per URL, so a long list never opens thousands of connections at once.
- **Results written by index**, so output order is deterministic without locks or sorting.
- **`context` per request** for timeouts, and a signal-aware context so `Ctrl+C` stops a run cleanly.
- **`run()` separated from `main()`**, which makes the CLI testable end to end, exit codes included.
- **Standard library only**: no dependencies to audit or update.
- **Static binary on Alpine as a non-root user**, with CA certificates included for HTTPS checks.

## Roadmap

- [ ] Retries with backoff for flaky endpoints
- [ ] Prometheus metrics endpoint
- [ ] Check TLS certificate expiry
- [ ] Release binaries for Linux, macOS and Windows

## License

MIT
