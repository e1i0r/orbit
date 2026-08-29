package supervisor

// What happens when the thread file itself is against you: unreadable on the
// way in, unwritable on the way out.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/record"
)

// unreadable makes the thread a file record.Read refuses: one line longer
// than the longest it will hold. Truncated JSON would not do — the reader
// turns a line it cannot parse into a record.Unreadable event and carries
// on, which is the point of that kind.
func unreadable(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if err := os.WriteFile(path, []byte(strings.Repeat("x", record.MaxLine+1)+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// TestARetractionOnAThreadThatWillNotReadSaysSo. Retract has to find the turn
// before it can take it back, and a thread it cannot read is not a thread with
// nothing in it — answering "nothing was written at that time" would tell the
// operator their timestamp was wrong when the truth is that the log is broken.
func TestARetractionOnAThreadThatWillNotReadSaysSo(t *testing.T) {
	s := fixture(t)
	unreadable(t, s.SupervisorLogPath())

	err := Retract(s, time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("Retract answered nil over a thread it could not read")
	}

	if strings.Contains(err.Error(), "nothing in the supervisor thread") {
		t.Errorf("a broken log was reported as a timestamp nobody wrote at: %v", err)
	}
}

// TestAnAnswerThatCannotBeWrittenDownIsStillReturned.
//
// The answer is the expensive part — a model ran to produce it — and the
// caller is a cockpit that draws it. Losing it because the log would not take
// it would mean paying for a supervisor turn and showing nothing, so the text
// comes back beside the error rather than instead of it.
func TestAnAnswerThatCannotBeWrittenDownIsStillReturned(t *testing.T) {
	s := fixture(t)

	path := s.SupervisorLogPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if err := os.WriteFile(path, nil, 0o400); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	t.Cleanup(func() { _ = os.Chmod(path, 0o600) }) //nolint:errcheck // best effort, the root is a temp dir

	ans, err := Supervise(context.Background(), s, &engine.Fake{Output: "the board is fine"}, "how is it going?")
	if err == nil {
		t.Fatal("Supervise answered nil over a thread it could not append to")
	}

	if ans != "the board is fine" {
		t.Errorf("the answer was dropped along with the write that failed: %q", ans)
	}
}
