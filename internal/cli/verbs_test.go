package cli

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/verb"
)

// TestEveryGeneratedCommandCanBeAskedFor is the test that was missing when
// `orbit learn` panicked on every invocation.
//
// A generated command builds its flags from what the verb declares, and the
// declaration is free to name a field anything — including one this package
// registers itself. Nothing ran those commands, so a collision that takes
// the process down with `flag redefined` shipped green.
//
// It asks for every one of them with no arguments. What comes back does not
// matter: a refusal is the ordinary answer for a verb that needs a task or
// a word, and the refusal is proof the flags were built and parsed. What is
// being held is that none of them panics.
//
// Nothing is held back as hand-written. A verb the command table spells for
// itself is still generated here, because what is being asked is whether
// the declaration can be turned into a command at all.
func TestEveryGeneratedCommandCanBeAskedFor(t *testing.T) {
	for _, c := range fromVerbs(nil) {
		t.Run(c.Name, func(t *testing.T) {
			defer func() {
				if died := recover(); died != nil {
					t.Fatalf("orbit %s panicked: %v", c.Name, died)
				}
			}()

			var out, said strings.Builder

			_ = c.Run(Context{Out: &out, Err: &said}, nil) //nolint:errcheck // see above
		})
	}
}

// TestNoGeneratedCommandFightsTheRepoFlag is the same defect, asked about
// directly: every command takes -repo, so a verb that declares a field by
// that name must be the one that wins rather than the one that crashes.
func TestNoGeneratedCommandFightsTheRepoFlag(t *testing.T) {
	for _, v := range verb.Every() {
		for _, f := range v.Takes {
			if f.Name != "repo" {
				continue
			}

			if f.Kind != verb.Named {
				t.Errorf("%s declares repo as something other than a name", v.Name)
			}
		}
	}
}

// TestAFamilyIsReachedByItsSecondWord.
//
// `orbit rules keep` and never `orbit keep`, which says nothing about what.
// The refusal is the proof: the child is what answers, and it names itself
// by both its words.
func TestAFamilyIsReachedByItsSecondWord(t *testing.T) {
	var out, said strings.Builder

	err := commandNamed(t, "rules").Run(Context{Out: &out, Err: &said}, []string{"keep"})
	if err == nil || !strings.Contains(err.Error(), "rules keep needs n") {
		t.Errorf("orbit rules keep answered %v", err)
	}
}

// TestAFamilysUsageLineNamesItsChildren, because that line is the whole of
// what a reader is told before they type.
func TestAFamilysUsageLineNamesItsChildren(t *testing.T) {
	args := commandNamed(t, "rules").Args

	for _, word := range []string{"keep", "drop", "<n>", "-check", "<text>"} {
		if !strings.Contains(args, word) {
			t.Errorf("orbit rules %s says nothing about %q", args, word)
		}
	}
}

// commandNamed is one of the generated commands.
func commandNamed(t *testing.T, name string) Command {
	t.Helper()

	for _, c := range fromVerbs(nil) {
		if c.Name == name {
			return c
		}
	}

	t.Fatalf("nothing generated a command called %q", name)

	return Command{}
}
