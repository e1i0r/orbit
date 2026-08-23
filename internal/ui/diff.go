package ui

// The diff tab: what the task changed, and the one key that leaves the
// window for an editor.
//
// It is git's own output, coloured on the two characters that carry the
// meaning and nothing else. There is no syntax highlighting and there will
// not be: it would mean a lexer per language in a terminal cockpit, and the
// question this pane answers — what did the agent change — is answered by
// the plus and the minus.
//
// Long lines scroll rather than wrap. A diff of a generated file arrives as
// one line of several thousand cells, and wrapping it would push the rest of
// the hunk off the bottom of the screen; the pane cuts, and ← and → move
// along it.

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/view"
)

// diffLines is the diff tab's content, ready for the pane.
func (m Model) diffLines() []string {
	text := strings.TrimSuffix(m.diff, "\n")
	if strings.TrimSpace(text) == "" {
		return []string{" " + Paint(Dim).Render(m.opts.Words.T("diff.unchanged", "no changes in this task's worktree"))}
	}
	raw := strings.Split(text, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		out = append(out, " "+Paint(diffRole(line)).Render(line))
	}
	return out
}

// diffRole is the colour of one diff line.
//
// The order of the tests is what makes it correct rather than nearly
// correct: `+++ b/x` starts with a plus and is not an added line, and a file
// header painted green is a header a reader reads as content.
func diffRole(line string) Role {
	switch {
	case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"),
		strings.HasPrefix(line, "diff "), strings.HasPrefix(line, "index "):
		return Dim
	case strings.HasPrefix(line, "@@"):
		return Accent
	case strings.HasPrefix(line, "+"):
		return OK
	case strings.HasPrefix(line, "-"):
		return Bad
	}
	return Dim
}

// edit opens the file under the top of the diff in $EDITOR.
//
// It refuses on the other two tabs rather than doing nothing. A key that
// silently does nothing is indistinguishable from a key that is broken, and
// the reader's next move is to press it again.
func (m Model) edit() (tea.Model, tea.Cmd) {
	p := m.opts.Words
	if m.tab != tabDiff {
		return m.say(p.T("msg.only_the_diff", "{key} opens a file, and only the diff tab has one to open",
			about("key", m.keys.Edit.Help().Key))), nil
	}
	cmd, err := m.editorFor()
	if err != nil {
		return m.say(err.Error()), nil
	}
	// ExecProcess suspends the window, hands the terminal to the editor and
	// redraws when it comes back. The alternative — running the editor with
	// the alternate screen still up — is what makes a terminal program look
	// like it has crashed.
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return editorMsg{Err: err} })
}

// editorFor builds the command $EDITOR would be run as: the file the top
// line of the pane belongs to, at the line of that file it is on, in the
// task's worktree.
//
// The line is taken from the top of the pane and not from a cursor, because
// this pane has no cursor. What a reader means by "open this" is the hunk
// they are looking at, and the top of the pane is the closest honest reading
// of that.
//
// The errors here are the sentences the band says, translated at this call
// site, which is the one place in the window an error is composed rather
// than passed on. They are refusals of the window's own — not a command's
// verdict — so there is nothing upstream whose words could go stale.
func (m Model) editorFor() (*exec.Cmd, error) {
	p := m.opts.Words
	editor := cmp.Or(os.Getenv("VISUAL"), os.Getenv("EDITOR"))
	if editor == "" {
		return nil, errors.New(p.T("msg.no_editor", "no $EDITOR is set, so there is nothing to open the file with"))
	}
	file, line, ok := fileAt(strings.Split(strings.TrimSuffix(m.diff, "\n"), "\n"), m.panes[tabDiff].YOffset())
	if !ok {
		return nil, errors.New(p.T("msg.no_file_here", "there is no file on this line of the diff"))
	}
	if m.worktree == "" {
		return nil, errors.New(p.T("msg.no_worktree", "this task has no worktree to open the file in"))
	}
	cmd := exec.Command(editor, "+"+strconv.Itoa(line), file)
	cmd.Dir = m.worktree
	return cmd, nil
}

