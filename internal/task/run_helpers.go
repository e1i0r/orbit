package task

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

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

// prompt is what the engine is told for one phase.
func prompt(t Task, p flow.Phase, notes []string, prevOutput string) string {
	base := fmt.Sprintf("Phase: %s\nRepository: %s\n\nTask %s:\n%s\n", p.Name, t.Repo.Name, t.ID, t.Text)
	if p.Prompt != "" {
		base += fmt.Sprintf("\nPhase Instructions:\n%s\n", p.Prompt)
	}

	if prevOutput != "" {
		base += fmt.Sprintf("\nPrevious Phase Output:\n%s\n", prevOutput)
	}

	if len(notes) == 0 {
		return base
	}

	var sb strings.Builder
	sb.WriteString(base + "\nOperator Notes:\n")

	for _, n := range notes {
		sb.WriteString("- " + n + "\n")
	}

	return sb.String()
}
