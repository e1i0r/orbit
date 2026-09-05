package ui

import (
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// deliverBySupervisor is what the record says was handed the work when the
// verb is one of the six the supervisor carries. The commands say their own
// name, which is what a reader would have typed to do it themselves.
const deliverBySupervisor = "supervisor"

// taskRepoPath is where the task's repository is on disk, and nothing at all
// when it has none.
//
// Nothing, and never ".". Orbit is opened on the directory above the
// repositories, so "." is the workspace: handing it to -repo makes the
// command answer that it is not a repository, which is a sentence about
// Orbit's own working directory in front of a reader who asked something
// about a task. A task written against no repository — which FRA-61 made a
// thing a reader can do — hit that on every verb.
//
// And never the repository's name either. A row carries the name to draw it;
// a name handed to -repo is a directory that is not there.
func (m Model) taskRepoPath(taskID string) string {
	for _, t := range m.board.Tasks {
		if t.ID == taskID {
			return t.RepoPath
		}
	}

	return ""
}

// taskCheckoutPath is where the task's branch is checked out: its worktree
// when one exists on disk, the repository path when the reader has no worktree
// port (in tests), or empty when there is no checkout to work in.
func (m Model) taskCheckoutPath(taskID string) string {
	t, ok := m.task(taskID)
	if !ok || t.RepoPath == "" {
		return ""
	}

	if m.opts.Reader == nil {
		return t.RepoPath
	}

	dir, err := m.opts.Reader.Worktree(t.RepoPath, t.ID)
	if err == nil && dir != "" {
		if info, statErr := os.Stat(dir); statErr == nil && info.IsDir() {
			return dir
		}
	}

	return ""
}

// repoArgs is a command's arguments with -repo in front of them, or without
// it when there is no repository to name.
//
// One helper rather than the same three lines at eight call sites: the eight
// are every verb the window can run about a task, and the one that forgets
// is the one a reader finds.
func repoArgs(path string, rest ...string) []string {
	if path == "" {
		return rest
	}

	return append([]string{"-repo", path}, rest...)
}

// deliverPending is the delivery verb this window is waiting on: the task it
// was pressed about, the caption it was offered under, and the command
// carrying it — empty where the supervisor is carrying it instead.
//
// One verb at a time, because the watch already runs one command at a time
// and the supervisor answers one ask at a time. A second field per carrier
// would be two ways to be waiting on the same thing.
type deliverPending struct {
	task view.Task
	verb string
	cmd  string
	// at is when it was asked for, for the line in the band that says how
	// long it has been: see waiting.go.
	at time.Time
}

// asked writes down that a delivery verb was pressed, and remembers it until
// it is answered. Nothing else on this screen leaves a trace on the task, so
// this is what a reader sees when they look at what the key did.
//
// A record that will not take the ask is said in the band and the verb still
// goes ahead: the work was asked for either way, and refusing to do it
// because the writing failed would be the window choosing its own bookkeeping
// over what the reader wanted.
func (m Model) asked(taskID, verb, by, cmd string) Model {
	t, ok := m.task(taskID)
	if !ok {
		return m
	}

	m.delivering = deliverPending{task: t, verb: verb, cmd: cmd, at: m.now}

	return m.deliver(t, Delivery{Verb: verb, By: by})
}

// answered closes the verb that was out, with what came back or why it broke.
func (m Model) answered(text string, err error) Model {
	out := m.delivering
	m.delivering = deliverPending{}

	if out.verb == "" {
		return m
	}

	return m.deliver(out.task, Delivery{Verb: out.verb, Text: text, Failure: err, Done: true})
}

// deliver hands one of those to the port, and says what stopped it.
func (m Model) deliver(t view.Task, d Delivery) Model {
	if m.opts.RecordDeliver == nil {
		return m
	}

	if err := m.opts.RecordDeliver(t, d); err != nil {
		return m.say(err.Error())
	}

	return m
}

// aboutTask is the task a verb on this screen is about — the one being read,
// or the row the cursor is on — and the repository it is checked out in.
//
// The refusal is said here rather than returned to be said seven times: a
// verb with no task in front of it has nothing to do, and every one of these
// answered that in the same sentence. What comes back is the model with the
// sentence already in it, so the caller returns it and stops.
func (m Model) aboutTask() (Model, string, string, bool) {
	id := m.taskInHand()
	if id == "" {
		return m.say(m.opts.Words.T("deliver.no_task", "select a task to continue")), "", "", false
	}

	return m, id, m.taskRepoPath(id), true
}

// askSupervisorTo hands one of these verbs to the supervisor: the window
// says what is wanted and where, and the supervisor finds out what that
// takes before doing it.
//
// What it is asked goes into the supervisor's own thread, exactly as a line
// typed on its screen does, so a keystroke and a sentence are one
// conversation in one order — and the answer comes back where the operator
// already reads its answers.
func (m Model) askSupervisorTo(caption, taskID, path, body, said string) (tea.Model, tea.Cmd) {
	p := m.opts.Words

	if path == "" {
		return m.say(p.T("deliver.no_checkout", "{id} has no checkout, so there is nothing to work in",
			about("id", taskID))), nil
	}

	next, cmd := m.sendSupervisorMessage(deliverPrompt(caption, taskID, path, body))
	if cmd == nil {
		// The thread refused the line. What it said about that is the
		// only true sentence there is here.
		return next, nil
	}

	return next.asked(taskID, caption, deliverBySupervisor, "").say(said), cmd
}

// deliverPR asks for the pull request to be opened, template and all.
func (m Model) deliverPR() (tea.Model, tea.Cmd) {
	m, taskID, _, ok := m.aboutTask()
	if !ok {
		return m, nil
	}

	return m.askSupervisorTo("CREATE PR", taskID, m.taskCheckoutPath(taskID), promptCreatePR,
		m.opts.Words.T("deliver.pr_asked", "the supervisor was asked to open the pull request for {id}",
			about("id", taskID)))
}

// fixChecks asks for the checks on the pull request to pass.
//
// The checks that matter are the ones GitHub ran on the branch, and what it
// takes to fix them is not known until they have been read: that is the
// whole reason this goes to the supervisor rather than to a command.
func (m Model) fixChecks() (tea.Model, tea.Cmd) {
	m, taskID, _, ok := m.aboutTask()
	if !ok {
		return m, nil
	}

	return m.askSupervisorTo("FIX CHECKS", taskID, m.taskCheckoutPath(taskID), promptFixChecks,
		m.opts.Words.T("deliver.checks_asked", "the supervisor was asked to make {id}'s checks pass",
			about("id", taskID)))
}

// addMoreTests asks for the tests this task's change is missing.
func (m Model) addMoreTests() (tea.Model, tea.Cmd) {
	m, taskID, _, ok := m.aboutTask()
	if !ok {
		return m, nil
	}

	return m.askSupervisorTo("MORE TESTS", taskID, m.taskCheckoutPath(taskID), promptMoreTests,
		m.opts.Words.T("deliver.tests_asked", "the supervisor was asked for more tests on {id}",
			about("id", taskID)))
}

// resolveComments answers the reviews on the pull request.
//
// It used to read the comments into the record and stop there, which left
// the reader with the reviews written down twice and nothing said back. The
// deciding is the work — apply this one, argue with that one — so it goes
// to the supervisor with the reviews still on the pull request, where the
// replies have to go anyway.
func (m Model) resolveComments() (tea.Model, tea.Cmd) {
	m, taskID, _, ok := m.aboutTask()
	if !ok {
		return m, nil
	}

	return m.askSupervisorTo(
		"RESOLVE COMMENTS", taskID, m.taskCheckoutPath(taskID), promptResolveComments,
		m.opts.Words.T("deliver.resolve_asked", "the supervisor was asked to answer the reviews on {id}",
			about("id", taskID)))
}

// reviewPR reads the pull request and says what it finds, on the pull
// request. It is the one verb here that changes nothing: a review is worth
// having precisely because the reader of it decides.
func (m Model) reviewPR() (tea.Model, tea.Cmd) {
	m, taskID, _, ok := m.aboutTask()
	if !ok {
		return m, nil
	}

	return m.askSupervisorTo("DEEP REVIEW", taskID, m.taskCheckoutPath(taskID), promptReview,
		m.opts.Words.T("deliver.review_asked", "the supervisor was asked to review {id}",
			about("id", taskID)))
}

// updatePRBranch brings the task's branch up to date with the branch it
// will be merged into: the base branch is brought up to the remote's, and
// then merged in here. It is not the same verb as creating the pull
// request, which is what this key used to run.
func (m Model) updatePRBranch() (tea.Model, tea.Cmd) {
	m, taskID, _, ok := m.aboutTask()
	if !ok {
		return m, nil
	}

	return m.askSupervisorTo("UPDATE PR", taskID, m.taskCheckoutPath(taskID), promptUpdatePR,
		m.opts.Words.T("deliver.update_asked",
			"the supervisor was asked to bring {id} up to date with its base branch",
			about("id", taskID)))
}

// mergePR merges the GitHub Pull Request and cleans up the remote branch.
func (m Model) mergePR() (tea.Model, tea.Cmd) {
	m, taskID, path, ok := m.aboutTask()
	if !ok {
		return m, nil
	}

	p := m.opts.Words
	m = m.asked(taskID, "MERGE PR", "merge", "merge")
	m = m.say(p.T("deliver.merging_pr", "merging pull request for {id}...", about("id", taskID)))

	return m.runWatched(Command{Name: "merge"}, repoArgs(path, taskID))
}

// closePR closes the GitHub Pull Request for the viewed task.
func (m Model) closePR() (tea.Model, tea.Cmd) {
	m, taskID, path, ok := m.aboutTask()
	if !ok {
		return m, nil
	}

	p := m.opts.Words
	m = m.asked(taskID, "CLOSE PR", "close-pr", "close-pr")
	m = m.say(p.T("deliver.closing_pr", "closing pull request for {id}...", about("id", taskID)))

	return m.runWatched(Command{Name: "close-pr"}, repoArgs(path, taskID))
}
