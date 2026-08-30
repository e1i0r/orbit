package cli

// The three commands that reach GitHub through gh, and the sentences they
// say on the way. Every one of them was written where it was printed, which
// is the state internal/words exists to end: a reader whose cockpit is in
// Spanish met English the moment a pull request was involved.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

// ghCommands is the three wrappers, by the name a reader types.
func ghCommands() map[string]func(Context, []string) error {
	return map[string]func(Context, []string) error{
		"pr":       createPR,
		"merge":    mergePR,
		"close-pr": closePR,
	}
}

// TestTheGhCommandsRefuseInTheReadersLanguage asks the one thing all three
// can be asked without a repository, a branch or gh: what they say when the
// task identifier is missing. It is one sentence for the three of them, with
// the command's own name left untranslated inside it.
func TestTheGhCommandsRefuseInTheReadersLanguage(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	english, spanish := words.For("en"), words.For("es")

	for name, run := range ghCommands() {
		t.Run(name, func(t *testing.T) {
			var out, errOut bytes.Buffer

			ctx := Context{Out: &out, Err: &errOut}

			ctx.Words = english

			said := run(ctx, nil)

			ctx.Words = spanish

			dicho := run(ctx, nil)

			if said == nil || dicho == nil {
				t.Fatalf("%s ran with no task identifier: %v, %v", name, said, dicho)
			}

			if said.Error() == dicho.Error() {
				t.Errorf("both readers get %q — the sentence is still written where it is printed", said)
			}

			if !strings.Contains(dicho.Error(), name) {
				t.Errorf("the Spanish refusal is %q, which does not name the command to type", dicho)
			}
		})
	}
}

// TestTheClosingCommentIsInTheReadersLanguage. The comment is read on GitHub
// rather than in the terminal, which is why it was left in English longest.
// Whoever closed the task from a Spanish cockpit is the one who reads it back.
func TestTheClosingCommentIsInTheReadersLanguage(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	english := closingComment(Context{Words: words.For("en")})
	if spanish := closingComment(Context{Words: words.For("es")}); spanish == english {
		t.Errorf("both readers leave %q on the pull request", english)
	}
}
