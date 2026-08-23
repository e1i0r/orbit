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
		// The one failure with words of this window's own is the bound
		// being hit, and those words are said here, in the reader's
		// language: errGitTimedOut is carried as a sentinel to be
		// recognised rather than as a sentence to be passed on, because an
		// errors.New in gitdiff.go cannot be translated and the pseudolocale
		// golden cannot see it. Every other failure is git's own output,
		// which is not this program's to translate, and is shown as it came.
		said := m.diffErr.Error()
		if errors.Is(m.diffErr, errGitTimedOut) {
			said = p.T("diff.timed_out", "git did not answer in time")
		}
		return []string{" " + Paint(Bad).Render(said)}
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
//
// It reads one line knowing nothing of the ones above it, which fileAt
// cannot afford and this can: a removed `-- comment` is dimmed rather than
// reddened, which costs a colour and not a file.
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
// It reads the diff forward from the top rather than upward from the
// cursor, because what a line of a diff means is not in the line: it is in
// what the lines above it opened. `--- a/x` is a file's old-side header when
// it stands in the furniture between two files, and the very same characters
// are an ordinary removed line — `-- add the index`, the comment syntax of
// SQL, Lua, Haskell and Ada, carrying the minus every removed line carries —
// when a hunk is open. `+++ ` leads the same double life on the added side:
// `++ xs` is Haskell, rendered `+++ xs`. Deciding from the prefix alone,
// which a walk upward from the cursor has to, reads a deleted comment as the
// start of the next file and takes every line under it in the hunk with it.
//
// So the scan carries one bit — is a hunk open — and exactly two lines can
// change it: `@@`, which opens one, and `diff --git `, which begins the next
// file's furniture. Neither can be forged from inside a hunk, because every
// line inside one is rendered with a space, a plus, a minus or a backslash
// in front of it, and neither prefix begins with any of those four. Outside
// a hunk there is no content to be confused with furniture. The distinction
// is structural, not a claim about what source text can look like.
//
// The line is counted rather than parsed, because a hunk header only says
// where the hunk begins. A removed line is not in the new file and does not
// advance the count; a context line and an added line both do. Getting this
// wrong opens the right file at the wrong line, which is worse than not
// opening it at all — a reader believes what the editor shows them.
//
// A line that is not inside a hunk still answers. Sitting on `diff --git` or
// on `+++ b/x` and pressing the key means "open this file", and refusing
// because there is no line number to be precise about would refuse the only
// thing the reader asked for; line 1 is the honest answer there, and
// fileBelow is what finds the file that furniture is introducing.
//
// A file git is deleting is where that stops: its new side is `/dev/null`,
// so there is nothing in the worktree to open and no line to open it at.
// Answering with whichever file comes next — what furniture with no header
// of its own otherwise falls through to — is the one answer this function
// must never give, so it refuses for the whole of that file, its furniture
// and its removed lines alike.
func fileAt(lines []string, at int) (string, int, bool) {
	if at < 0 || at >= len(lines) {
		return "", 0, false
	}
	var (
		file   string // the file whose `+++` header the scan last passed
		line   int    // the line of that file the scan is standing on
		inHunk bool   // a `@@` has opened a hunk and nothing has closed it
		gone   bool   // this file's new side is /dev/null: git is deleting it
	)
	for i := 0; i <= at; i++ {
		s := lines[i]
		if strings.HasPrefix(s, "diff --git ") {
			// The next file's furniture begins here, and nothing above it
			// says anything about anything below it.
			file, line, inHunk, gone = "", 0, false, false
			continue
		}
		if start, ok := hunkStart(s); ok {
			inHunk, line = true, start
			continue
		}
		if !inHunk {
			// Furniture. The only part of it that matters here is the new
			// side's header: the path it names, or the fact that git is
			// naming /dev/null because the file is being deleted.
			if name, ok := fileHeader(s); ok {
				file, line, gone = name, 1, false
			} else if strings.HasPrefix(s, "+++ ") {
				gone = true
			}
			continue
		}
		// Content. The line the cursor is on is the one being asked about,
		// so the count stops under it rather than passing over it.
		if i < at && !strings.HasPrefix(s, "-") && !strings.HasPrefix(s, `\`) {
			line++
		}
	}
	if gone {
		return "", 0, false
	}
	if file == "" {
		// Furniture the scan met before its own file's header: whichever
		// file's `+++` comes next is the file it is introducing.
		return fileBelow(lines, at)
	}
	return file, line, true
}

// fileBelow is the file introduced by the next `+++` header at or after the
// given line. It is what a cursor sitting in furniture that has not reached
// its file's header yet answers with, and what fileAt hands off to once it
// knows there is no file above the cursor to ask about.
//
// It stops on two things rather than reading to the end of the diff. A `@@`
// means the furniture is behind it and every `+++` after it is somebody's
// content — the same double life fileAt tracks a hunk for. And
// `+++ /dev/null` is git deleting the file whose furniture the cursor is in:
// reading past it would answer with whatever file happens to come next, at a
// line the reader never asked about, which is the one answer this pair must
// never give.
func fileBelow(lines []string, at int) (string, int, bool) {
	for i := at; i < len(lines); i++ {
		if _, ok := hunkStart(lines[i]); ok {
			return "", 0, false
		}
		if name, ok := fileHeader(lines[i]); ok {
			return name, 1, true
		}
		if strings.HasPrefix(lines[i], "+++ ") {
			return "", 0, false
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
