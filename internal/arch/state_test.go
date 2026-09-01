package arch

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// A task's state is folded from its events. It is never stored.
//
// This is the rule the move to SQL was most likely to lose, and it is a test
// rather than a habit because it costs nothing to break by accident: a board
// that reads a band out of a column is faster to write than one that folds a
// history, and it is wrong in a way nobody sees until a process is killed.
// The events say a phase started and nothing after; the column says
// "running"; the fold says "running" too, until reconcile writes
// task.abandoned and only one of the two hears about it. Two answers to one
// question separate the first time a run ends without being able to say so,
// which is exactly the case the record exists for.
//
// What may be stored is what a task *is* — the id somebody typed, the words
// they wrote, the flow they picked, when they wrote it down. What may not is
// what state it is in.
//
// run.outcome and phase.outcome are not exceptions to this, and the
// difference is worth naming. A span that has ended has an outcome that
// cannot change again, and the row is written in the same transaction as the
// event that ended it — so there is no window in which the two disagree, and
// no later event that would move one and not the other. A band is a function
// of every event a task has, including the ones not written yet.

// taskColumns is every column the task table is allowed to carry.
//
// It is an allowlist rather than a list of forbidden words for the reason
// arch.approved is one: a rule naming what is banned is a rule somebody gets
// past by choosing another noun, and "situation" would sail through a list
// that said "state". A column that belongs here can be added — in a commit
// of its own, saying why, like every other line of this package.
var taskColumns = []string{"id", "task_id", "text", "flow", "created_at"}

func TestTheRecordStoresNoTaskState(t *testing.T) {
	body := schemaSource(t)

	for _, got := range columnsOf(t, body, "task") {
		if !slices.Contains(taskColumns, got) {
			t.Errorf("the task table carries a column %q, which is not one of the things a task is (%s) — "+
				"if it is what state the task is in, it is folded from the events and must not be stored",
				got, strings.Join(taskColumns, ", "))
		}
	}
}

// TestNoTableIsATasksState is the same rule against the other way round it:
// a column refused on the task table is a table of its own the next morning.
func TestNoTableIsATasksState(t *testing.T) {
	for _, name := range tablesOf(schemaSource(t)) {
		for _, word := range []string{"state", "band", "status"} {
			if strings.Contains(name, word) {
				t.Errorf("the record has a table named %q — a task's state is folded from its events, not kept", name)
			}
		}
	}
}

// schemaSource is the file the record's shape is written in.
func schemaSource(t *testing.T) string {
	t.Helper()

	path := filepath.Join(root(t), "internal", "db", "schema.go")

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(body)
}

// tablesOf is the name of every table the schema creates.
func tablesOf(schema string) []string {
	var names []string

	for _, line := range strings.Split(schema, "\n") {
		name, ok := tableName(line)
		if ok {
			names = append(names, name)
		}
	}

	return names
}

// tableName reads the name out of a CREATE TABLE line, and says so when the
// line is not one.
func tableName(line string) (string, bool) {
	const opens = "CREATE TABLE IF NOT EXISTS "

	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, opens) {
		return "", false
	}

	return strings.TrimSuffix(strings.TrimPrefix(trimmed, opens), "("), true
}

// columnsOf is the name of every column one table declares.
//
// The schema is read as the text it is rather than by opening a database,
// because the rule is about what the file says: a column added and not yet
// released is still a column, and this has to fail in the commit that writes
// it and not the first time somebody runs the binary.
func columnsOf(t *testing.T, schema, table string) []string {
	t.Helper()

	var (
		names  []string
		inside bool
	)

	for _, line := range strings.Split(schema, "\n") {
		if name, ok := tableName(line); ok {
			inside = name == table
			continue
		}

		if !inside {
			continue
		}

		if strings.HasPrefix(strings.TrimSpace(line), ")") {
			return names
		}

		if name, ok := columnName(line); ok {
			names = append(names, name)
		}
	}

	t.Fatalf("the schema declares no table named %q", table)

	return nil
}

// columnName is the first word of a column line, and nothing for a line that
// declares a constraint rather than a column.
func columnName(line string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) == 0 {
		return "", false
	}

	switch strings.ToUpper(fields[0]) {
	case "PRIMARY", "FOREIGN", "UNIQUE", "CHECK", "CONSTRAINT":
		return "", false
	}

	return fields[0], true
}
