package task

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/store"
)

// maxOutput is how much of an engine's answer is kept in the record.
const maxOutput = 1 << 20

// captured cuts an engine's output down to what the record can hold and says
// in the text when it had to. Truncation that announces itself is honest;
// silent loss is not. The second return is the size of what was said, zero
// when nothing was cut.
func captured(out string) (text string, full int) {
	if len(out) <= maxOutput {
		return out, 0
	}

	n := maxOutput
	// Never cut a rune in half: the record is UTF-8, and a severed tail
	// would come back from the log as a replacement character.
	for n > 0 && !utf8.RuneStart(out[n]) {
		n--
	}

	return out[:n] + fmt.Sprintf("\n…[truncated, full output was %d bytes]", len(out)), len(out)
}

// prepare makes the worktree, reusing one that is already there so that a
// re-run picks up where the last one stopped rather than starting over.
func prepare(s *store.Store, t Task) (string, error) {
	wt, err := s.CreateWorktreeParent(t.Repo.Path, t.ID)
	if err != nil {
		return "", err
	}

	if _, statErr := os.Stat(wt); statErr == nil {
		return wt, nil
	}

	if err := t.Repo.AddWorktree(wt, "orbit/"+t.ID); err != nil {
		return "", err
	}

	return wt, nil
}

// prompt is what the engine is told for one phase: the task, the phase it is
// running, what the phase before it said, and whatever the operator has
// added since.
//
// It is written in Markdown because the answer is asked for in Markdown, and
// a prompt that asks in one shape for another is asking twice.
func prompt(t Task, p flow.Phase, notes []string, prevOutput string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s\n\n%s\n\n", t.ID, strings.TrimSpace(t.Text))
	fmt.Fprintf(&b, "## Phase\n\n`%s`, in repository `%s`.\n", p.Name, t.Repo.Name)

	if p.Prompt != "" {
		fmt.Fprintf(&b, "\n## Phase instructions\n\n%s\n", strings.TrimSpace(p.Prompt))
	}

	// Fenced rather than set as prose: what the phase before wrote is
	// Markdown of its own, and its headings loose under a heading of this
	// prompt would read as sections of the prompt.
	if prevOutput != "" {
		fmt.Fprintf(&b, "\n## Previous phase output\n\n%s\n", engine.Fenced(prevOutput))
	}

	if len(notes) > 0 {
		b.WriteString("\n## Operator notes\n\n")

		for _, n := range notes {
			fmt.Fprintf(&b, "- %s\n", n)
		}
	}

	b.WriteString("\n" + engine.AnswerContract)

	return b.String()
}
