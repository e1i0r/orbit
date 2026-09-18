package store

// The pull requests of a task, through the store's own doors.
//
// All three read zero. The command line marks a pull request through these
// and never touches the database directly — the handle is the store's to
// hold, and a second opener in one process would be two writers contending
// for one lock — so a door that quietly reached the wrong way would show up
// as the board stopping rather than as anything about a pull request.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// delivered is a task worked in one checkout, which is what a pull request
// needs behind it: the task and the repository are rows by the time one is
// opened, because a task is worked before it is delivered.
func delivered(t *testing.T, s *Store, id, repoAbs string) {
	t.Helper()

	d, err := s.Record()
	if err != nil {
		t.Fatalf("open the record: %v", err)
	}

	if err := d.Append(id, record.Event{Kind: record.TaskCreated, Text: "pay the thing"}); err != nil {
		t.Fatalf("write the task down: %v", err)
	}

	joined := record.Event{
		Kind: record.RepoJoined,
		Data: map[string]string{"path": repoAbs, "repo": "acme"},
	}

	if err := d.Append(id, joined); err != nil {
		t.Fatalf("join the checkout: %v", err)
	}
}

// TestAPullRequestIsWrittenDownAndReadBackThroughTheStore.
func TestAPullRequestIsWrittenDownAndReadBackThroughTheStore(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer closed(t, s)

	delivered(t, s, "ACME-1", "/src/acme")

	if err := s.OpenedPR("ACME-1", "/src/acme", "https://github.test/acme/pull/1"); err != nil {
		t.Fatalf("open a pull request: %v", err)
	}

	prs, err := s.PullRequests("ACME-1")
	if err != nil {
		t.Fatalf("read the pull requests: %v", err)
	}

	if len(prs) != 1 {
		t.Fatalf("the task has %d pull requests, want one", len(prs))
	}

	if prs[0].URL != "https://github.test/acme/pull/1" {
		t.Errorf("it came back as %+v, want the opening that was written", prs[0])
	}
}

// TestMarkingIsPerRepositoryAndPerOpening. Merging and closing are per
// repository, and an opening already answered is history: a task delivered
// twice has a row per opening, and restating the first one is a merge the
// record says never happened.
func TestMarkingIsPerRepositoryAndPerOpening(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer closed(t, s)

	delivered(t, s, "ACME-1", "/src/acme")

	if err := s.OpenedPR("ACME-1", "/src/acme", "https://github.test/acme/pull/1"); err != nil {
		t.Fatalf("open the first: %v", err)
	}

	if err := s.MarkPR("ACME-1", "/src/acme", PRMerged); err != nil {
		t.Fatalf("merge the first: %v", err)
	}

	if err := s.OpenedPR("ACME-1", "/src/acme", "https://github.test/acme/pull/2"); err != nil {
		t.Fatalf("open the second: %v", err)
	}

	if err := s.MarkPR("ACME-1", "/src/acme", PRClosed); err != nil {
		t.Fatalf("close the second: %v", err)
	}

	prs, err := s.PullRequests("ACME-1")
	if err != nil {
		t.Fatalf("read the pull requests: %v", err)
	}

	if len(prs) != 2 {
		t.Fatalf("the task has %d pull requests, want two", len(prs))
	}

	if prs[0].State != PRClosed || prs[1].State != PRMerged {
		t.Errorf("they read back as %q and %q, want closed and the merged it was",
			prs[0].State, prs[1].State)
	}
}

// TestATaskWithNoOpeningHasNone. No rows is an empty reading and not a
// refusal: most tasks have never been delivered.
func TestATaskWithNoOpeningHasNone(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer closed(t, s)

	prs, err := s.PullRequests("ACME-404")
	if err != nil {
		t.Fatalf("read the pull requests of a task nobody delivered: %v", err)
	}

	if len(prs) != 0 {
		t.Errorf("it has %d pull requests, want none", len(prs))
	}
}
