package ui

// Everything the window is waiting on, said the same way.
//
// A gesture that starts something slow and then goes quiet is a gesture the
// reader presses again — and behind most of these is a run somebody pays
// for. So each one says four things in the band, always in this order: that
// it is moving, what it is, how long it has been, and the key that ends the
// wait when there is one.
//
// One shape rather than a sentence per verb, because the reader learns it
// once. What differs between them is only what can be said honestly: Orbit
// holds a handle for some of this work and not for the rest, and a line that
// offered a key that does nothing would be worse than one that offers none.

import "time"

// busy is one thing in flight.
type busy struct {
	what  string    // what is happening, in the reader's language
	since time.Time // when it started, for the count of seconds
	stop  string    // the key that ends the wait, or empty when there is none
}

// waitingOn is everything in flight right now, the newest first.
func (m Model) waitingOn() []busy {
	p := m.opts.Words

	var out []busy

	if m.flows.saying {
		out = append(out, busy{
			what:  p.T("wait.draft", "asking {engine} for a flow", about("engine", m.sayEngineName())),
			since: m.flows.sayAt,
			// Orbit did not spawn that engine with a handle it can kill, so
			// escape stops waiting and drops the answer: see stopWaiting.
			stop: "esc",
		})
	}

	if m.supervisorBusy {
		out = append(out, busy{
			what:  p.T("wait.supervisor", "the supervisor is thinking"),
			since: m.supervisorAt,
		})
	}

	if m.delivering.verb != "" {
		out = append(out, busy{
			what: p.T("wait.deliver", "{verb} on {id}",
				about("verb", m.delivering.verb), about("id", m.delivering.task.ID)),
			since: m.delivering.at,
		})
	}

	if _, done := watchState(m.watching); m.watching != nil && !done {
		out = append(out, busy{
			what:  p.T("wait.command", "{name} is running", about("name", m.watching.name)),
			since: m.watching.at,
		})
	}

	return out
}

// waitingLine is those, drawn: the first of them in full, and a count of the
// rest when more than one thing is out at once.
func (m Model) waitingLine() string {
	waits := m.waitingOn()
	if len(waits) == 0 {
		return ""
	}

	first := waits[0]

	line := first.what
	if !first.since.IsZero() {
		line += " — " + m.now.Sub(first.since).Round(time.Second).String()
	}

	if first.stop != "" {
		line += " · " + m.opts.Words.T("wait.stop", "[{key}] stop waiting", about("key", first.stop))
	}

	if rest := len(waits) - 1; rest > 0 {
		line += " · " + m.opts.Words.P("wait.more", rest, "{n} more", "{n} more")
	}

	return m.spinner(Live) + Paint(Live).Render(line)
}

// watchState is whether the command a watch is holding has finished, and
// nothing when there is no watch at all: the buffer is written from the
// goroutine that runs the command, so it is asked rather than read.
func watchState(w *commandWatch) (string, bool) {
	if w == nil {
		return "", true
	}

	return w.snapshot()
}
