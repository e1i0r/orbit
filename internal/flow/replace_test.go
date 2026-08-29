package flow

// Putting a flow in place. save_test.go covers the rules a save obeys; this
// covers the one step it happens in, which is the difference between an edit
// that lands and a flow name that stops working.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAFlowIsReplacedInOneStep is the property os.WriteFile does not have.
//
// It truncates the file and then fills it, so a reader that arrives between
// those two gets a flow that is half there. A flow is read at the moment a
// task is about to run on it, and Resolve is deliberately hard about what it
// finds — a file that will not parse is reported rather than replaced by the
// built-in of the same name — so the cost of losing that race is a flow name
// that is in the list and refuses to run.
//
// The reader below never sees a torn file, and cannot: a rename either has
// happened or has not.
func TestAFlowIsReplacedInOneStep(t *testing.T) {
	src := flowsIn(t)

	short, long := aFlow("busy"), aFlow("busy")
	long.Description = strings.Repeat("a flow with a great deal to say for itself. ", 500)

	if _, err := Save(src, short); err != nil {
		t.Fatalf("Save: %v", err)
	}

	path := filepath.Join(src.FlowDir(), "busy.json")

	saving := make(chan struct{})

	go func() {
		defer close(saving)

		for i := range 200 {
			f := short
			if i%2 == 0 {
				f = long
			}

			if _, err := Save(src, f); err != nil {
				t.Errorf("Save: %v", err)
				return
			}
		}
	}()

	var (
		reads   int
		readErr error
	)

	for readErr == nil {
		select {
		case <-saving:
			if reads == 0 {
				t.Fatal("the saving was over before the flow was read once")
			}

			return
		default:
		}

		if _, err := Load(path); err != nil {
			readErr = err
		}

		reads++
	}

	<-saving
	t.Fatalf("after %d reads the flow could not be read while it was being saved: %v", reads, readErr)
}

// TestSavingLeavesNothingBehind. The file it is written beside is this
// package's own business, and a flow directory that fills up with them is a
// listing full of names nobody chose.
func TestSavingLeavesNothingBehind(t *testing.T) {
	src := flowsIn(t)
	for range 3 {
		if _, err := Save(src, aFlow("tidy")); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	entries, err := os.ReadDir(src.FlowDir())
	if err != nil {
		t.Fatalf("read the flow directory: %v", err)
	}

	if len(entries) != 1 || entries[0].Name() != "tidy.json" {
		var got []string
		for _, e := range entries {
			got = append(got, e.Name())
		}

		t.Errorf("the flow directory holds %v, want tidy.json on its own", got)
	}
}
