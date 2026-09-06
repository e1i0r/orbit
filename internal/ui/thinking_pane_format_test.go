package ui

import (
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/ui/theme"
	"github.com/e1i0r/orbit/internal/view"
)

func TestFormatThoughtLineAndThinkingLines(t *testing.T) {
	// 1. formatThoughtLine categorizations
	tests := []struct {
		input string
		want  string
		role  theme.Role
	}{
		{"decided to refactor", "🎯 decided to refactor", theme.OK},
		{"rejected outdated plan", "🚫 rejected outdated plan", theme.Warn},
		{"investigating the cache issue", "🔍 investigating the cache issue", theme.Live},
		{"because the disk was full", "💡 because the disk was full", theme.Accent},
		{"plain thinking text", "• plain thinking text", theme.Dim},
	}

	for _, tt := range tests {
		got, role := formatThoughtLine(tt.input)
		if got != tt.want || role != tt.role {
			t.Errorf("formatThoughtLine(%q) = (%q, %v), want (%q, %v)",
				tt.input, got, role, tt.want, tt.role)
		}
	}

	// 2. thinkingLines rendering
	m, _ := testModel(t, 120, 30)
	m.entries = []view.Entry{
		{
			Kind:    "phase.thought",
			Phase:   "plan",
			Attempt: 1,
			At:      time.Now(),
			Text:    "decided to implement caching\nreject memory bloat\ninvestigate patterns\nbecause it is faster",
		},
	}

	lines := m.thinkingLines()
	if len(lines) == 0 {
		t.Error("expected thinkingLines to render output blocks")
	}
}
