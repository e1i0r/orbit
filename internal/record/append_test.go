package record

// The writing side of the log: what Append puts on disk, and what it refuses
// to put there. read_test.go covers reading it back.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAppendThenReadRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	want := []Event{
		{Kind: "task.created", Text: "retry the webhook on 5xx"},
		{Kind: "phase.started", Phase: "implement"},
		{Kind: "phase.finished", Phase: "implement", Data: map[string]string{"exit": "0"}},
	}
	for _, e := range want {
		if err := Append(path, e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("read %d events, wrote %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Kind != want[i].Kind || got[i].Phase != want[i].Phase || got[i].Text != want[i].Text {
			t.Errorf("event %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	if got[2].Data["exit"] != "0" {
		t.Errorf("data did not survive: %+v", got[2].Data)
	}
}

func TestAppendStampsTheTime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	before := time.Now().Add(-time.Second)
	if err := Append(path, Event{Kind: "task.created"}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got[0].At.Before(before) {
		t.Errorf("At = %v, expected a stamp from now", got[0].At)
	}
}

func TestAppendKeepsAGivenTime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	want := time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC)
	if err := Append(path, Event{Kind: "task.created", At: want}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !got[0].At.Equal(want) {
		t.Errorf("At = %v, want %v", got[0].At, want)
	}
}

func TestOneEventPerLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	for i := 0; i < 3; i++ {
		if err := Append(path, Event{Kind: "phase.started", Text: "a\nb"}); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("file has %d lines for 3 events — a newline in the text broke the format", len(lines))
	}
}

// atTheCap builds an event whose marshalled line, newline included, is
// exactly n bytes. The padding is plain ASCII so that JSON escaping cannot
// change the count under it.
func atTheCap(t *testing.T, n int) Event {
	t.Helper()
	e := Event{Kind: "phase.finished", At: time.Date(2026, 8, 22, 10, 0, 0, 0, time.UTC), Text: "x"}
	line, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	e.Text = strings.Repeat("x", n-len(line))
	line, err = json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(line)+1 != n {
		t.Fatalf("built a line of %d bytes, wanted %d", len(line)+1, n)
	}
	return e
}

func TestALineAtTheCapRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	want := atTheCap(t, MaxLine)
	if err := Append(path, want); err != nil {
		t.Fatalf("Append refused a line of exactly MaxLine bytes: %v", err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d events, want 1 — the reader cannot read back what the writer accepted", len(got))
	}
	if got[0].Text != want.Text {
		t.Errorf("text came back %d bytes, wrote %d", len(got[0].Text), len(want.Text))
	}
}

func TestAppendRefusesALineOverTheCap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	tooBig := atTheCap(t, MaxLine+1)
	err := Append(path, tooBig)
	if err == nil {
		t.Fatal("Append wrote a line the reader can never read back, poisoning the whole log")
	}
	if !strings.Contains(err.Error(), "phase.finished") {
		t.Errorf("the refusal does not name the event kind: %v", err)
	}
	if !strings.Contains(err.Error(), strconv.Itoa(MaxLine+1)) {
		t.Errorf("the refusal does not say how big the line was: %v", err)
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Error("the refused write still created the log")
	}
}

func TestAGoodLineStillReadsAfterOneWasRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	if err := Append(path, Event{Kind: "task.created", Text: "first"}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := Append(path, atTheCap(t, MaxLine+1)); err == nil {
		t.Fatal("Append accepted an oversized line")
	}
	if err := Append(path, Event{Kind: "task.finished"}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v — one oversized event must not cost the whole record", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2", len(got))
	}
}

func TestTheLogIsPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	if err := Append(path, Event{Kind: "task.created", Text: "retry the webhook on 5xx"}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("the log is %o, want 600 — it holds every word the engines printed", perm)
	}
}

// TestAppendSurfacesAFailureRatherThanSwallowingIt is the reachable half of
// "the log must never fail quietly". Forcing fsync or close to fail needs a
// fault-injecting filesystem, so what is pinned here is that a log which
// cannot be written reports it instead of returning nil.
func TestAppendSurfacesAFailureRatherThanSwallowingIt(t *testing.T) {
	dir := t.TempDir()
	if err := Append(dir, Event{Kind: "task.created"}); err == nil {
		t.Error("Append reported success writing to a path it cannot possibly have written")
	}
}
