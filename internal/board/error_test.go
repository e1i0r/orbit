package board

// A failure names what it was reading: the task, the repository, and why.
// The band is what the window draws; this sentence is what the log keeps.

import (
	"errors"
	"strings"
	"testing"
)

// TestATaskErrorSaysWhatItWasReading.
func TestATaskErrorSaysWhatItWasReading(t *testing.T) {
	cause := errors.New("the file is gone")
	err := &TaskError{Repo: "acme", ID: "ACME-1", Err: cause}

	if got := err.Error(); !strings.Contains(got, "ACME-1") || !strings.Contains(got, "acme") {
		t.Errorf("a task error reads %q", got)
	}

	if !errors.Is(err, cause) {
		t.Error("the cause does not reach through Unwrap")
	}
}
