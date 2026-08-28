package layout

// The drop order is the decision this file exists to hold. The program this
// replaces dropped the two entries that reached everything — its menu and its
// command line — and kept a reminder about the arrow keys, which is what a
// drop order nobody wrote down produces. So the order here is a table, and
// the widths at which each column goes are numbers a test can fail on.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// stateBudget is what the catalogue declares for the state word in this
// test. It is deliberately not defaultStateCells: a test that used the same
// number could not tell a plan that read the budget from one that ignored it.
const stateBudget = 12

func budget(key string) int {
	if key == StateKey {
		return stateBudget
	}

	return 0
}

// board is the fixture the thresholds below are computed from: two
// repositories, ids of nine cells, one model of four. Every number in the
// tables is derived from it, so changing it changes them.
func board() []view.Task {
	return []view.Task{
		{ID: "ACME-2705", Repo: "app", Model: "opus", Title: "Reconciliation endpoint"},
		{ID: "ACME-2706", Repo: "payments", Model: "opus", Title: "Index on settlements"},
	}
}

// oneRepo is the same board with everything in a single repository.
func oneRepo() []view.Task {
	return []view.Task{
		{ID: "ACME-2705", Repo: "app", Model: "opus", Title: "Reconciliation endpoint"},
		{ID: "ACME-2706", Repo: "app", Model: "opus", Title: "Index on settlements"},
	}
}

// longIDs is a board whose ids eat the row. It is what makes Fallback
// reachable at a width the window actually accepts: the id never drops, so a
// board named this way is the one that cannot afford an aligned row.
func longIDs() []view.Task {
	return []view.Task{
		{ID: "2026-08-23-reconciliation-endpoint", Repo: "app", Title: "Reconciliation endpoint"},
		{ID: "2026-08-23-index-on-settlements", Repo: "payments", Title: "Index on settlements"},
	}
}

// TestTheDropOrderFiresInTheStatedSequence is the whole point of the file.
// Read down the table: the model goes first, then the title stops absorbing
// and starts shrinking, then the repository column goes, and only then does
// the row give up on alignment altogether.
func TestTheDropOrderFiresInTheStatedSequence(t *testing.T) {
	cases := []struct {
		name string
		w    int
		want Plan
	}{
		{"a wide terminal gives every column and the title the slack", 200, Plan{Repo: 8, ID: 9, Title: 150, State: 12, Model: 4, Elapsed: 7}},
		{"the last width that fits the model", 70, Plan{Repo: 8, ID: 9, Title: 20, State: 12, Model: 4, Elapsed: 7}},
		{"one column narrower and the model is gone", 69, Plan{Repo: 8, ID: 9, Title: 25, State: 12, Model: 0, Elapsed: 7}},
		{"the last width that fits the repository", 64, Plan{Repo: 8, ID: 9, Title: 20, State: 12, Model: 0, Elapsed: 7}},
		{"one column narrower and the repository is gone", 63, Plan{Repo: 0, ID: 9, Title: 29, State: 12, Model: 0, Elapsed: 7}},
		{"the last width that keeps a readable title", 54, Plan{Repo: 0, ID: 9, Title: 20, State: 12, Model: 0, Elapsed: 7}},
		{"one column narrower and the row is id, state and elapsed", 53, Plan{Repo: 0, ID: 9, Title: 0, State: 12, Model: 0, Elapsed: 7, Fallback: true}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Columns(c.w, board(), budget)
			if got != c.want {
				t.Errorf("Columns(%d) = %+v, want %+v", c.w, got, c.want)
			}
		})
	}
}

// TestTheRepositoryColumnIsAbsentForOneRepository is the column rule the
// program this replaces could not have: repository was bolted on to a
// one-repository tool, so the column was always there and always said the
// same thing. A column with one value in it is not information.
func TestTheRepositoryColumnIsAbsentForOneRepository(t *testing.T) {
	one := Columns(200, oneRepo(), budget)
	if one.Repo != 0 {
		t.Errorf("Columns over one repository gave the repo column %d cells, want none", one.Repo)
	}

	two := Columns(200, board(), budget)
	if two.Repo == 0 {
		t.Error("Columns over two repositories dropped the repo column — which repository a task is in is the whole point of the board")
	}

	if one.Title <= two.Title {
		t.Errorf("the title got %d cells with the repo column gone and %d with it there — the title absorbs what a dropped column leaves", one.Title, two.Title)
	}

	if none := Columns(200, nil, budget); none.Repo != 0 {
		t.Errorf("Columns over an empty board gave the repo column %d cells, want none", none.Repo)
	}
}

