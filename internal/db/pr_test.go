package db

// The pull requests opened for a task, and what became of each.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

func worked(t *testing.T, d *DB) {
	t.Helper()

	tick := clock()

	created := record.Event{At: tick(), Kind: record.TaskCreated, Text: "Pay the thing"}

	if err := d.Append("ACME-1", created); err != nil {
		t.Fatalf("append task.created: %v", err)
	}

	join := record.Event{
		At:   tick(),
		Kind: record.RepoJoined,
		Data: map[string]string{"path": "/src/acme", "repo": "acme"},
	}

	if err := d.Append("ACME-1", join); err != nil {
		t.Fatalf("append repo.joined: %v", err)
	}
}

// TestAnOpeningIsWrittenDownWhereItOpened. The row is the answer `pr show`
// reads, and it carries the repository because a task worked in three
// checkouts has three pull requests to tell apart.
func TestAnOpeningIsWrittenDownWhereItOpened(t *testing.T) {
	d := open(t)
	worked(t, d)

	if err := d.OpenedPR("ACME-1", "/src/acme", "https://github.test/acme/pull/1"); err != nil {
		t.Fatalf("open a pull request: %v", err)
	}

	prs, err := d.PullRequests("ACME-1")
	if err != nil {
		t.Fatalf("read the pull requests: %v", err)
	}

	if len(prs) != 1 {
		t.Fatalf("there are %d pull requests, want one", len(prs))
	}

	if prs[0].URL != "https://github.test/acme/pull/1" || prs[0].Repo != "acme" ||
		prs[0].State != PROpen {
		t.Errorf("the row is %+v, want the acme pull request still open", prs[0])
	}
}

// TestWhatBecameOfItIsMarkedWhereItHappened. Merging and closing mark the
// rows; opening again after a close is a second row, not a reopened one.
func TestWhatBecameOfItIsMarkedWhereItHappened(t *testing.T) {
	d := open(t)
	worked(t, d)

	if err := d.OpenedPR("ACME-1", "/src/acme", "https://github.test/acme/pull/1"); err != nil {
		t.Fatalf("open a pull request: %v", err)
	}

	if err := d.MarkPR("ACME-1", "/src/acme", PRMerged); err != nil {
		t.Fatalf("mark it merged: %v", err)
	}

	if err := d.OpenedPR("ACME-1", "/src/acme", "https://github.test/acme/pull/2"); err != nil {
		t.Fatalf("open a second pull request: %v", err)
	}

	prs, err := d.PullRequests("ACME-1")
	if err != nil {
		t.Fatalf("read the pull requests: %v", err)
	}

	if len(prs) != 2 {
		t.Fatalf("there are %d pull requests, want two", len(prs))
	}

	if prs[0].URL != "https://github.test/acme/pull/2" || prs[0].State != PROpen {
		t.Errorf("the newest row is %+v, want the second pull request still open", prs[0])
	}

	if prs[1].URL != "https://github.test/acme/pull/1" || prs[1].State != PRMerged {
		t.Errorf("the oldest row is %+v, want the first pull request merged", prs[1])
	}
}

// TestATaskNobodyOpenedForHasNone. No rows is an empty reading, not a
// refusal: most tasks have never been delivered.
func TestATaskNobodyOpenedForHasNone(t *testing.T) {
	d := open(t)
	worked(t, d)

	prs, err := d.PullRequests("ACME-1")
	if err != nil {
		t.Fatalf("read the pull requests: %v", err)
	}

	if len(prs) != 0 {
		t.Errorf("there are %d pull requests, want none", len(prs))
	}
}
