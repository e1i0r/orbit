package ui

import (
	tea "charm.land/bubbletea/v2"
)

func (m Model) taskRepoPath(taskID string) string {
	for _, t := range m.board.Tasks {
		if t.ID == taskID {
			if t.RepoPath != "" {
				return t.RepoPath
			}

			return t.Repo
		}
	}

	return "."
}

// deliverPR initiates the GitHub Pull Request creation for the viewed task.
func (m Model) deliverPR() (tea.Model, tea.Cmd) {
	p := m.opts.Words

	taskID := m.detail
	if taskID == "" {
		if r, ok := m.selected(); ok && !r.head {
			taskID = r.task.ID
		}
	}

	if taskID == "" {
		return m.say(p.T("deliver.no_task", "select a task to continue")), nil
	}

	path := m.taskRepoPath(taskID)
	m = m.say(p.T("deliver.creating_pr", "creating pull request for {id}...", about("id", taskID)))

	return m.runWatched(Command{Name: "pr"}, []string{"-repo", path, taskID})
}

// fixChecks runs tests and linters in the task worktree and automatically patches issues.
func (m Model) fixChecks() (tea.Model, tea.Cmd) {
	p := m.opts.Words

	taskID := m.detail
	if taskID == "" {
		if r, ok := m.selected(); ok && !r.head {
			taskID = r.task.ID
		}
	}

	if taskID == "" {
		return m.say(p.T("deliver.no_task", "select a task to continue")), nil
	}

	path := m.taskRepoPath(taskID)
	instruction := "Run the full test suite and linter (go test ./..., golangci-lint). Investigate any failures, fix the source code and tests, and ensure 100% green checks."
	m = m.say(p.T("deliver.fixing_checks", "running checks and fixing issues for {id}...", about("id", taskID)))

	return m.runWatched(Command{Name: "note"}, []string{"-repo", path, taskID, "--", instruction})
}

// addMoreTests generates unit tests, fuzz testing, and property invariants up to >=90% coverage.
func (m Model) addMoreTests() (tea.Model, tea.Cmd) {
	p := m.opts.Words

	taskID := m.detail
	if taskID == "" {
		if r, ok := m.selected(); ok && !r.head {
			taskID = r.task.ID
		}
	}

	if taskID == "" {
		return m.say(p.T("deliver.no_task", "select a task to continue")), nil
	}

	path := m.taskRepoPath(taskID)
	instruction := "Analyze package coverage and write comprehensive unit tests, native Go fuzz tests (testing.F), and boundary property tests to achieve >=90% test coverage."
	m = m.say(p.T("deliver.adding_tests", "generating unit and fuzz tests for {id}...", about("id", taskID)))

	return m.runWatched(Command{Name: "note"}, []string{"-repo", path, taskID, "--", instruction})
}

// resolveComments brings back what reviewers asked for on the pull request,
// so the next run answers it.
//
// It reads and records, exactly as the verb does: the comments land in the
// task and the reader decides whether to spend a run on them. A key that
// silently started one would make "what did they say?" cost money.
func (m Model) resolveComments() (tea.Model, tea.Cmd) {
	p := m.opts.Words

	taskID := m.detail
	if taskID == "" {
		if r, ok := m.selected(); ok && !r.head {
			taskID = r.task.ID
		}
	}

	if taskID == "" {
		return m.say(p.T("deliver.no_task", "select a task to continue")), nil
	}

	path := m.taskRepoPath(taskID)
	m = m.say(p.T("deliver.resolving", "reading the reviews on {id}...", about("id", taskID)))

	return m.runWatched(Command{Name: "resolve"}, []string{"-repo", path, taskID})
}

// updatePRBranch commits any pending worktree modifications and pushes them to update the PR.
func (m Model) updatePRBranch() (tea.Model, tea.Cmd) {
	p := m.opts.Words

	taskID := m.detail
	if taskID == "" {
		if r, ok := m.selected(); ok && !r.head {
			taskID = r.task.ID
		}
	}

	if taskID == "" {
		return m.say(p.T("deliver.no_task", "select a task to continue")), nil
	}

	path := m.taskRepoPath(taskID)
	m = m.say(p.T("deliver.updating_pr", "updating branch and PR for {id}...", about("id", taskID)))

	return m.runWatched(Command{Name: "pr"}, []string{"-repo", path, taskID})
}

// mergePR merges the GitHub Pull Request and cleans up the remote branch.
func (m Model) mergePR() (tea.Model, tea.Cmd) {
	p := m.opts.Words

	taskID := m.detail
	if taskID == "" {
		if r, ok := m.selected(); ok && !r.head {
			taskID = r.task.ID
		}
	}

	if taskID == "" {
		return m.say(p.T("deliver.no_task", "select a task to continue")), nil
	}

	path := m.taskRepoPath(taskID)
	m = m.say(p.T("deliver.merging_pr", "merging pull request for {id}...", about("id", taskID)))

	return m.runWatched(Command{Name: "merge"}, []string{"-repo", path, taskID})
}

// closePR closes the GitHub Pull Request for the viewed task.
func (m Model) closePR() (tea.Model, tea.Cmd) {
	p := m.opts.Words

	taskID := m.detail
	if taskID == "" {
		if r, ok := m.selected(); ok && !r.head {
			taskID = r.task.ID
		}
	}

	if taskID == "" {
		return m.say(p.T("deliver.no_task", "select a task to continue")), nil
	}

	path := m.taskRepoPath(taskID)
	m = m.say(p.T("deliver.closing_pr", "closing pull request for {id}...", about("id", taskID)))

	return m.runWatched(Command{Name: "close-pr"}, []string{"-repo", path, taskID})
}
