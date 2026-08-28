package ui

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

func TestExtractFileRationales(t *testing.T) {
	files := []diffFile{
		{Path: "internal/ui/badge.go", Status: "NEW"},
		{Path: "internal/ui/badge_test.go", Status: "NEW"},
	}

	entries := []view.Entry{
		{
			Kind: "phase.finished",
			Text: `• - internal/ui/badge.go (20 lines) — FormatLatency(ms int64) string, colors via Paint(Role) convention
• - internal/ui/badge_test.go (45 lines) — table-driven threshold/boundary tests plus a purity check`,
		},
	}

	p := words.For("en")

	rationales := extractFileRationales(entries, files, p)
	if len(rationales) != 2 {
		t.Fatalf("expected 2 rationales, got %d", len(rationales))
	}

	r1 := rationales["internal/ui/badge.go"]
	if r1 == "" || !strings.Contains(r1, "FormatLatency") {
		t.Errorf("badge.go rationale = %q, want FormatLatency explanation", r1)
	}

	r2 := rationales["internal/ui/badge_test.go"]
	if r2 == "" || !strings.Contains(r2, "table-driven") {
		t.Errorf("badge_test.go rationale = %q, want table-driven explanation", r2)
	}
}
