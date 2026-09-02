package cli

import (
	"flag"
	"io"
	"strings"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/task"
)

func createPR(ctx Context, args []string) error {
	fs := flag.NewFlagSet("pr", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dir := fs.String("repo", ".", "a repository the task was worked in")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	if len(fs.Args()) < 1 {
		return needsTaskID(ctx, "pr")
	}

	taskID := fs.Args()[0]

	s, r, err := openBoth(*dir)
	if err != nil {
		logger.Error("cli/pr", "open repository %q failed: %v", *dir, err)
		return err
	}

	// Where the task was worked is read before the task is. That listing is
	// where the identifier is measured against what a path can hold, and a
	// task read first measures the same thing on its way to the file, which
	// leaves this check standing in front of a case already refused.
	where, err := worked(s, r, taskID)
	if err != nil {
		logger.Error("cli/pr", "read the repositories of task %q failed: %v", taskID, err)
		return err
	}

	t, err := task.Load(s, r, taskID)
	if err != nil {
		logger.Error("cli/pr", "load task %q failed: %v", taskID, err)
		return err
	}

	return deliverTask(ctx, s, t, where)
}

// clipWords cuts s down to at most limit characters, at the last space when
// there is a reasonable one, and never through the middle of a character.
//
// Characters and not bytes. len counts bytes, so a title written in a script
// whose letters take more than one — an accent, a kanji — is cut halfway
// through a letter: git records the half, the pull request title carries
// U+FFFD, and nothing on the way said anything had gone wrong. Cutting on
// runes costs one conversion and cannot do that.
//
// The last space is preferred so a subject ends on a whole word, and a space
// too near the start is ignored — a title whose first word is longer than the
// limit would otherwise be cut down to almost nothing.
func clipWords(s string, limit int) string {
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	// A space is one byte and always a boundary, so a byte index into the
	// already-shortened string is a safe place to cut it a second time.
	cut := string(r[:limit])
	if i := strings.LastIndex(cut, " "); i > 20 {
		return cut[:i]
	}

	return cut
}
