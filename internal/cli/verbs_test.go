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
func TestEveryGeneratedCommandCanBeAskedFor(t *testing.T) {
	for _, c := range fromVerbs(handWritten(t)) {
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

// handWritten is the command table without the generated tail, which is
// what fromVerbs is asked to fill in.
func handWritten(t *testing.T) []Command {
	t.Helper()

	var hand []Command

	for _, c := range commands() {
		if c.Args == "" || !strings.HasPrefix(c.Args, "[-repo <dir>]") {
			hand = append(hand, c)
		}
	}

	return hand
}
