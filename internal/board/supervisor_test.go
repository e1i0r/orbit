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
	if err := record.Append(s.SupervisorLogPath(), ev); err != nil {
		t.Fatalf("record.Append: %v", err)
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
