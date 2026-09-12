package web

// The impact route against something there is to look in.
//
// flow_test.go asks the two states that need no repository: a task nobody ran
// answers "missing", and the folding keeps the three claims apart. What is
// left unasked there is the reading itself — the part that leaves the process
// and runs git — and this is it.
//
// A real repository rather than a stand-in, because the reading is git's: a
// fake that answered a canned Impact would test the fake. What is faked is the
// port that says where the checkout is, which is the part Orbit owns.

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/view"
)

// TestTheImpactOfACheckoutThatIsThere.
//
// The reading only reaches the history when there are changes to weigh and a
// base branch to weigh them against. Without the base branch it asks git what
// is uncommitted, and a task that committed its work has nothing uncommitted —
// which is how the cockpit's pane came to say "this change touched no files"
// beside a diff of nineteen.
func TestTheImpactOfACheckoutThatIsThere(t *testing.T) {
	dir := aCheckout(t)

	code, body := ask(t, withCheckout(t, "LED-34", dir, dir), "/api/tasks/LED-34/impact")
	if code != http.StatusOK {
		t.Fatalf("the impact answered %d: %v", code, body)
	}

	if body["missing"] == true {
		t.Errorf("a checkout that is there was answered as missing: %v", body)
	}

	if failed, said := body["failed"].(string); said && failed != "" {
		t.Errorf("the reading failed: %s", failed)
	}

	if len(listIn(t, body, "changed")) == 0 {
		t.Errorf("the reading does not carry the file that changed: %v", body)
	}

	// The history behind it was read, which is the part that only happens
	// when the base branch came along.
	commits, ok := body["commits"].(float64)
	if !ok {
		t.Fatalf("the reading carried no count of commits: %v", body)
	}

	if commits == 0 {
		t.Errorf("the reading counted no commits, so the history was never read: %v", body)
	}
}

// TestARecordThatWillNotReadIsAFailure.
//
// Not the same claim as a checkout that is not there, and not the same as a
// repository that will not answer either. The record is Orbit's own, so a
// refusal here is a thing that broke; the history is somebody else's, so a
// refusal there is a reading that failed. Folding the two would leave a page
// unable to say which one it is looking at.
func TestARecordThatWillNotReadIsAFailure(t *testing.T) {
	dir := aCheckout(t)

	s := server(aBoard{
		tasks:  []view.Task{{ID: "LED-35", Title: "the record will not read", Band: view.Running, Repo: "ledger", RepoPath: dir}},
		logErr: errors.New("the record is locked by another process"),
	}, there{path: dir}, "/code", built)

	code, body := ask(t, s, "/api/tasks/LED-35/impact")
	if code != http.StatusInternalServerError {
		t.Fatalf("a record that would not read answered %d: %v", code, body)
	}

	if body == nil {
		t.Error("the failure said nothing about itself")
	}
}

// TestTheDeltaOfTheAttemptThatStands.
//
// The last, for the reason the report shows the last: a task run three times
// said this three times, and the two before it are about work that was thrown
// away. Answering with the first would put a claim about abandoned work
// beside the history's word on the work that stands.
func TestTheDeltaOfTheAttemptThatStands(t *testing.T) {
	got := impactOf("LED-36", repo.Impact{}, []view.Entry{
		{Kind: "task.delta", Delta: &view.Delta{Needs: []string{"the first attempt said this"}}},
		{Kind: "task.started"},
		{Kind: "task.delta", Delta: &view.Delta{Needs: []string{"a Decimal now"}}},
	})

	if got.Delta == nil {
		t.Fatal("a record that carried a delta came back without one")
	}

	if strings.Contains(strings.Join(got.Delta.Needs, " "), "first attempt") {
		t.Errorf("the answer carries the delta of an attempt that was thrown away: %+v", got.Delta)
	}

	if got.Delta.Needs[0] != "a Decimal now" {
		t.Errorf("the answer carries %v", got.Delta.Needs)
	}
}

// TestADeltaThatSaysNothingIsNotADelta.
//
// An engine that answered with an empty delta block made no claim, and
// drawing an empty section beside the history's word would look like a
// section that failed to load.
func TestADeltaThatSaysNothingIsNotADelta(t *testing.T) {
	got := impactOf("LED-37", repo.Impact{}, []view.Entry{
		{Kind: "task.delta", Delta: &view.Delta{}},
		{Kind: "task.delta"},
		{Kind: "phase.finished", Phase: "review"},
	})

	if got.Delta != nil {
		t.Errorf("an empty delta block was carried as a claim: %+v", got.Delta)
	}
}

// TestAReadingThatFoundNothingIsNotAMissingOne.
//
// Three states travel as three fields. A change that reaches nothing is a
// finding; a task nobody ran has nothing to find; and a repository that will
// not answer is a reading that failed. A page that folded them would tell a
// reader this repository has no coupling when what happened is that git said
// no.
func TestAReadingThatFoundNothingIsNotAMissingOne(t *testing.T) {
	got := impactOf("LED-38", repo.Impact{}, nil)

	if got.Missing {
		t.Error("a reading that found nothing said there was no checkout")
	}

	if got.Failed != "" {
		t.Errorf("a reading that found nothing said it had failed: %q", got.Failed)
	}

	if len(got.Coupled) != 0 || len(got.Contracts) != 0 {
		t.Errorf("a reading that found nothing carried findings: %+v", got)
	}
}
