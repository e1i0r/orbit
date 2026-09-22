package repo

// Which checks stand between a pull request and a merge.

import "testing"

// TestBlockingSeparatesRedFromStillRunning.
//
// Two different sentences for a reader: a red check is a decision to make
// and a pending one is only a wait, and telling them apart is the whole
// reason this reading is taken before `gh pr merge` refuses.
func TestBlockingSeparatesRedFromStillRunning(t *testing.T) {
	red, waiting := Blocking([]CheckRun{
		{Name: "build", Bucket: "pass"},
		{Name: "lint", Bucket: "fail"},
		{Name: "e2e", Bucket: "pending"},
		{Name: "deploy-preview", Bucket: "skipping"},
		{Name: "flaky", Bucket: "cancel"},
	})

	if len(red) != 1 || red[0] != "lint" {
		t.Errorf("red = %v, want just lint", red)
	}

	if len(waiting) != 1 || waiting[0] != "e2e" {
		t.Errorf("waiting = %v, want just e2e", waiting)
	}
}

// TestAPullRequestWithNothingRedMergesAsItAlwaysDid: a repository with no
// CI reports nothing, and refusing to merge there would be this reading
// inventing a rule GitHub does not have.
func TestAPullRequestWithNothingRedMergesAsItAlwaysDid(t *testing.T) {
	for _, runs := range [][]CheckRun{
		nil,
		{},
		{{Name: "build", Bucket: "pass"}},
		{{Name: "optional", Bucket: "skipping"}},
	} {
		red, waiting := Blocking(runs)
		if len(red) != 0 || len(waiting) != 0 {
			t.Errorf("Blocking(%v) = (%v, %v), want nothing in the way", runs, red, waiting)
		}
	}
}
