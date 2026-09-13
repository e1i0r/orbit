package migrate

// What one pass moved, in one sentence.

import (
	"strings"
	"testing"
)

// TestAResultSaysWhatItMoved. Events, tasks and turns in one sentence,
// for the log line the migration writes.
func TestAResultSaysWhatItMoved(t *testing.T) {
	got := Result{Tasks: 2, Events: 3, Messages: 1}.String()

	for _, want := range []string{"2", "3", "1"} {
		if !strings.Contains(got, want) {
			t.Errorf("a result reads %q", got)
		}
	}
}
