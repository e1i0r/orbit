package verb

// What the readings read.
//
// A reading answers twice. Saw is the thing itself, in the shape it was read
// in, for a surface that can draw a structure; Said is the same answer
// written out for a terminal. Both, rather than one, because the alternative
// was each surface folding the record its own way — which is how `orbit
// supervisor` and the browser's thread came to number their lines
// differently in the first place.

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/supervisor"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// heard is the supervisor's thread, numbered.
//
// The number is the position in this listing and nothing more durable than
// that: an event has no id, and the thread only grows at the end, so what is
// line 3 today is line 3 tomorrow. retract reads it back the same way.
func heard(w World) (Out, error) {
	lines, err := thread(w)
	if err != nil {
		return Out{}, err
	}

	if len(lines) == 0 {
		return Out{Said: w.Words().T("verb.thread.empty", "the supervisor thread is empty"), Saw: lines}, nil
	}

	var b strings.Builder
	for i, l := range lines {
		fmt.Fprintf(&b, "%3d  %s\n", i+1, spokenLine(l))
	}

	return Out{Said: strings.TrimRight(b.String(), "\n"), Saw: lines}, nil
}

// unsaid takes back a line of the thread.
//
// The line stays in the record and stays in the listing. It was said, and
// hiding it would leave the rest of the thread answering something that is
// not there.
func unsaid(w World, in In) (Out, error) {
	n, err := strconv.Atoi(in.Arg("line"))
	if err != nil {
		return Out{}, errors.New(w.Words().T("verb.retract.not_a_number",
			"retract takes the number of a line in the thread, not {value}",
			words.Arg{Name: "value", Value: in.Arg("line")}))
	}

	lines, err := thread(w)
	if err != nil {
		return Out{}, err
	}

	if n < 1 || n > len(lines) {
		return Out{}, errors.New(w.Words().T("verb.retract.no_such_line",
			"there is no line {n} in the supervisor thread; it has {count}",
			words.Arg{Name: "n", Value: strconv.Itoa(n)},
			words.Arg{Name: "count", Value: strconv.Itoa(len(lines))}))
	}

	l := lines[n-1]
	if l.Retracted {
		return Out{}, errors.New(w.Words().T("verb.retract.already",
			"line {n} was already taken back", words.Arg{Name: "n", Value: strconv.Itoa(n)}))
	}

	if err := supervisor.Retract(w.Store(), l.At); err != nil {
		return Out{}, fmt.Errorf("retract supervisor message: %w", err)
	}

	return Out{Said: w.Words().T("verb.retract.done", "took back line {n}: {text}",
		words.Arg{Name: "n", Value: strconv.Itoa(n)},
		words.Arg{Name: "text", Value: l.Text})}, nil
}

// said is the thread as the record holds it.
func thread(w World) ([]view.SupervisorLine, error) {
	lines, err := board.SupervisorLog(w.Store())
	if err != nil {
		return nil, fmt.Errorf("read the supervisor's history: %w", err)
	}

	return lines, nil
}

// spokenLine is one thread entry as it reads on a terminal.
func spokenLine(l view.SupervisorLine) string {
	tag := fmt.Sprintf("[%s via %s]", l.By, l.Channel)
	if l.TaskID != "" {
		tag += fmt.Sprintf(" (%s)", l.TaskID)
	}

	if l.Retracted {
		tag += " (retracted)"
	}

	return fmt.Sprintf("%s %s %s", l.At.Format("15:04:05"), tag, l.Text)
}

// kept is every setting and what it holds.
func kept(w World) (Out, error) {
	cfg, err := w.Store().Settings()
	if err != nil {
		return Out{}, err
	}

	p := w.Words()
	all := make([]Setting, 0, len(settingTable()))

	for _, one := range settingTable() {
		all = append(all, Setting{Name: one.Name, Value: one.Value(cfg), About: one.About(p)})
	}

	name, value := 0, 0
	for _, s := range all {
		name = max(name, len(s.Name))
		value = max(value, len(unset(s.Value)))
	}

	var b strings.Builder
	for _, s := range all {
		fmt.Fprintf(&b, "%-*s  %-*s  %s\n", name, s.Name, value, unset(s.Value), s.About)
	}

	return Out{Said: strings.TrimRight(b.String(), "\n"), Saw: all}, nil
}

// changed writes one setting.
//
// The read, the change and the write are one step. Between them, the window
// is another process writing the whole of this file back from a copy it read
// before this line ran: two settings changed at once would be one setting
// changed and one silently discarded.
func changed(w World, in In) (Out, error) {
	var now string

	err := w.Store().UpdateSettings(func(cfg *store.Settings) error {
		var err error

		now, err = assign(w.Words(), cfg, in.Arg("key"), in.Arg("value"))

		return err
	})
	if err != nil {
		return Out{}, err
	}

	return Out{Said: w.Words().T("verb.set.now", "{key} is now {value}",
		words.Arg{Name: "key", Value: in.Arg("key")},
		words.Arg{Name: "value", Value: now})}, nil
}

// unset is what an empty setting prints as. An empty column reads as a table
// that failed to render; a dash reads as an answer.
func unset(value string) string {
	if value == "" {
		return "—"
	}

	return value
}
