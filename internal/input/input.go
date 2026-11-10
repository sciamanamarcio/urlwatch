// Package input reads URL lists from text sources.
package input

import (
"bufio"
"io"
"strings"
)

// ParseLines returns one entry per non-empty line, ignoring blank lines and
// lines starting with '#'.
func ParseLines(r io.Reader) ([]string, error) {
var lines []string
scanner := bufio.NewScanner(r)
for scanner.Scan() {
line := strings.TrimSpace(scanner.Text())
if line == "" || strings.HasPrefix(line, "#") {
continue
}
lines = append(lines, line)
}
return lines, scanner.Err()
}
