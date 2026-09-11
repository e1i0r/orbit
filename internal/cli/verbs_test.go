package cli

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
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

	err := commandNamed(t, fromVerbs(nil), "rules").Run(Context{Out: &out, Err: &said}, []string{"keep"})
	if err == nil || !strings.Contains(err.Error(), "rules keep needs n") {
		t.Errorf("orbit rules keep answered %v", err)
	}
}

// TestAFamilysUsageLineNamesItsChildren, because that line is the whole of
// what a reader is told before they type.
func TestAFamilysUsageLineNamesItsChildren(t *testing.T) {
	args := commandNamed(t, fromVerbs(nil), "rules").Args

	for _, word := range []string{"keep", "drop", "<n>", "-check", "<text>"} {
		if !strings.Contains(args, word) {
			t.Errorf("orbit rules %s says nothing about %q", args, word)
		}
	}
}

// TestAHandWrittenParentDispatchesItsChildren.
//
// `orbit pr` is written by hand because of what it does with the terminal,
// and it still has to be the front of its family. The dispatch is the
// generated one rather than a second copy written here.
func TestAHandWrittenParentDispatchesItsChildren(t *testing.T) {
	pr := commandNamed(t, commands(), "pr")

	for _, word := range []string{"merge", "close"} {
		if !strings.Contains(pr.Args, word) {
			t.Errorf("orbit pr %s says nothing about %q", pr.Args, word)
		}
	}

	if !strings.Contains(pr.Args, "-repo <dir> <id>") {
		t.Errorf("orbit pr %s no longer says what it takes on its own", pr.Args)
	}
}

// TestAnOldNameIsACommandThatSaysWhereItWent.
//
// It still works, so a script does not break. What it does not do is look
// like a second verb on the help screen: the line says where the name went,
// and the new spelling is the one that describes the work.
func TestAnOldNameIsACommandThatSaysWhereItWent(t *testing.T) {
	for was, now := range map[string]string{
		"merge":    "orbit pr merge",
		"close-pr": "orbit pr close",
		"set":      "orbit settings set",
	} {
		c := commandNamed(t, commands(), was)

		if got := c.About(words.For("")); got != "the old name for "+now {
			t.Errorf("orbit %s says %q about itself", was, got)
		}
	}
}

// TestAnOldNameSaysWhatToTypeInsteadOnTheErrorStream, so that a reader who
// piped the answer somewhere still gets told, and what they piped is only
// the answer.
func TestAnOldNameSaysWhatToTypeInsteadOnTheErrorStream(t *testing.T) {
	var out, said strings.Builder

	// Asked for with nothing after it, so what comes back is the refusal
	// that says what it needs. The notice is said before the command runs,
	// which is what makes it reach a reader either way.
	if err := commandNamed(t, commands(), "set").Run(Context{Out: &out, Err: &said}, nil); err == nil {
		t.Error("orbit set with no setting named was accepted")
	}

	if !strings.Contains(said.String(), "set is now settings set") {
		t.Errorf("orbit set said %q on the error stream", said.String())
	}

	if strings.Contains(out.String(), "is now settings set") {
		t.Errorf("the notice was mixed into the answer: %q", out.String())
	}
}

// commandNamed is one command out of a table.
func commandNamed(t *testing.T, table []Command, name string) Command {
	t.Helper()

	for _, c := range table {
		if c.Name == name {
			return c
		}
	}

	t.Fatalf("nothing is called %q", name)

	return Command{}
}
