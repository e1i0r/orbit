package cli

// Where a command gets its language from.
//
// docs/env.md calls $ORBIT_LANG "the language for one command, over the
// settings file", and shows `ORBIT_LANG=en orbit quota` as the way to use
// it. It reached `orbit top` and nothing else: every other command took its
// printer from the saved setting alone, so the variable was weighed once, at
// the window's composition root, and ignored by the twenty-odd commands a
// reader types it in front of.
//
// These two tests are the pair the documented sentence needs — the variable
// is read at all, and it beats what was saved.

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestORBITLANGIsWeighedByCommandsThatAreNotTop runs everyday commands over
// an empty settings file with the variable set, and refuses the English they
// used to answer with.
//
// The five are five layers rather than five commands: a reading printed
// through a verb, a refusal raised by one, a refusal the command line writes
// itself, the usage screen the dispatcher prints before any command is
// looked up, and the one listing that used to open the settings file for a
// language of its own. A fix that reaches some of those and not the rest
// leaves the variable half working, which is the shape this bug already had.
func TestORBITLANGIsWeighedByCommandsThatAreNotTop(t *testing.T) {
	root, _ := workspace(t)
	dir := filepath.Join(root, "payments")

	t.Setenv("LANG", "")
	t.Setenv("ORBIT_LANG", "es")

	for _, step := range []struct {
		args    []string
		english string
	}{
		{args: []string{"board", "list", "-repo", dir}, english: "no tasks yet"},
		{args: []string{"task", "show", "-repo", dir, "PAY-404"}, english: "nothing recorded for"},
		{args: []string{"task", "note", "-repo", dir, "PAY-1"}, english: "needs text"},
		{args: []string{"help"}, english: "State lives in"},
		{args: []string{"flows"}, english: "built in"},
	} {
		t.Run(strings.Join(step.args, " "), func(t *testing.T) {
			_, out, errOut := run(t, step.args...)

			said := out + errOut
			if strings.TrimSpace(said) == "" {
				t.Fatalf("%v said nothing at all", step.args)
			}

			if strings.Contains(said, step.english) {
				t.Errorf("$ORBIT_LANG is es and %v answered %q", step.args, said)
			}
		})
	}
}

// TestORBITLANGWinsOverTheSavedLanguage is the other half of the documented
// sentence: it is the language for one command, over the settings file, so a
// reader who saved Spanish and typed the variable in front of one command
// gets that one command in English and leaves the window alone.
func TestORBITLANGWinsOverTheSavedLanguage(t *testing.T) {
	dir := speaking(t, "es")

	t.Setenv("LANG", "")
	t.Setenv("ORBIT_LANG", "en")

	_, _, errOut := run(t, "task", "note", "-repo", dir, "PAY-1")
	if !strings.Contains(errOut, "needs text") {
		t.Errorf("$ORBIT_LANG is en over a saved es, and the refusal was %q", errOut)
	}
}
