//go:build integration

package integration

// Reading back what a run wrote, through the door a person is given.
//
// `orbit export` and not the database file: the schema is Orbit's to change
// and the export is the promise it makes about what it holds, so a test that
// opened orbit.db would be testing something nobody is allowed to depend on.

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// event is one line of the exported record, in the fields these tests read.
type event struct {
	Kind  string            `json:"kind"`
	Phase string            `json:"phase"`
	Text  string            `json:"text"`
	Data  map[string]string `json:"data"`
}

// record is everything written about one task, oldest first.
func (b board) record(t *testing.T, id string) []event {
	t.Helper()

	dir := filepath.Join(t.TempDir(), "export")
	b.must(t, "export", "-task", id, dir)

	path := filepath.Join(dir, id, "events.jsonl")

	f, err := os.Open(path)
	if err != nil {
		// The export lays a task out under its id; find it wherever it put
		// it rather than assuming the shape of the tree.
		if found := findLog(t, dir); found != "" {
			f, err = os.Open(found)
		}

		if err != nil {
			t.Fatalf("open the exported record: %v", err)
		}
	}

	defer f.Close()

	var out []event

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), 8<<20)

	for sc.Scan() {
		var e event
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatalf("read a line of the exported record: %v", err)
		}

		out = append(out, e)
	}

	if err := sc.Err(); err != nil {
		t.Fatalf("read the exported record: %v", err)
	}

	return out
}

// findLog is the first events.jsonl under a directory.
func findLog(t *testing.T, dir string) string {
	t.Helper()

	var found string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && found == "" && !info.IsDir() && strings.HasSuffix(path, ".jsonl") {
			found = path
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk the export: %v", err)
	}

	return found
}

// kinds is the record as the list of what happened, which is what a flow
// test is about: the order of the phases, the gates, and how it ended.
func kinds(events []event) []string {
	out := make([]string, 0, len(events))
	for _, e := range events {
		out = append(out, e.Kind)
	}

	return out
}

// holds reports whether the record carries an event of this kind.
func holds(events []event, kind string) bool {
	for _, e := range events {
		if e.Kind == kind {
			return true
		}
	}

	return false
}

// count is how many of one kind the record holds.
func count(events []event, kind string) int {
	n := 0

	for _, e := range events {
		if e.Kind == kind {
			n++
		}
	}

	return n
}
