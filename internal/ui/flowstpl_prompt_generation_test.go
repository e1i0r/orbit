package ui

// generatePhasePrompt's whole keyword table: one case per branch, whether it
// is reading the reader's own draft or falling back to the phase's name. The
// drafts are in either language and the instruction that comes back is in
// English, because it is read by an engine and not by the reader.

import (
	"strings"
	"testing"
)

func TestGeneratePhasePromptFromDraft(t *testing.T) {
	cases := []struct {
		name, input, wantSub string
	}{
		{"validate", "validate the new endpoint", "Validate what was implemented"},
		{"test keyword", "run the test suite", "Validate what was implemented"},
		{"security", "audit for security holes", "Audit the code for security holes"},
		{"refactor", "clean up this module", "Refactor for clarity"},
		{"fix", "fix the crash on startup", "Find the root cause"},
		{"docs", "write the readme", "technical documentation"},
		{"default fallback", "ship the release notes", "architecture and quality rules"},
	}
	for _, c := range cases {
		got := generatePhasePrompt(c.input, "some-phase", "some-flow")
		if !strings.Contains(got, c.wantSub) {
			t.Errorf("%s: generatePhasePrompt(%q) = %q, want it to contain %q", c.name, c.input, got, c.wantSub)
		}

		if !strings.Contains(got, c.input) {
			t.Errorf("%s: expected the draft %q to survive into the generated prompt %q", c.name, c.input, got)
		}
	}
}

func TestGeneratePhasePromptFromPhaseName(t *testing.T) {
	cases := []struct {
		phase, wantSub string
	}{
		{"1-plan", "design a technical plan"},
		{"design-review", "design a technical plan"},
		{"2-implement", "Implement the agreed plan"},
		{"build", "Implement the agreed plan"},
		{"3-test", "automated tests"},
		{"qa-gate", "automated tests"},
		{"4-review", "Audit the diff"},
		{"security-audit", "Audit the diff"},
		{"5-fix", "Fix the failures"},
		{"remediate", "Fix the failures"},
	}
	for _, c := range cases {
		got := generatePhasePrompt("", c.phase, "some-flow")
		if !strings.Contains(got, c.wantSub) {
			t.Errorf("phase %q: generatePhasePrompt = %q, want it to contain %q", c.phase, got, c.wantSub)
		}
	}
}

func TestGeneratePhasePromptDefaultFallback(t *testing.T) {
	withFlow := generatePhasePrompt("", "mystery-phase", "my-flow")
	if !strings.Contains(withFlow, "mystery-phase") || !strings.Contains(withFlow, "my-flow") {
		t.Errorf("expected both names in the fallback, got %q", withFlow)
	}

	withoutFlow := generatePhasePrompt("", "mystery-phase", "")
	if !strings.Contains(withoutFlow, "mystery-phase") {
		t.Errorf("expected the phase name in the fallback, got %q", withoutFlow)
	}

	if strings.Contains(withoutFlow, "flow") {
		t.Errorf("expected no flow clause when flowName is empty, got %q", withoutFlow)
	}
}
