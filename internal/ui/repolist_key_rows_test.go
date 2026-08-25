package ui

// repolist_coverage_test.go is the repository picker: collecting the list
// from RepoList when the reader gave one and from the tasks when they did
// not, and every key the screen answers to.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/board"
)

func TestCollectReposFromRepoListAndFromTasks(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. A board with an explicit RepoList: names and paths come from it.
	m.board.RepoList = []board.RepoInfo{{Name: "payments", Path: "/r/payments"}, {Name: "app", Path: "/r/app"}}
	repos := m.collectRepos()
	if len(repos) != 2 {
		t.Fatalf("collectRepos with a RepoList = %d entries, want 2", len(repos))
	}
	var total int
	for _, r := range repos {
		total += r.total
	}
	if total == 0 {
		t.Error("collectRepos did not count any tasks against the listed repositories")
	}

	// 2. No RepoList at all: repositories are derived from the tasks
	// themselves, one entry per distinct name.
	m.board.RepoList = nil
	repos = m.collectRepos()
	if len(repos) == 0 {
		t.Fatal("collectRepos with no RepoList found nothing")
	}
	seen := map[string]bool{}
	for _, r := range repos {
		if seen[strings.ToLower(r.name)] {
			t.Errorf("collectRepos listed %q twice", r.name)
		}
		seen[strings.ToLower(r.name)] = true
	}
}

func TestRepolistKeyNavigationAndFilter(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openRepos()
	repos := m.collectRepos()
	if len(repos) < 2 {
		t.Fatalf("fixture board has %d repositories, want at least 2 to test wrapping", len(repos))
	}

	// 1. Up from row 0 wraps to the last row.
	next, _ := m.repolistKey(tea.KeyPressMsg{Code: tea.KeyUp})
	got := asModel(t, next)
	if got.repolist.sel != len(repos)-1 {
		t.Errorf("up from row 0 = %d, want %d", got.repolist.sel, len(repos)-1)
	}

	// 2. Down from the last row wraps to 0.
	next, _ = got.repolistKey(tea.KeyPressMsg{Code: tea.KeyDown})
	got = asModel(t, next)
	if got.repolist.sel != 0 {
		t.Errorf("down from the last row = %d, want 0", got.repolist.sel)
	}

	// 3. Open filters to the selected repository; pressing it again on the
	// same one clears the filter instead.
	next, _ = got.repolistKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	got = asModel(t, next)
	if got.screen != screenList || got.repoFilter != repos[0].name {
		t.Errorf("open on a repo left screen=%v repoFilter=%q, want screenList/%q",
			got.screen, got.repoFilter, repos[0].name)
	}
	got = got.openRepos()
	next, _ = got.repolistKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	got = asModel(t, next)
	if got.repoFilter != "" {
		t.Errorf("opening the already-filtered repo again left repoFilter=%q, want cleared", got.repoFilter)
	}

	// 4. A space bar does the same as Open.
	got = got.openRepos()
	next, _ = got.repolistKey(tea.KeyPressMsg{Code: ' ', Text: " "})
	got = asModel(t, next)
	if got.repoFilter == "" {
		t.Error("space on a repo row did not filter to it")
	}

	// 5. Back and Quit both abandon the screen.
	got = got.openRepos()
	next, _ = got.repolistKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	got = asModel(t, next)
	if got.screen != screenList {
		t.Errorf("esc from repos left screen %v, want screenList", got.screen)
	}
}

func TestRepolistKeyWithNoRepositories(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.board = board.Board{}
	m = m.openRepos()

	// With nothing to list, only back/quit do anything.
	next, _ := m.repolistKey(tea.KeyPressMsg{Code: tea.KeyDown})
	got := asModel(t, next)
	if got.screen != screenRepos {
		t.Error("down with no repositories moved off the repos screen")
	}
	next, _ = got.repolistKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	got = asModel(t, next)
	if got.screen != screenList {
		t.Errorf("esc with no repositories left screen %v, want screenList", got.screen)
	}
}

func TestRepolistRowsMarksSelectionAndFilter(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m = m.openRepos()
	repos := m.collectRepos()
	m.repolist.sel = 0
	m.repoFilter = repos[0].name

	rows := m.repolistRows(20, 100)
	joined := strings.Join(rows, "\n")
	if !strings.Contains(joined, "[filtered]") {
		t.Errorf("repolistRows with a repo filtered = %q, want the [filtered] tag", joined)
	}

	// An empty board draws the "no repositories" sentence instead of rows.
	m.board = board.Board{}
	m.repoFilter = ""
	rows = m.repolistRows(20, 100)
	joined = strings.Join(rows, "\n")
	if !strings.Contains(joined, "no repositories") {
		t.Errorf("repolistRows with no repositories = %q, want the empty sentence", joined)
	}
}
