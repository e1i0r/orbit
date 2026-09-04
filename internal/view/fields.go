package view

// The small readers: one event's fields turned into one Task's. They are
// separate from the walk in fold.go so that the walk stays a flat table of
// kinds a reader can check against the writer, and so that neither file has
// to grow past the point where it is read in one sitting.

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// waiting decides which of the two stops this is. The flow asking a phase to
// wait needs you: nothing moves until you answer. The reader asking for it
// does not — a paused run still holds a worktree and still holds a slot, so
// it stays in Running with a word that says it is held, and reporting the
// operator's own pause as a warning is how a warning channel stops being
// read.
//
// A phase.waiting that does not say why is treated as the flow's, because
// the two mistakes are not equal: a held task shown as needing you is noise,
// and a task that needs you shown as merely held is a task nobody comes back
// to.
func waiting(e record.Event) (state, Reason) {
	args := []Arg{{Name: "phase", Value: e.Phase}}
	if e.Data["why"] == whyPaused {
		return stateHeld, Reason{Key: ReasonHeld, Args: args}
	}

	return stateWaiting, Reason{Key: ReasonGate, Args: args}
}

// failure names the phase a run stopped in, or says the run never got that
// far. internal/task writes task.failed with no Phase on it in every case —
// task-level events name no phase, phase-level events do — so the phase is
// whatever the fold knew while it was still inside the attempt, and a run
// that was inside one without a phase to its name is a run that had not
// reached its first.
func failure(phase string) Reason {
	if phase == "" {
		return Reason{Key: ReasonFailedToStart}
	}

	return Reason{Key: ReasonFailed, Args: []Arg{{Name: "phase", Value: phase}}}
}

// flow takes the flow's name from an event that carries one. A missing key
// leaves the last name standing: task.created says what the task was written
// against and task.started says what a run overrode it with, and an event
// that names neither is not an event saying the flow was withdrawn.
func flow(t *Task, e record.Event) {
	if name, ok := e.Data["flow"]; ok {
		t.Flow = name
	}
}

// stamp moves a time only when the event carried an honest one. A zero At is
// a damaged timestamp, and it would draw as an elapsed of half a century;
// the last real time this task had is a better answer than that, and a Task
// that never had one keeps the zero so the window can say so.
func stamp(when *time.Time, at time.Time) {
	if !at.IsZero() {
		*when = at
	}
}

// count reads a 1-based phase number. Anything that is not one — absent,
// misspelled, zero, negative — is 0, which is this package's way of saying
// the record does not know. Guessing 1 would put `1/3` on a row whose record
// never said which phase it was in.
func count(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 0
	}

	return n
}

// money reads one phase's cost. A value that will not parse contributes
// nothing rather than poisoning the sum — and NaN and the infinities are
// refused by name, because ParseFloat accepts all three and a single NaN
// would make every total after it NaN for the rest of the log. A negative
// cost is not a discount, it is a damaged field.
func money(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}

	return v
}

// firstLine is the title: everything up to the first line break, trimmed.
// The rest of task.md is the task itself and the window has a pane for it.
//
// A carriage return breaks a line as surely as a newline does, and it breaks
// it twice over on a terminal: the row is drawn, the cursor returns to
// column one, and the rest of the title is drawn over the columns beside it.
// A task written with CR line endings arrives that way.
func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}

	return strings.TrimSpace(s)
}

// actionKeys are the arguments a tool call is about, in the order they are
// looked for. What a reader wants off a row is the command that is running,
// the file that is being read or the pattern that is being searched for, not
// the JSON around it.
var actionKeys = []string{"command", "file_path", "path", "pattern"}

// actionPathKeys are the ones among them whose value is a path, and so the
// ones worth shortening against the worktree the run was given.
var actionPathKeys = map[string]bool{"file_path": true, "path": true}

// ToolLine is one tool call written for a reader: the name of the tool, and
// then the one argument it is about.
//
// It is not cut here. This is the model three readers share — the band, the
// overview and the MCP server — and a measure kept in the model is the
// measure of whichever of them was thought of first. Fifty characters was
// the band's, where the row also carries an id, a phase, an elapsed time, an
// engine and a flow; the overview, which has the width of the window, drew
// the same fifty and cut a command that had room to be read. Whoever draws
// the line is the one that knows what it has to fit in.
//
// It is exported because the timeline had a reader of its own that searched
// the arguments for `"command"` and then for `:"` — the search this package
// documents as the one that does not work — and so showed raw JSON for every
// call about a file. One formatter, and every reader reads the same.
func ToolLine(tool, args string) string {
	tool = strings.TrimSpace(tool)
	if tool == "" {
		return ""
	}

	head := oneLine(subject(strings.TrimSpace(args)))
	if head == "" {
		return tool
	}

	return tool + ": " + head
}

// subject is the one argument of a tool call worth putting on a row, or the
// whole of what was given if none of them is there.
//
// It goes through encoding/json rather than searching the text for `"path"`
// and then for `:"`. That search wants the value to begin immediately after
// the colon, so an engine that writes `{"command": "go test"}` with the
// space every JSON encoder puts there matches the key, misses the value, and
// puts the whole document on the row. What such a search finds it also hands
// over unescaped, so a Windows path arrives with its backslashes doubled and
// a quotation mark inside a string cuts the value short.
func subject(args string) string {
	if !strings.HasPrefix(args, "{") {
		return args
	}

	var fields map[string]any
	if err := json.Unmarshal([]byte(args), &fields); err != nil {
		// Arguments that are not JSON after all. They were shown as they
		// stood before this function was reached for, and they still are.
		return args
	}

	for _, key := range actionKeys {
		if value, ok := fields[key].(string); ok && strings.TrimSpace(value) != "" {
			if actionPathKeys[key] {
				return underWorktree(value)
			}

			return value
		}
	}

	return args
}

// underWorktree is a path with the worktree it is inside taken off the front
// of it.
//
// A run works in ~/.orbit/worktrees/<repo>/<task>, so every path it reports
// begins with those five segments. They are the same on every row of the
// run, they are most of what fits on a row, and cutting the row to the
// measure cuts away the file — which is the only part that differs.
//
// The two segments after `worktrees` are the repository and the task, and a
// path that is only those is the worktree root: there is nothing under it to
// name, so it names itself.
func underWorktree(path string) string {
	parts := strings.Split(path, "/")
	for i, seg := range parts {
		if seg == "worktrees" && len(parts) > i+3 {
			return strings.Join(parts[i+3:], "/")
		}
	}

	return path
}

// oneLine folds a value written over several lines onto the one line a row
// has for it.
//
// Stopping at the first break is what this did, and a shell command written
// with a trailing backslash begins `grep -rn \` — which is where the break
// is. Every such row read the same and said nothing about any of them. The
// backslash goes with the break it was continuing.
func oneLine(s string) string {
	lines := strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == '\r' })
	for i, line := range lines {
		lines[i] = strings.TrimSuffix(strings.TrimSpace(line), "\\")
	}

	return strings.TrimSpace(strings.Join(strings.Fields(strings.Join(lines, " ")), " "))
}
