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
	"unicode/utf8"

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

// firstLine is the title: everything up to the first newline, trimmed. The
// rest of task.md is the task itself and the window has a pane for it.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}

	return strings.TrimSpace(s)
}

// actionChars is how much of an action fits on a row that also carries an
// id, a phase, an elapsed time and an engine.
const actionChars = 50

// actionKeys are the arguments a tool call is about, in the order they are
// looked for. What a reader wants off a row is the command that is running
// or the file that is being written, not the JSON around it.
var actionKeys = []string{"command", "file_path", "path"}

// formatAction turns one tool call into the line beside a running task.
func formatAction(tool, args string) string {
	tool = strings.TrimSpace(tool)
	if tool == "" {
		return ""
	}

	if head := firstLine(subject(strings.TrimSpace(args))); head != "" {
		return tool + ": " + clip(head, actionChars)
	}

	return tool
}

// subject is the one argument of a tool call worth putting on a row, or the
// whole of what was given if none of them is there.
//
// It goes through encoding/json rather than searching the text for `"path"`
// and then for `:"`, which is what it used to do — three near-identical
// blocks of it. That search wanted the value to begin immediately after the
// colon, so an engine that wrote `{"command": "go test"}` with the space
// every JSON encoder puts there matched the key, missed the value, and put
// the whole document on the row. What it found it also handed over
// unescaped, so a Windows path arrived with its backslashes doubled and a
// quotation mark inside a string cut the value short.
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
			return value
		}
	}

	return args
}

// clip shortens a line to n characters, counting characters and not bytes.
//
// Slicing bytes is what this did — head[:47] — and a path or a prompt with
// an accent in it can have a character on that boundary. Half of one is not
// a character at all, and what the terminal draws for the half that arrives
// is up to the terminal.
func clip(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}

	return string([]rune(s)[:n-1]) + "…"
}