// fileAt is the file one line of a diff belongs to, and the line of that
// file it lands on.
//
// The line is counted rather than parsed, because a hunk header only says
// where the hunk begins. A removed line is not in the new file and does not
// advance the count; a context line and an added line both do. Getting this
// wrong opens the right file at the wrong line, which is worse than not
// opening it at all — a reader believes what the editor shows them.
//
// A line that is not inside a hunk still answers. Sitting on `diff --git` or
// on `+++ b/x` and pressing the key means "open this file", and refusing
// because there is no line number to be precise about would be refusing the
// only thing the reader asked for; line 1 is the honest answer there.
func fileAt(lines []string, at int) (string, int, bool) {
	if at < 0 || at >= len(lines) {
		return "", 0, false
	}
	// Walking up, whichever comes first decides: a hunk header means the
	// cursor is inside a hunk and the line has to be counted, and a file
	// header means it is in the furniture between two files.
	line, from := 1, at
	for i := at; i >= 0; i-- {
		if name, ok := fileHeader(lines[i]); ok {
			return name, 1, true
		}
		if start, ok := hunkStart(lines[i]); ok {
			line, from = start, i
			break
		}
	}
	for i := from + 1; i < at; i++ {
		if !strings.HasPrefix(lines[i], "-") && !strings.HasPrefix(lines[i], `\`) {
			line++
		}
	}
	for i := from; i >= 0; i-- {
		if name, ok := fileHeader(lines[i]); ok {
			return name, line, true
		}
	}
	// Above every file header there is still the `diff --git` line the
	// pane opens on, and the file it announces is the one below it.
	for i := at; i < len(lines); i++ {
		if name, ok := fileHeader(lines[i]); ok {
			return name, 1, true
		}
	}
	return "", 0, false
}

// fileHeader reads the new file's path out of a diff's `+++` line. The `b/`
// prefix is git's own and is not part of any path in the worktree.
func fileHeader(s string) (string, bool) {
	if name, ok := strings.CutPrefix(s, "+++ b/"); ok {
		return name, true
	}
	if name, ok := strings.CutPrefix(s, "+++ "); ok && name != "/dev/null" {
		return name, true
	}
	return "", false
}

// hunkStart reads the new file's first line number out of a hunk header:
// `@@ -12,7 +34,9 @@` begins at line 34 of the new file.
func hunkStart(s string) (int, bool) {
	if !strings.HasPrefix(s, "@@") {
		return 0, false
	}
	_, after, ok := strings.Cut(s, "+")
	if !ok {
		return 0, false
	}
	digits, _, _ := strings.Cut(after, ",")
	digits, _, _ = strings.Cut(digits, " ")
	n, err := strconv.Atoi(digits)
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

// diffOf runs git diff in the task's worktree, against the branch the
// repository's work is measured from.
//
// The worktree is asked for through the port rather than worked out here.
// Where a task's checkout lives is internal/store's answer — it is a hash of
// the repository's path under the state root — and internal/ui may not name
// that package. Running the diff in the repository instead, which is what
// this did before the tab that draws it existed, shows the reader whatever
// they happen to have uncommitted in their own checkout under the heading of
// an agent's task.
func diffOf(r Reader, t view.Task) tea.Cmd {
	return func() tea.Msg {
		if r == nil {
			return diffMsg{ID: t.ID, Err: errors.New("this window was opened without a way to find the worktree")}
		}
		if t.RepoPath == "" {
			return diffMsg{ID: t.ID, Err: errors.New("this task does not say where its repository is")}
		}
		dir, err := r.Worktree(t.RepoPath, t.ID)
		if err != nil {
			return diffMsg{ID: t.ID, Err: err}
		}
		out, err := gitDiff(dir, baseOf(t.RepoPath))
		if err != nil {
			return diffMsg{ID: t.ID, Tree: dir, Err: err}
		}
		return diffMsg{ID: t.ID, Tree: dir, Text: out}
	}
}

// gitDiff asks git twice at most: once against the base branch, and once for
// the working tree alone if that failed.
//
// The fallback is not a way of hiding an error. A worktree cut before its
// base existed, or one whose base has been deleted since, still has changes
// worth showing, and refusing to show them because the comparison is
// unavailable would be refusing the reader the only view they have. If the
// plain diff fails too, that failure is the one reported — it is the one
// that says the worktree itself is not there.
func gitDiff(dir, base string) (string, error) {
	if base != "" {
		if out, err := exec.Command("git", "-C", dir, "diff", "--merge-base", base).CombinedOutput(); err == nil {
			return string(out), nil
		}
	}
	out, err := exec.Command("git", "-C", dir, "diff").CombinedOutput()
	if err != nil {
		// CombinedOutput is what git said; err is only "exit status 128".
		// Dropping the bytes turns "not a git repository" into a number the
		// reader has to go and look up somewhere else, so they go in the
		// message and err stays wrapped underneath.
		return "", fmt.Errorf("git diff in %s: %w: %s", dir, err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// baseOf is the branch a repository's work is measured against, or nothing
// when it cannot be read. Nothing is a usable answer — gitDiff falls back to
// the working tree — so a repository that is detached, or gone, costs the
// comparison and not the pane.
func baseOf(repoPath string) string {
	r, err := repo.Open(repoPath)
	if err != nil {
		return ""
	}
	return r.Base
}
