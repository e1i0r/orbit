package record

// The kinds added for stuck tasks, decisions and repositories joining: each
// one written down and read back with the fields it is meant to carry, and
// the older log that has none of them still read the same way.

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTheNewKindsSurviveTheRoundTrip(t *testing.T) {
	at := time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)

	want := []Event{
		{At: at, Kind: TaskStuck, Text: "the suite is red on every attempt for the same test", Data: map[string]string{"attempts": "3"}},
		{At: at.Add(time.Minute), Kind: DecisionMade, Text: "the index goes on (merchant_id, created_at)", Data: map[string]string{"id": "idx-order", "scope": "db/migrations"}},
		{At: at.Add(2 * time.Minute), Kind: DecisionSuperseded, Text: "created_at first: every query filters by date", Data: map[string]string{"id": "idx-order-2", "at": Stamp(at.Add(time.Minute))}},
		{At: at.Add(3 * time.Minute), Kind: RepoJoined, Data: map[string]string{"repo": "app", "path": "/w/app"}},
	}

	path := filepath.Join(t.TempDir(), "events.jsonl")
	for _, e := range want {
		if err := Append(path, e); err != nil {
			t.Fatalf("Append %s: %v", e.Kind, err)
		}
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("read %d events, wrote %d", len(got), len(want))
	}

	for i, w := range want {
		if got[i].Kind != w.Kind || got[i].Text != w.Text || !got[i].At.Equal(w.At) {
			t.Errorf("event %d = %+v, want %+v", i, got[i], w)
		}

		for k, v := range w.Data {
			if got[i].Data[k] != v {
				t.Errorf("event %d lost data[%q]: %q, want %q", i, k, got[i].Data[k], v)
			}
		}
	}

	// The superseded line names the decision it replaces the way a
	// retraction names a turn: by the stamp of the line itself.
	if got[2].Data["at"] != Stamp(got[1].At) {
		t.Errorf("the replacement names %q, and the decision it replaces is %q", got[2].Data["at"], Stamp(got[1].At))
	}
}

// A log written before any of these kinds existed is read exactly as it was.
// The bytes are what somebody's .orbit already holds; a reader that needs
// them rewritten to keep working is a reader that ate the record.
func TestALogOlderThanTheNewKindsStillReads(t *testing.T) {
	const old = `{"at":"2026-08-01T10:00:00Z","kind":"task.created","text":"retry the webhook on 5xx"}
{"at":"2026-08-01T10:00:05Z","kind":"task.started","data":{"flow":"task"}}
{"at":"2026-08-01T10:00:06Z","kind":"phase.started","phase":"implement","data":{"engine":"claude","model":"opus","n":"1"}}
{"at":"2026-08-01T10:04:00Z","kind":"phase.finished","phase":"implement","data":{"cost":"0.40"}}
{"at":"2026-08-01T10:04:01Z","kind":"task.finished"}
`

	path := filepath.Join(t.TempDir(), "events.jsonl")
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	wantKinds := []string{TaskCreated, TaskStarted, PhaseStarted, PhaseFinished, TaskFinished}
	if len(got) != len(wantKinds) {
		t.Fatalf("read %d events of the old log, want %d", len(got), len(wantKinds))
	}

	for i, k := range wantKinds {
		if got[i].Kind != k {
			t.Errorf("event %d is %q, want %q", i, got[i].Kind, k)
		}
	}

	if got[0].Text != "retry the webhook on 5xx" || got[2].Data["model"] != "opus" {
		t.Errorf("the old log came back changed: %+v %+v", got[0], got[2])
	}
}
