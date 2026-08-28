package record

// What a reader sees while somebody is still writing.
//
// A log is polled twice a second by a window whose runs are appending to it
// the whole time, so "the file changed between two of my own syscalls" is
// not an edge case here — it is the ordinary case, and both readers used to
// measure the file twice and describe the one that never existed.

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

// TestReadFromHandsBackEveryEventExactlyOnce is the regression. ReadFrom
// stated the size, then looked at the last byte against a fresh stat, then
// scanned to whatever the end had become — and returned the first of those
// three as the next offset. Anything appended in between was handed to the
// caller and then handed to it again on the next call, so a poller folding
// what it was given saw one attempt twice.
func TestReadFromHandsBackEveryEventExactlyOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	const total = 300

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range total {
			if err := Append(path, Event{
				Kind: TaskNoted,
				Text: strconv.Itoa(i),
			}); err != nil {
				panic(err)
			}
		}
	}()

	seen := map[string]int{}
	var offset int64
	deadline := time.Now().Add(20 * time.Second)
	for len(seen) < total {
		if time.Now().After(deadline) {
			t.Fatalf("only %d of %d events arrived", len(seen), total)
		}
		events, next, err := ReadFrom(path, offset)
		if err != nil {
			t.Fatalf("ReadFrom at %d: %v", offset, err)
		}
		if next < offset {
			t.Fatalf("the offset went backwards, %d to %d, on a log that only grew", offset, next)
		}
		offset = next
		for _, e := range events {
			if e.Kind == Unreadable {
				t.Fatalf("a write in flight was reported as damage at byte %s", e.Data["byte"])
			}
			seen[e.Text]++
			if seen[e.Text] > 1 {
				t.Fatalf("event %q came back %d times", e.Text, seen[e.Text])
			}
		}
	}
	wg.Wait()

	// And nothing was skipped on the way, which is the other half of
	// "exactly once".
	for i := range total {
		if seen[strconv.Itoa(i)] != 1 {
			t.Fatalf("event %d was seen %d times", i, seen[strconv.Itoa(i)])
		}
	}
}

// TestReadOfALogBeingWrittenReportsNoDamage: Read checked the last byte
// against one file and scanned another, so a line that landed in between was
// torn against a file that had already been called whole — and a
// record.unreadable was synthesised for a write that was merely still in
// flight. A reader would have gone looking for damage that was never there.
func TestReadOfALogBeingWrittenReportsNoDamage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	const total = 300

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range total {
			if err := Append(path, Event{Kind: TaskNoted, Text: strconv.Itoa(i)}); err != nil {
				panic(err)
			}
		}
	}()

	for reads := 0; ; reads++ {
		events, err := Read(path)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		for _, e := range events {
			if e.Kind == Unreadable {
				t.Fatalf("read %d called a write in flight damage at byte %s", reads, e.Data["byte"])
			}
		}
		select {
		case <-done:
			return
		default:
		}
	}
}

// TestReadFromNamesTheDamagedLineByWhereItIsInTheLog: the mark used to carry
// a line number counted from wherever that particular read started, so the
// same broken line was "line 2" to a reader that had read the log before and
// "line 4" to one that had not. A byte offset is the same fact for both.
func TestReadFromNamesTheDamagedLineByWhereItIsInTheLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	for i := range 3 {
		if err := Append(path, Event{Kind: TaskNoted, Text: strconv.Itoa(i)}); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	_, offset, err := ReadFrom(path, 0)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := fmt.Fprintln(f, "{not json}"); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	events, _, err := ReadFrom(path, offset)
	if err != nil {
		t.Fatalf("ReadFrom at %d: %v", offset, err)
	}
	if len(events) != 1 || events[0].Kind != Unreadable {
		t.Fatalf("events = %+v, want one mark", events)
	}
	if got := events[0].Data["byte"]; got != strconv.FormatInt(offset, 10) {
		t.Errorf("the mark names byte %s, want %d — where the line begins in the log", got, offset)
	}
}
