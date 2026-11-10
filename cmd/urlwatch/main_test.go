package main

import (
"bytes"
"encoding/json"
"net/http"
"net/http/httptest"
"strings"
"testing"

"github.com/sciamanamarcio/urlwatch/internal/checker"
)

func targets(t *testing.T) (okURL, failURL string) {
t.Helper()
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
if r.URL.Path == "/fail" {
w.WriteHeader(http.StatusInternalServerError)
return
}
w.WriteHeader(http.StatusOK)
}))
t.Cleanup(srv.Close)
return srv.URL + "/ok", srv.URL + "/fail"
}

func exec(t *testing.T, stdin string, args ...string) (code int, stdout, stderr string) {
t.Helper()
var out, errOut bytes.Buffer
code = run(args, strings.NewReader(stdin), &out, &errOut)
return code, out.String(), errOut.String()
}

func TestExitCodeZeroWhenAllHealthy(t *testing.T) {
okURL, _ := targets(t)

code, out, _ := exec(t, "", okURL)

if code != 0 {
t.Fatalf("exit code = %d, want 0\n%s", code, out)
}
if !strings.Contains(out, "1 checked, 1 ok, 0 failed") {
t.Fatalf("unexpected output:\n%s", out)
}
}

func TestExitCodeOneWhenAnyFails(t *testing.T) {
okURL, failURL := targets(t)

code, out, _ := exec(t, "", okURL, failURL)

if code != 1 {
t.Fatalf("exit code = %d, want 1\n%s", code, out)
}
}

func TestJSONOutput(t *testing.T) {
okURL, _ := targets(t)

code, out, _ := exec(t, "", "-json", okURL)

var results []checker.Result
if err := json.Unmarshal([]byte(out), &results); err != nil {
t.Fatalf("invalid JSON: %v\n%s", err, out)
}
if code != 0 || len(results) != 1 || results[0].Status != 200 {
t.Fatalf("unexpected: code=%d results=%+v", code, results)
}
}

func TestReadsURLsFromStdin(t *testing.T) {
okURL, failURL := targets(t)

code, out, _ := exec(t, "# list\n"+okURL+"\n\n"+failURL+"\n", "-f", "-")

if code != 1 || !strings.Contains(out, "2 checked, 1 ok, 1 failed") {
t.Fatalf("unexpected: code=%d\n%s", code, out)
}
}

func TestUsageErrorsReturnTwo(t *testing.T) {
if code, _, _ := exec(t, ""); code != 2 {
t.Errorf("no URLs: exit code = %d, want 2", code)
}
if code, _, _ := exec(t, "", "-f", "/does/not/exist"); code != 2 {
t.Errorf("missing file: exit code = %d, want 2", code)
}
if code, _, _ := exec(t, "", "-nope"); code != 2 {
t.Errorf("unknown flag: exit code = %d, want 2", code)
}
}
