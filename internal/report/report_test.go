package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sciamanamarcio/urlwatch/internal/checker"
)

var sample = []checker.Result{
	{URL: "https://a.test", Status: 200, OK: true, LatencyMS: 12},
	{URL: "https://b.test", Error: "timeout", LatencyMS: 5000},
}

func TestSummary(t *testing.T) {
	ok, failed := Summary(sample)

	if ok != 1 || failed != 1 {
		t.Fatalf("got ok=%d failed=%d, want 1 and 1", ok, failed)
	}
}

func TestWriteTable(t *testing.T) {
	var buf bytes.Buffer

	if err := WriteTable(&buf, sample); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	for _, want := range []string{"OK", "FAIL", "https://a.test", "(timeout)", "2 checked, 1 ok, 1 failed"} {
		if !strings.Contains(out, want) {
			t.Errorf("table output missing %q:\n%s", want, out)
		}
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer

	if err := WriteJSON(&buf, sample); err != nil {
		t.Fatal(err)
	}

	var decoded []checker.Result
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if len(decoded) != 2 || decoded[0].URL != "https://a.test" {
		t.Fatalf("unexpected decoded output: %+v", decoded)
	}
}
