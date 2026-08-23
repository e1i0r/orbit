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
//
// Running git and waiting for it is gitdiff.go's job, not this file's: what
// is here reads the answer once there is one — which file a line belongs
// to, what colour a line is, and which file and line the editor opens.

import (
	"cmp"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// diffLines is the diff tab's content, ready for the pane.
//
// It answers in one of three states, because there are three different true
// things it can say. Before the first diffMsg lands there is no answer yet,
// and that is not the same fact as an answer that came back empty —
// collapsing the two, which this pane once did, is how a git that hangs
// ends up asserting "no changes" on a question it was never actually
// answered. An answer that came back as a failure is the third state, and
// it is said rather than folded into the diff text.
func (m Model) diffLines() []string {
	p := m.opts.Words
	if !m.diffKnown {
		return []string{" " + Paint(Dim).Render(p.T("diff.pending", "reading this task's worktree…"))}
	}
	if m.diffErr != nil {
		return []string{" " + Paint(Bad).Render(m.diffErr.Error())}
	}
	text := strings.TrimSuffix(m.diff, "\n")
	if strings.TrimSpace(text) == "" {
		return []string{" " + Paint(Dim).Render(p.T("diff.unchanged", "no changes in this task's worktree"))}
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
//
// Walking up from the cursor, three things can be met first, and which one
// is met decides everything: a file's own `+++` header answers directly; a
// hunk header means the cursor is inside that hunk and the line has to be
// counted from there; and a boundary line — `diff --git`, `index ` or
// `--- ` — means the cursor is sitting in the furniture that introduces the
// NEXT file, not inside the hunk of the file above it. In a diff of more
// than one file, that boundary is met before the previous file's hunk
// header is, and walking on through it to keep looking upward used to be
// this function's bug: it would borrow the previous file's last hunk and
// answer with a line counted for a file the cursor was never on. Stopping
// at the boundary and handing off to fileBelow — which finds whichever
// file's `+++` comes next — is what a boundary line answers with instead.
func fileAt(lines []string, at int) (string, int, bool) {
	if at < 0 || at >= len(lines) {
		return "", 0, false
	}
	line, from, boundary := 1, at, false
	for i := at; i >= 0; i-- {
		if name, ok := fileHeader(lines[i]); ok {
			return name, 1, true
		}
		if start, ok := hunkStart(lines[i]); ok {
			line, from = start, i
			break
		}
		if fileBoundary(lines[i]) {
			boundary = true
			break
		}
	}
	if !boundary {
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
	}
	// Either the walk never found a hunk above the cursor at all, or it met
	// a boundary line before it found one — either way the file above the
	// cursor is not the file to answer with, and the honest answer is
	// whichever file's `+++` comes next, at its own first line.
	return fileBelow(lines, at)
}

// fileBelow is the file introduced by the next `+++` header at or after the
// given line. It is what a cursor sitting in furniture that has not reached
// its file's header yet answers with, and what fileAt hands off to once it
// knows the file above the cursor is the wrong one to ask.
func fileBelow(lines []string, at int) (string, int, bool) {
	for i := at; i < len(lines); i++ {
		if name, ok := fileHeader(lines[i]); ok {
			return name, 1, true
		}
	}
	return "", 0, false
}

// fileBoundary is a line that begins the furniture introducing a new file,
// before that file's own `+++` header has been seen: `diff --git`, `index `,
// or `--- `. Each of the three is unambiguous on sight — no hunk's content
// line can start with any of them, because every content line is prefixed
// with a space, a plus or a minus, and none of those three is any of them.
func fileBoundary(s string) bool {
	return strings.HasPrefix(s, "diff --git ") || strings.HasPrefix(s, "index ") || strings.HasPrefix(s, "--- ")
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
