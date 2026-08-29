package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/task"
)

func createPR(ctx Context, args []string) error {
	fs := flag.NewFlagSet("pr", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dir := fs.String("repo", ".", "the repository the task is against")
	if err := parse(ctx, fs, args); err != nil {
		return err
	}

	if len(fs.Args()) < 1 {
		return fmt.Errorf("pr needs the task identifier")
	}

	taskID := fs.Args()[0]

	s, r, err := openBoth(*dir)
	if err != nil {
		logger.Error("cli/pr", "open repository %q failed: %v", *dir, err)
		return err
	}

	t, err := task.Load(s, r, taskID)
	if err != nil {
		logger.Error("cli/pr", "load task %q failed: %v", taskID, err)
		return err
	}

	wtDir, err := s.WorktreeDir(r.Path, taskID)
	if err != nil {
		logger.Error("cli/pr", "get worktree for task %q failed: %v", taskID, err)
		return err
	}

	branch := "orbit/" + taskID

	firstLine := strings.SplitN(strings.TrimSpace(t.Text), "\n", 2)[0]

	commitMsg := clipWords(fmt.Sprintf("feat(%s): %s", taskID, firstLine), 72)

	if err := r.SyncBaseBranch(wtDir, r.Base); err != nil {
		logger.Warn("cli/pr", "sync base branch %q into %q: %v", r.Base, wtDir, err)
	}

	if err := r.CommitWorktree(wtDir, commitMsg); err != nil {
		logger.Error("cli/pr", "commit worktree %q failed: %v", wtDir, err)
		return err
	}

	if err := r.PushBranch(wtDir, branch); err != nil {
		logger.Error("cli/pr", "push branch %q failed: %v", branch, err)
		return err
	}

	body := fmt.Sprintf("## Orbit Task: %s\n\n%s\n\nGenerated automatically by Orbit.", taskID, t.Text)

	title := fmt.Sprintf("%s: %s", taskID, firstLine)
	// 87 and not 90, because the three dots are part of what a reader sees:
	// the branch that cuts at a space appends them, and 90 plus three dots
	// is 93 characters in a field meant to hold 90.
	if short := clipWords(title, 87); short != title {
		title = short + "..."
	}

	prURL, err := r.CreatePR(wtDir, title, body, branch, r.Base)
	if err != nil {
		logger.Error("cli/pr", "gh pr create failed: %v", err)
		return fmt.Errorf("branch %q pushed, but gh pr create failed: %w", branch, err)
	}

	logger.Info("cli/pr", "created pull request %s for task %s (base=%s)", prURL, taskID, r.Base)
	fmt.Fprintf(ctx.Out, "Pull Request created: %s\n", prURL)

	return nil
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
