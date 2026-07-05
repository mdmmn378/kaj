package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseImportLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		max   int
		want  []string
	}{
		{
			name:  "multi-line trimmed",
			input: "buy milk\nwalk dog\n  clean desk  ",
			max:   100,
			want:  []string{"buy milk", "walk dog", "clean desk"},
		},
		{
			name:  "blank lines dropped",
			input: "one\n\n\ntwo\n   \nthree",
			max:   100,
			want:  []string{"one", "two", "three"},
		},
		{
			name:  "crlf line endings",
			input: "alpha\r\nbeta\r\ngamma",
			max:   100,
			want:  []string{"alpha", "beta", "gamma"},
		},
		{
			name:  "empty input",
			input: "",
			max:   100,
			want:  nil,
		},
		{
			name:  "whitespace only",
			input: "   \n\t\n  ",
			max:   100,
			want:  nil,
		},
		{
			name:  "single line no newline",
			input: "just one",
			max:   100,
			want:  []string{"just one"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseImportLines(tt.input, tt.max)
			if !equalSlices(got, tt.want) {
				t.Errorf("parseImportLines(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseImportLinesCap(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 250; i++ {
		fmt.Fprintf(&b, "task %d\n", i)
	}

	got := parseImportLines(b.String(), 100)
	if len(got) != 100 {
		t.Fatalf("expected 100 lines after cap, got %d", len(got))
	}
	if got[0] != "task 0" || got[99] != "task 99" {
		t.Errorf("expected first 100 tasks in order, got first=%q last=%q", got[0], got[99])
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
