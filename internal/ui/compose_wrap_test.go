package ui

import (
	"strings"
	"testing"
)

func TestSplitIntoLines(t *testing.T) {
	text := "Create a helper function in internal/ui/badge.go with pure layout FormatLatency"
	lines := splitIntoLines(text, 30)

	if len(lines) < 2 {
		t.Fatalf("expected multiple lines, got %d", len(lines))
	}

	for i, l := range lines {
		if len(l) > 30 {
			t.Errorf("line %d exceeds max width 30: %q (len %d)", i, l, len(l))
		}
	}

	rejoined := strings.Join(lines, " ")
	if rejoined != text {
		t.Errorf("rejoined text mismatch: got %q, want %q", rejoined, text)
	}
}

func TestSplitIntoLinesEmptyAndZero(t *testing.T) {
	if got := splitIntoLines("", 30); len(got) != 1 || got[0] != "" {
		t.Errorf("splitIntoLines empty string = %v, want ['']", got)
	}

	if got := splitIntoLines("hello", 0); len(got) != 1 || got[0] != "hello" {
		t.Errorf("splitIntoLines zero width = %v, want ['hello']", got)
	}
}
