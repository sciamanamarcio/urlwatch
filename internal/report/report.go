// Package report renders check results for humans and machines.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/sciamanamarcio/urlwatch/internal/checker"
)

// Summary counts how many results succeeded and failed.
func Summary(results []checker.Result) (ok, failed int) {
	for _, r := range results {
		if r.OK {
			ok++
		} else {
			failed++
		}
	}
	return ok, failed
}

// WriteTable prints an aligned table followed by a summary line.
func WriteTable(w io.Writer, results []checker.Result) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "STATUS\tCODE\tLATENCY\tURL")
	for _, r := range results {
		label, code := "OK", fmt.Sprint(r.Status)
		if !r.OK {
			label = "FAIL"
		}
		if r.Status == 0 {
			code = "-"
		}
		detail := r.URL
		if r.Error != "" {
			detail += "  (" + r.Error + ")"
		}
		fmt.Fprintf(tw, "%s\t%s\t%dms\t%s\n", label, code, r.LatencyMS, detail)
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	ok, failed := Summary(results)
	_, err := fmt.Fprintf(w, "\n%d checked, %d ok, %d failed\n", len(results), ok, failed)
	return err
}

// WriteJSON prints the results as indented JSON.
func WriteJSON(w io.Writer, results []checker.Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}
