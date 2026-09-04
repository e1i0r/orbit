package ui

import (
	tea "charm.land/bubbletea/v2"
)

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

// deliverPR initiates the GitHub Pull Request creation for the viewed task.
func (m Model) deliverPR() (tea.Model, tea.Cmd) {
	m, taskID, path, ok := m.aboutTask()
	if !ok {
		return m, nil
	}

	p := m.opts.Words
	m = m.say(p.T("deliver.creating_pr", "creating pull request for {id}...", about("id", taskID)))

	return m.runWatched(Command{Name: "pr"}, repoArgs(path, taskID))
}

// fixChecks asks the task to run its checks and fix what fails.
//
// It records the instruction, exactly as resolveComments records the
// reviews: what the reader pressed is not a run, and nothing here starts
// one. The sentence says so — it said "running checks and fixing issues"
// and then "note finished", which is two accounts of the same keystroke and
// neither of them what happened.
func (m Model) fixChecks() (tea.Model, tea.Cmd) {
	m, taskID, path, ok := m.aboutTask()
	if !ok {
		return m, nil
	}

	p := m.opts.Words
	instruction := "Run the full test suite and linter (go test ./..., golangci-lint). Investigate any failures, fix the source code and tests, and ensure 100% green checks."
	said := p.T("deliver.checks_asked", "{id} was asked to fix the failing checks; a run has to pick it up",
		about("id", taskID))

	return m.say(said).runWatchedSaying(Command{Name: "note"}, repoArgs(path, taskID, instruction), said)
}

// addMoreTests asks the task for unit tests, fuzz tests and property
// invariants up to >=90% coverage. Like fixChecks, it records the
// instruction and leaves the run to the reader.
func (m Model) addMoreTests() (tea.Model, tea.Cmd) {
	m, taskID, path, ok := m.aboutTask()
	if !ok {
		return m, nil
	}

	p := m.opts.Words
	instruction := "Analyze package coverage and write comprehensive unit tests, native Go fuzz tests (testing.F), and boundary property tests to achieve >=90% test coverage."
	said := p.T("deliver.tests_asked", "{id} was asked for more tests; a run has to pick it up",
		about("id", taskID))

	return m.say(said).runWatchedSaying(Command{Name: "note"}, repoArgs(path, taskID, instruction), said)
}

// resolveComments brings back what reviewers asked for on the pull request,
// so the next run answers it.
//
// It reads and records, exactly as the verb does: the comments land in the
// task and the reader decides whether to spend a run on them. A key that
// silently started one would make "what did they say?" cost money.
func (m Model) resolveComments() (tea.Model, tea.Cmd) {
	m, taskID, path, ok := m.aboutTask()
	if !ok {
		return m, nil
	}

	p := m.opts.Words
	m = m.say(p.T("deliver.resolving", "reading the reviews on {id}...", about("id", taskID)))

	return m.runWatched(Command{Name: "resolve"}, repoArgs(path, taskID))
}

// updatePRBranch commits any pending worktree modifications and pushes them to update the PR.
func (m Model) updatePRBranch() (tea.Model, tea.Cmd) {
	m, taskID, path, ok := m.aboutTask()
	if !ok {
		return m, nil
	}

	p := m.opts.Words
	m = m.say(p.T("deliver.updating_pr", "updating branch and PR for {id}...", about("id", taskID)))

	return m.runWatched(Command{Name: "pr"}, repoArgs(path, taskID))
}

// mergePR merges the GitHub Pull Request and cleans up the remote branch.
func (m Model) mergePR() (tea.Model, tea.Cmd) {
	m, taskID, path, ok := m.aboutTask()
	if !ok {
		return m, nil
	}

	p := m.opts.Words
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
	m = m.say(p.T("deliver.closing_pr", "closing pull request for {id}...", about("id", taskID)))

	return m.runWatched(Command{Name: "close-pr"}, repoArgs(path, taskID))
}
