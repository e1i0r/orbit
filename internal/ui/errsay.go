package ui

import (
	"errors"
	"strings"
)

// The refusals below are values rather than sentences built where they are
// raised, because every one of them is drawn in the reader's language and
// none of them is raised anywhere a printer can be reached: refresh,
// rescan, diffOf, logOf, control, start, markRead, takeSession and
// runCommand are free functions handed to a tea.Cmd, and threading a
// *words.Printer through the nine of them to reach eleven strings would put
// a translation argument on some forty call sites, most of them tests.
//
// A value costs nothing to compare, and errSaid compares it at the one
// place the text is finally read.
var (
	errReadBoard      = errors.New("read the board")
	errFindRepos      = errors.New("look for new repositories")
	errNoControlPort  = errors.New("this window was opened without a way to control a task")
	errNoStartPort    = errors.New("this window was opened without a way to start a run")
	errNoReadPort     = errors.New("this window was opened without a way to mark a task read")
	errNoRequeuePort  = errors.New("this window was opened without a way to put a task back in the queue")
	errNoSessionPort  = errors.New("this window was opened without a way to take the keyboard")
	errNoWorktreePort = errors.New("this window was opened without a way to find the worktree")
	errNoRecordPort   = errors.New("this window was opened without a way to read the record")
	errNoCommandPort  = errors.New("this window was opened without a way to run commands")
	errNoRepoPath     = errors.New("this task does not say where its repository is")
	errNoSupervisor   = errors.New("supervisor engine is not configured")
)

// errSaid is how one of them reaches a reader.
//
// Every refusal above is wrapped with its sentinel first, so what follows
// the sentence — the cause git or the filesystem gave, the command line
// that timed out, the name of the command that never ran — survives the
// sentence being translated. That tail is left exactly as it arrived: a
// path, an exit status and git's own stderr are evidence, and evidence is
// not prose.
//
// An error this window did not raise is passed on in the words its author
// chose, because this package has no way to translate a sentence it has
// never seen.
func (m Model) errSaid(err error) string {
	if err == nil {
		return ""
	}

	said, sentinel := m.errSentence(err)
	if sentinel == nil {
		return err.Error()
	}

	return said + strings.TrimPrefix(err.Error(), sentinel.Error())
}

// errSentence says one refusal, and hands back the sentinel it matched so
// errSaid knows how much of the error text the sentence already covers.
//
// A switch rather than a table because a table is read with a variable for
// a key, and internal/words can only hold a translation to account when the
// key is written out where it is asked for.
func (m Model) errSentence(err error) (string, error) {
	p := m.opts.Words

	switch {
	case errors.Is(err, errReadBoard):
		return p.T("err.read_board", "read the board"), errReadBoard
	case errors.Is(err, errFindRepos):
		return p.T("err.find_repos", "look for new repositories"), errFindRepos
	case errors.Is(err, errNoControlPort):
		return p.T("err.no_control_port", "this window was opened without a way to control a task"), errNoControlPort
	case errors.Is(err, errNoStartPort):
		return p.T("err.no_start_port", "this window was opened without a way to start a run"), errNoStartPort
	case errors.Is(err, errNoReadPort):
		return p.T("err.no_read_port", "this window was opened without a way to mark a task read"), errNoReadPort
	case errors.Is(err, errNoRequeuePort):
		return p.T("err.no_requeue_port", "this window was opened without a way to put a task back in the queue"), errNoRequeuePort
	case errors.Is(err, errNoSessionPort):
		return p.T("err.no_session_port", "this window was opened without a way to take the keyboard"), errNoSessionPort
	case errors.Is(err, errNoWorktreePort):
		return p.T("err.no_worktree_port", "this window was opened without a way to find the worktree"), errNoWorktreePort
	case errors.Is(err, errNoRecordPort):
		return p.T("err.no_record_port", "this window was opened without a way to read the record"), errNoRecordPort
	case errors.Is(err, errNoCommandPort):
		return p.T("err.no_command_port", "this window was opened without a way to run commands"), errNoCommandPort
	case errors.Is(err, errNoRepoPath):
		return p.T("err.no_repo_path", "this task does not say where its repository is"), errNoRepoPath
	case errors.Is(err, errNoSupervisor):
		return p.T("err.no_supervisor", "supervisor engine is not configured"), errNoSupervisor
	case errors.Is(err, errGitTimedOut):
		return p.T("err.git_timeout", "git did not answer in time"), errGitTimedOut
	}

	return "", nil
}
