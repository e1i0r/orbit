package mcp

// Telling two checkouts of the same name apart, which is the one thing a
// name cannot do.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/view"
)

// TestAPathHintAnswersForThatCheckoutAlone. A caller filtering by the exact
// path it read off orbit_list_repos means that checkout: answered by the
// name the path resolves to, the other payments matched just as well.
func TestAPathHintAnswersForThatCheckoutAlone(t *testing.T) {
	b := board.Board{RepoList: []board.RepoInfo{
		{Name: "payments", Path: "/a/payments"},
		{Name: "payments", Path: "/b/payments"},
	}}

	mine := view.Task{
		ID: "PAY-1", Repo: "payments", RepoPath: "/a/payments",
		Repos: []string{"payments"}, RepoPaths: []string{"/a/payments"},
	}

	if !sameRepo(b, mine, "/a/payments") {
		t.Error("the task's own checkout did not answer to its path")
	}

	if sameRepo(b, mine, "/b/payments") {
		t.Error("a task in /a/payments answered for the payments in /b")
	}
}

// TestANameStillAnswers, because a model has been shown both and the rows
// carry names.
func TestANameStillAnswers(t *testing.T) {
	b := board.Board{RepoList: []board.RepoInfo{{Name: "ledger", Path: "/a/ledger"}}}

	t1 := view.Task{
		ID: "LED-1", Repo: "ledger", RepoPath: "/a/ledger",
		Repos: []string{"ledger"}, RepoPaths: []string{"/a/ledger"},
	}

	if !sameRepo(b, t1, "ledger") {
		t.Error("the name the row draws did not answer")
	}
}
