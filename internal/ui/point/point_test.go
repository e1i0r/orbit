package point

// The one question this vocabulary answers on its own: whether two targets
// are the same thing.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/ui/layout"
	"github.com/e1i0r/orbit/internal/view"
)

// TestSameIgnoresWhichFieldWasPointedAt. A drag that starts on a task's title
// and ends on its cost is still a drag on that one task: the column is where
// inside the row the pointer was, and the row is what was pressed.
func TestSameIgnoresWhichFieldWasPointedAt(t *testing.T) {
	title := Target{Kind: Task, ID: "ACME-1", Band: view.Running, Column: layout.ColumnTitle}
	cost := Target{Kind: Task, ID: "ACME-1", Band: view.Running, Column: layout.ColumnElapsed}

	if !title.Same(cost) {
		t.Error("two points inside one row came back as different targets")
	}

	for _, other := range []Target{
		{Kind: Task, ID: "ACME-2", Band: view.Running},
		{Kind: Repo, ID: "ACME-1", Band: view.Running},
		{Kind: Task, ID: "ACME-1", Band: view.Done},
		{},
	} {
		if title.Same(other) {
			t.Errorf("%+v came back as the same target as the task's title row", other)
		}
	}
}
