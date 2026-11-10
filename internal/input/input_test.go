package input

import (
"reflect"
"strings"
"testing"
)

func TestParseLinesSkipsBlanksAndComments(t *testing.T) {
src := "# my sites\nexample.com\n\n  https://go.dev  \n# disabled\n"

got, err := ParseLines(strings.NewReader(src))
if err != nil {
t.Fatal(err)
}

want := []string{"example.com", "https://go.dev"}
if !reflect.DeepEqual(got, want) {
t.Fatalf("got %v, want %v", got, want)
}
}
