package task

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
)

// TestATaskIsOrdinaryUntilSomebodyMarksIt. A protocol that applied to
// everything would be a permission prompt on every push, and the tenth one
// is approved without being read.
func TestATaskIsOrdinaryUntilSomebodyMarksIt(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-36", "an ordinary task", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if Critical(s, tk) {
		t.Error("a task nobody marked is critical")
	}

	if err := Mark(s, tk, true, "elio"); err != nil {
		t.Fatalf("Mark: %v", err)
	}

	if !Critical(s, tk) {
		t.Error("a task that was marked is not critical")
	}

	if err := Mark(s, tk, false, "elio"); err != nil {
		t.Fatalf("Mark off: %v", err)
	}

	if Critical(s, tk) {
		t.Error("a mark that was taken off is still on — a reader who marked a task by mistake would be stuck with it")
	}
}

// TestTheQuestionIsNotAskedWithoutABackupBehindIt. A reader approving
// something they cannot take back is the opposite of what this is for, so a
// snapshot with no backup after it is not a question anybody can answer.
func TestTheQuestionIsNotAskedWithoutABackupBehindIt(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-37", "half a question", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := emit(s, tk, record.Event{
		Kind: record.CriticalSnapshot,
		Text: "push it",
		Data: map[string]string{"action": "pr", "repo": r.Name, "ref": "abc123"},
	}); err != nil {
		t.Fatalf("emit: %v", err)
	}

	if _, waiting := Waiting(s, tk); waiting {
		t.Error("a snapshot with no backup behind it reads as a question to answer")
	}
}

// TestApprovalIsOfAPlanAndAPlanIsAboutACommit. Work that moved after the yes
// was given is work nobody said yes to.
func TestApprovalIsOfAPlanAndAPlanIsAboutACommit(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-38", "approve one commit", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	at := Action{Name: "pr", Repo: r.Name, Ref: "abc123", Plan: "push it"}
	if err := Answer(s, tk, at, true, "elio"); err != nil {
		t.Fatalf("Answer: %v", err)
	}

	if !Permitted(s, tk, at) {
		t.Error("the action that was approved is not permitted")
	}

	moved := at
	moved.Ref = "def456"

	if Permitted(s, tk, moved) {
		t.Error("work that moved after the yes is still permitted")
	}

	other := at
	other.Name = "merge"

	if Permitted(s, tk, other) {
		t.Error("a yes to a push permitted a merge")
	}
}

// TestSnapshotWritesTheStateAndTagsABackup, in that order and both before
// anybody is asked anything.
func TestSnapshotWritesTheStateAndTagsABackup(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-39", "back it up first", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	wt, err := Join(s, tk, r)
	if err != nil {
		t.Fatalf("Join: %v", err)
	}

	a, err := Snapshot(s, tk, r, wt, Action{Name: "pr", Plan: "push it"})
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	if a.Ref == "" || a.Revert == "" {
		t.Errorf("the action carries %+v, want the commit it stands at and the way back", a)
	}

	kinds := kindsOf(mustEvents(t, s, tk))
	if count(kinds, record.CriticalSnapshot) != 1 || count(kinds, record.CriticalBackup) != 1 {
		t.Errorf("the record holds %v, want one snapshot and one backup", kinds)
	}

	waiting, ok := Waiting(s, tk)
	if !ok {
		t.Fatal("the task is not waiting on anybody after a whole question was asked")
	}

	if waiting.Ref != a.Ref {
		t.Errorf("the task waits on %q, want the commit the snapshot took", waiting.Ref)
	}

	if !strings.Contains(Refused(a), a.Revert) {
		t.Errorf("the refusal does not say how to undo it:\n%s", Refused(a))
	}
}