// TestThePlanNeverExceedsTheWidthItWasGiven sweeps every width from wider
// than any terminal down to nothing. A plan that overruns its width is a row
// that wraps, and one wrapped row moves every row under it.
func TestThePlanNeverExceedsTheWidthItWasGiven(t *testing.T) {
	boards := map[string][]view.Task{"two repositories": board(), "one repository": oneRepo(), "long ids": longIDs(), "empty": nil}
	for name, tasks := range boards {
		for w := 300; w >= 0; w-- {
			p := Columns(w, tasks, budget)
			if p.Width() > w {
				t.Fatalf("%s at width %d: the plan %+v is %d cells wide", name, w, p, p.Width())
			}

			for _, col := range []struct {
				name  string
				cells int
			}{{"repo", p.Repo}, {"id", p.ID}, {"title", p.Title}, {"state", p.State}, {"model", p.Model}, {"elapsed", p.Elapsed}} {
				if col.cells < 0 {
					t.Fatalf("%s at width %d: the %s column is %d cells", name, w, col.name, col.cells)
				}
			}

			if w < MinWidth {
				continue
			}
			// An empty board is exempt from the id: the column is as
			// wide as the widest id on it, and there is no id to be as
			// wide as. Everything else still holds, because the row a
			// board of no tasks does not draw is still a row that has to
			// fit.
			if p.Elapsed < 1 {
				t.Errorf("%s at width %d: elapsed is %d — it never drops, because it is the only number on the row", name, w, p.Elapsed)
			}

			if len(tasks) > 0 && p.ID < 1 {
				t.Errorf("%s at width %d: id is %d — it never drops, because it is how a reader says which task", name, w, p.ID)
			}

			if p.State != stateBudget {
				t.Errorf("%s at width %d: the state word got %d cells against a declared budget of %d", name, w, p.State, stateBudget)
			}
		}
	}
}

// TestFallbackIsSetExactlyWhenTheAlignedRowCannotBeAfforded checks both
// halves of the claim: it is set at the width below the boundary and not at
// the boundary, it never comes back on once the terminal grows, and when it
// is set the row really is only id, state and elapsed.
func TestFallbackIsSetExactlyWhenTheAlignedRowCannotBeAfforded(t *testing.T) {
	if p := Columns(54, board(), budget); p.Fallback {
		t.Errorf("Columns(54) fell back with %+v — an aligned row still fits here", p)
	}

	if p := Columns(53, board(), budget); !p.Fallback {
		t.Errorf("Columns(53) = %+v, want Fallback — the state word cannot have its budget beside a readable title", p)
	}
	// The width the window itself accepts, on a board whose ids are long
	// enough to eat it. This is the case Fallback exists for.
	narrow := Columns(MinWidth, longIDs(), budget)
	if !narrow.Fallback {
		t.Errorf("Columns(%d) over long ids = %+v, want Fallback", MinWidth, narrow)
	}

	if narrow.State != stateBudget {
		t.Errorf("the fallback row gave the state word %d cells against a budget of %d — the state word is what the fallback protects", narrow.State, stateBudget)
	}

	if narrow.Repo != 0 || narrow.Title != 0 || narrow.Model != 0 {
		t.Errorf("the fallback row is %+v, want only id, state and elapsed", narrow)
	}
	// Monotone in the width: a wider terminal never falls back where a
	// narrower one did not.
	sawAligned := false

	for w := 0; w <= 300; w++ {
		if !Columns(w, board(), budget).Fallback {
			sawAligned = true
			continue
		}

		if sawAligned {
			t.Fatalf("Columns(%d) fell back at a width wider than one that did not — Fallback must be monotone in the width", w)
		}
	}
}

// TestAMissingBudgetLeavesTheStateWordWhole is what happens when the
// catalogue cannot be read. A budget of zero is not a column of zero: a
// catalogue that lost its numbers should narrow the title, never delete the
// word that says what the task is doing.
func TestAMissingBudgetLeavesTheStateWordWhole(t *testing.T) {
	p := Columns(200, board(), func(string) int { return 0 })
	if p.State != defaultStateCells {
		t.Errorf("with no declared budget the state word got %d cells, want %d", p.State, defaultStateCells)
	}
}
