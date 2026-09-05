package supervisor

// How the supervisor is asked to answer, which is not how a phase is.

import (
	"strings"
	"testing"
)

// TestTheSupervisorIsAskedForAnAnswerAndNotAReport.
//
// A phase writes a work report: it ran for minutes, nobody watched, and the
// panes that draw it lay out headings and sections. The supervisor is a
// conversation — somebody asked it something and is sitting there waiting —
// and the same contract turned every "what happened?" into a page of
// headings, bullet lists and file:// links.
func TestTheSupervisorIsAskedForAnAnswerAndNotAReport(t *testing.T) {
	asked := buildSupervisorPrompt("", "what happened while I was out?", nil)

	for _, want := range []string{"few sentences", "question"} {
		if !strings.Contains(strings.ToLower(asked), want) {
			t.Errorf("the supervisor is not asked for %q:\n%s", want, asked)
		}
	}

	if strings.Contains(asked, "Head sections with `##`") {
		t.Errorf("the supervisor still carries the contract a phase answers to:\n%s", asked)
	}
}

// TestTheSupervisorIsToldNotToLinkLocalFiles. Nothing in a terminal can be
// clicked, and a file:// address is a hundred characters of noise that wraps
// mid-path in the middle of a sentence.
func TestTheSupervisorIsToldNotToLinkLocalFiles(t *testing.T) {
	if asked := buildSupervisorPrompt("", "status", nil); !strings.Contains(asked, "file://") {
		t.Errorf("nothing tells the supervisor to leave local links out:\n%s", asked)
	}
}
