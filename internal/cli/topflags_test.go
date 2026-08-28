package cli

// top's flags, asked directly what they read.
//
// This is the only file in the package where -once can decide anything.
// Everything in top_test.go goes through run(), which hands the command a
// bytes.Buffer, and interactive() refuses any writer that is not the
// process's own os.Stdout — so those tests take the plain branch whether or
// not the flag was ever parsed, and they stayed green with -once neutralised
// entirely. Asking parseTop what it parsed is what makes the flag this
// command exists for a thing that can fail.

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

// flag stops at the first argument that is not a flag, so a directory typed
// before -once would leave the flag unread — and that failure is not a
// message but a full-screen window where a frame was asked for, in a pipe or
// in CI. Both orders, and a flag on each side of the directory, have to come
// back with the flag set.
func TestTopReadsItsFlagsOnEitherSideOfTheDirectory(t *testing.T) {
	const dir = "/work/repos"

	cases := []struct {
		name string
		args []string
		once bool
		lang string
	}{
		{"the directory before the flag", []string{dir, "-once"}, true, ""},
		{"the directory after the flag", []string{"-once", dir}, true, ""},
		{"both flags after the directory", []string{dir, "-once", "-lang", "es"}, true, "es"},
		{"both flags before it", []string{"-once", "-lang", "es", dir}, true, "es"},
		{"one flag on each side", []string{"-lang", "es", dir, "-once"}, true, "es"},
		{"a flag with a value, after the directory", []string{dir, "-lang", "es"}, false, "es"},
		{"no flags at all", []string{dir}, false, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, once, lang, err := parseTop(Context{Out: io.Discard}, c.args)
			if err != nil {
				t.Fatalf("parseTop(%q): %v", c.args, err)
			}

			if got != dir {
				t.Errorf("parseTop(%q) watched %q, want %q", c.args, got, dir)
			}

			if once != c.once {
				t.Errorf("parseTop(%q) read -once as %v, want %v", c.args, once, c.once)
			}

			if lang != c.lang {
				t.Errorf("parseTop(%q) read -lang as %q, want %q", c.args, lang, c.lang)
			}
		})
	}
}

// No directory is the working one, because `orbit top` on its own is what a
// person types and they mean where they are.
func TestTopWithNoDirectoryWatchesTheOneYouAreIn(t *testing.T) {
	want, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	dir, once, lang, err := parseTop(Context{Out: io.Discard}, []string{"-once"})
	if err != nil {
		t.Fatalf("parseTop: %v", err)
	}

	if dir != want || !once || lang != "" {
		t.Errorf("parseTop(-once) = (%q, %v, %q), want (%q, true, \"\")", dir, once, lang, want)
	}
}

// Two directories is a person meaning something this command cannot do, and
// picking the first is how they find out an hour later.
func TestTopRefusesTwoDirectoriesRatherThanPickingOne(t *testing.T) {
	_, _, _, err := parseTop(Context{Out: io.Discard}, []string{"/work/one", "/work/two", "-once"})
	if err == nil {
		t.Fatal("two directories were accepted and one of them chosen silently")
	}

	if !strings.Contains(err.Error(), "one directory") {
		t.Errorf("the refusal is %q, want it to say how many directories it takes", err)
	}
}

// Asking a program what it takes is not a failure: -h prints the flags and
// the dispatcher turns errHelpShown into exit 0.
func TestTopPrintsItsFlagsWhenAsked(t *testing.T) {
	var b strings.Builder

	_, _, _, err := parseTop(Context{Out: &b}, []string{"-h"})
	if !errors.Is(err, errHelpShown) {
		t.Fatalf("parseTop(-h) answered %v, want errHelpShown", err)
	}

	for _, want := range []string{"-once", "-lang"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("the help does not mention %q:\n%s", want, b.String())
		}
	}
}

// A flag nobody declared comes back as an error with the flags beside it,
// rather than as flag's own message on a stream this program does not own.
func TestTopRefusesAFlagItDoesNotHave(t *testing.T) {
	_, _, _, err := parseTop(Context{Out: io.Discard}, []string{"-onec"})
	if err == nil {
		t.Fatal("an undeclared flag was accepted")
	}

	if !strings.Contains(err.Error(), "onec") {
		t.Errorf("the refusal is %q, want it to name the flag", err)
	}
}
