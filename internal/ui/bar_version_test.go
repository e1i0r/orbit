package ui

// The version in the corner of the bar: what a build calls itself, cut down
// to what that corner has room for.

import (
	"strings"
	"testing"
)

// TestWhatABuildCallsItselfInTheCorner over every shape the two things that
// stamp a version hand in: goreleaser, which passes the bare tag, and
// `git describe`, which passes the tag plus where the checkout is relative
// to it.
func TestWhatABuildCallsItselfInTheCorner(t *testing.T) {
	for _, c := range []struct{ stamped, want string }{
		{"0.1.83", "v0.1.83"},
		{"v0.1.83", "v0.1.83"},
		{"v0.1.83-27-gabd558b", "v0.1.83+27"},
		{"v0.1.83-27-gabd558b-dirty", "v0.1.83+27*"},
		{"v0.1.83-dirty", "v0.1.83*"},
		{"abd558b", "abd558b"},
		{"dev", "dev"},
		{"", ""},
	} {
		if got := buildMark(c.stamped); got != c.want {
			t.Errorf("a build stamped %q calls itself %q, want %q", c.stamped, got, c.want)
		}
	}
}

// TestTheVersionIsAtTheFarEndOfTheBar: it is the last chip, so it sits in the
// corner of the window and not between two things that are pressed.
func TestTheVersionIsAtTheFarEndOfTheBar(t *testing.T) {
	m, _ := testModel(t, 120, 30)
	m.opts.Version = "v0.1.83-27-gabd558b"

	chips := m.barFooterChips()
	if len(chips) == 0 {
		t.Fatal("the bar has no chips at all")
	}

	last := chips[len(chips)-1]
	if !strings.Contains(last.text, "v0.1.83+27") {
		t.Errorf("the last chip reads %q, want the version", last.text)
	}

	// Read and not pressed: there is no screen behind a version.
	if last.target.Kind != TargetNone {
		t.Errorf("a click on the version = %+v, want nothing", last.target)
	}

	line := m.barLine(120)
	if !strings.Contains(line, "v0.1.83+27") {
		t.Errorf("the bar is drawn as %q, with no version in it", line)
	}
}

// TestABuildWithNoVersionStampedShowsNoChip. That is `go test`, where the
// version would be a fact about the harness rather than about orbit.
func TestABuildWithNoVersionStampedShowsNoChip(t *testing.T) {
	m, _ := testModel(t, 120, 30)
	m.opts.Version = ""

	for _, c := range m.barFooterChips() {
		if c.target.Kind == TargetNone {
			t.Errorf("an unstamped build drew %q at the end of the bar", c.text)
		}
	}
}
