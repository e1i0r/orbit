package ui

// What the operator typed into the supervisor, taken apart.

import "testing"

// TestPlainTextIsAMessage. Everything that is not one of the three gestures
// is what it has always been, and nothing that starts with a slash by
// accident becomes a command by accident either.
func TestPlainTextIsAMessage(t *testing.T) {
	for _, said := range []string{
		"look at ORB-115, it is stuck",
		"/ruleset the thing",
		"/",
		"and/or this",
		"@",
	} {
		if got := parseSaid(said); got.Kind != saidMessage || got.Phrase != said {
			t.Errorf("%q was read as %+v, want the message it is", said, got)
		}
	}
}

// TestSlashRuleIsAFactThatStops, and slash aware one that only warns. The
// two words are the two things a fact can do, and the operator says which
// by which word they use.
func TestSlashRuleIsAFactThatStops(t *testing.T) {
	rule := parseSaid("/rule coverage stays above 90%")
	if rule.Kind != saidRule || rule.Phrase != "coverage stays above 90%" {
		t.Errorf("/rule was read as %+v", rule)
	}

	aware := parseSaid("/aware the fuzz tests hang sometimes")
	if aware.Kind != saidAware || aware.Phrase != "the fuzz tests hang sometimes" {
		t.Errorf("/aware was read as %+v", aware)
	}
}

// TestAScopeCanBeSaidOutLoud. Without one a fact is about the repository
// being worked in, which is what somebody typing quickly means. The two
// scopes above a repository have to be asked for, because they reach
// everywhere and nobody should arrive at them by not saying anything.
func TestAScopeCanBeSaidOutLoud(t *testing.T) {
	for said, want := range map[string]spoken{
		"/rule the PRs are in English":            {Kind: saidRule, Phrase: "the PRs are in English"},
		"/rule --general the PRs are in English":  {Kind: saidRule, Scope: "general", Phrase: "the PRs are in English"},
		"/aware --lang go never discard an error": {Kind: saidAware, Scope: "go", Phrase: "never discard an error"},
	} {
		if got := parseSaid(said); got != want {
			t.Errorf("%q was read as %+v,\nwant %+v", said, got, want)
		}
	}
}

// TestAtPointsAtATask. What follows lands in that task's notes, where
// somebody opening it tomorrow will read it.
func TestAtPointsAtATask(t *testing.T) {
	got := parseSaid("@ORB-115 this one is stuck, do not let it hang")
	if got.Kind != saidNote || got.Task != "ORB-115" {
		t.Errorf("the mention was read as %+v", got)
	}

	if got.Phrase != "this one is stuck, do not let it hang" {
		t.Errorf("the note says %q", got.Phrase)
	}
}

// TestAGestureWithNothingAfterItSaysNothing. "/rule" alone is somebody who
// has not finished typing, not a rule with an empty sentence.
func TestAGestureWithNothingAfterItSaysNothing(t *testing.T) {
	for _, said := range []string{"/rule", "/rule   ", "/aware", "@ORB-115", "/rule --general"} {
		if got := parseSaid(said); got.Kind != saidNothing {
			t.Errorf("%q was read as %+v, want nothing to act on", said, got)
		}
	}
}
