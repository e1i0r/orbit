package board

import (
	"path/filepath"
	"testing"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

func TestReaderSupervisorLog(t *testing.T) {
	root := t.TempDir()

	s, err := store.New(filepath.Join(root, ".orbit"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	r := NewReader(s, root)

	// When log file doesn't exist
	lines, err := r.SupervisorLog()
	if err != nil {
		t.Fatalf("SupervisorLog on non-existing: %v", err)
	}

	if len(lines) != 0 {
		t.Errorf("lines len = %d, want 0", len(lines))
	}

	// Append event
	ev := record.Event{
		Kind: record.SupervisorMessage,
		Text: "running smoothly",
		Data: map[string]string{
			"by":      "supervisor",
			"channel": "autopilot",
		},
	}
	if err := appendMessage(t, s, ev); err != nil {
		t.Fatalf("append a turn to the thread: %v", err)
	}

	lines, err = r.SupervisorLog()
	if err != nil {
		t.Fatalf("SupervisorLog: %v", err)
	}

	if len(lines) != 1 {
		t.Fatalf("lines len = %d, want 1", len(lines))
	}

	if lines[0].Text != "running smoothly" || lines[0].By != "supervisor" || lines[0].Channel != "autopilot" {
		t.Errorf("lines[0] = %+v", lines[0])
	}
}

// TestTheSupervisorThreadIsReadWithoutABoard. It is one file under the state
// root and it belongs to no repository, so a caller that wants it needs no
// directory to point a Reader at — and both callers that wanted it were
// pointing one at "", which repo.Discover resolves to wherever the process
// was started from.
func TestTheSupervisorThreadIsReadWithoutABoard(t *testing.T) {
	s, err := store.New(filepath.Join(t.TempDir(), ".orbit"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	lines, err := SupervisorLog(s)
	if err != nil || len(lines) != 0 {
		t.Fatalf("SupervisorLog on a thread nobody has written = %v, %v; want none and no error", lines, err)
	}

	ev := record.Event{
		Kind: record.SupervisorMessage,
		Text: "the webhook task is stuck on its gate",
		Data: map[string]string{"by": "operator", "channel": "cli"},
	}
	if err := appendMessage(t, s, ev); err != nil {
		t.Fatalf("append a turn to the thread: %v", err)
	}

	lines, err = SupervisorLog(s)
	if err != nil {
		t.Fatalf("SupervisorLog: %v", err)
	}

	if len(lines) != 1 || lines[0].Text != ev.Text {
		t.Fatalf("the thread reads %+v, want the one line that was written", lines)
	}

	// And the Reader's method is the same answer, because it is the same
	// function underneath.
	viaReader, err := NewReader(s, t.TempDir()).SupervisorLog()
	if err != nil || len(viaReader) != 1 || viaReader[0].Text != ev.Text {
		t.Errorf("through a Reader the thread reads %+v, %v", viaReader, err)
	}
}

// appendMessage writes one turn of the supervisor thread through the state
// root's own handle, which is what internal/supervisor does and what the
// reader under test reads back.
func appendMessage(t *testing.T, s *store.Store, e record.Event) error {
	t.Helper()

	d, err := s.Record()
	if err != nil {
		return err
	}

	return d.AppendMessage(e)
}
