package web

// The flow a task was started under, and how far the run got through it.
//
// The reading is the cockpit's: the flow resolved off disk, and the record
// walked for what happened to each phase. What is different here is the
// answer — data rather than a tree of box-drawing characters — and one thing
// the terminal does not show. A loop is drawn there as a single node saying
// "going round", because a tree of a block of phases inside a phase is more
// depth than a pane has room for. A browser has the room, so the phases
// inside a loop are answered too.
//
// The delivery verbs the cockpit hangs off the foot of its tree are not
// here. Pairing an answer to the ask it closes is a reading of the record,
// and internal/view is where a reading of the record belongs; a second copy
// in this package would be a second opinion about the same events. The
// timeline shows both of them already.

import (
	"net/http"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/view"
)

// flowAnswer is the flow and every phase of it.
type flowAnswer struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Attempts    int           `json:"attempts"`
	DiffBudget  int           `json:"diffBudget,omitempty"`
	Phases      []phaseAnswer `json:"phases,omitempty"`
	// Failed is why there is no flow to show. A name the disk no longer
	// answers is the reader's business and not a fault of the server, so it
	// arrives in the shape everything else does rather than as a 500.
	Failed string `json:"failed,omitempty"`
}

// phaseAnswer is one step: how the flow set it up, and what the record says
// became of it.
type phaseAnswer struct {
	Name        string       `json:"name"`
	Standing    string       `json:"standing,omitempty"`
	Engine      string       `json:"engine,omitempty"`
	Model       string       `json:"model,omitempty"`
	Effort      string       `json:"effort,omitempty"`
	Thinking    string       `json:"thinking,omitempty"`
	Waits       bool         `json:"waits,omitempty"`
	Permissions []string     `json:"permissions,omitempty"`
	Gates       []gateAnswer `json:"gates,omitempty"`
	Loop        *loopAnswer  `json:"loop,omitempty"`
	Cost        float64      `json:"cost,omitempty"`
	Started     *time.Time   `json:"started,omitempty"`
	Ended       *time.Time   `json:"ended,omitempty"`
	Cause       string       `json:"cause,omitempty"`
	Exit        string       `json:"exit,omitempty"`
	Said        string       `json:"said,omitempty"`
}

// gateAnswer is one command a phase has to satisfy, by name.
type gateAnswer struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// loopAnswer is a phase that is a block of phases: what it repeats, what
// says it may stop, and how many turns it has taken.
type loopAnswer struct {
	Max    int           `json:"max"`
	Turns  int           `json:"turns"`
	Until  []gateAnswer  `json:"until,omitempty"`
	Phases []phaseAnswer `json:"phases,omitempty"`
}

// serveFlow is the flow of one task and where the run got to in it.
func (s *Server) serveFlow(w http.ResponseWriter, r *http.Request) {
	t, ok := s.find(w, r)
	if !ok {
		return
	}

	entries, err := s.board.Log(t.RepoPath, t.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "read the record of "+t.ID, err)

		return
	}

	f, err := flow.Resolve(s.flows, t.Flow)
	if err != nil {
		answer(w, flowAnswer{ID: t.ID, Name: t.Flow, Failed: err.Error()})

		return
	}

	answer(w, flowOf(t, f, entries))
}

// flowOf is one resolved flow read against one record.
func flowOf(t view.Task, f flow.Flow, entries []view.Entry) flowAnswer {
	out := flowAnswer{
		ID: t.ID, Name: f.Name, Description: f.Description,
		Attempts: f.AttemptCap(), DiffBudget: f.DiffBudget,
	}

	for i := range f.Phases {
		out.Phases = append(out.Phases, phaseOf(t, entries, f.Phases, i))
	}

	return out
}

// phaseOf is one phase of a list of them, and what happened to it.
func phaseOf(t view.Task, entries []view.Entry, phases []flow.Phase, i int) phaseAnswer {
	p := phases[i]
	ran := ranAs(entries, p.Name)

	out := phaseShape(p)
	out.Cost, out.Cause, out.Exit, out.Said = ran.cost, ran.cause, ran.exit, ran.said
	out.Engine = latest(p.Engine, ran.engine)
	out.Model = latest(p.Model, ran.model)
	out.Standing = standingOf(ran, t, p.Name, past(entries, phases, i))
	out.Started, out.Ended = at(ran.began), at(ran.ended)

	if p.Loop != nil {
		out.Loop = loopOf(t, entries, *p.Loop, ran.turns)
	}

	return out
}

// phaseShape is a phase as the flow file writes it, with nothing about a run
// in it. It is what the flows screen shows, where no run has happened, and
// the half of a phase the task screen does not have to read the record for.
func phaseShape(p flow.Phase) phaseAnswer {
	out := phaseAnswer{
		Name: p.Name, Engine: p.Engine, Model: p.Model, Effort: p.Effort,
		Thinking: p.Thinking, Waits: p.Wait, Permissions: p.Permissions,
	}

	for _, g := range p.Gates {
		out.Gates = append(out.Gates, gateAnswer{Name: g.Name, Command: g.Command})
	}

	if p.Loop != nil {
		out.Loop = &loopAnswer{Max: p.Loop.Max}
		for _, g := range p.Loop.Until {
			out.Loop.Until = append(out.Loop.Until, gateAnswer{Name: g.Name, Command: g.Command})
		}

		for _, inner := range p.Loop.Phases {
			out.Loop.Phases = append(out.Loop.Phases, phaseShape(inner))
		}
	}

	return out
}

// loopOf is the block a loop repeats, with each phase inside it read the
// same way a phase outside one is.
func loopOf(t view.Task, entries []view.Entry, l flow.Loop, turns int) *loopAnswer {
	out := &loopAnswer{Max: l.Max, Turns: turns}

	for _, g := range l.Until {
		out.Until = append(out.Until, gateAnswer{Name: g.Name, Command: g.Command})
	}

	for i := range l.Phases {
		out.Phases = append(out.Phases, phaseOf(t, entries, l.Phases, i))
	}

	return out
}

// ran is what the record says happened to one phase.
type ran struct {
	finished  bool
	failed    bool
	cancelled bool
	waiting   bool
	// turns is how many times a loop wrote down a turn under this phase's
	// name. A loop runs no engine of its own — the phases inside it do — so
	// it has no phase.started, and this is the only thing that tells a loop
	// going round from a loop nobody has reached.
	turns  int
	cost   float64
	cause  string
	exit   string
	said   string
	engine string
	model  string
	began  time.Time
	ended  time.Time
}

// ranAs reads the record for one phase, by name.
func ranAs(entries []view.Entry, name string) ran {
	var out ran

	for _, e := range entries {
		if !strings.EqualFold(e.Phase, name) {
			continue
		}

		switch e.What() {
		case view.EntryLoopChecked:
			out.turns++
		case view.EntryStarted:
			out.began = e.At
			out.engine = latest(out.engine, e.Engine)
			out.model = latest(out.model, e.Model)
		case view.EntryFinished:
			out.finished, out.ended, out.cost, out.said = true, e.At, e.Cost, e.Said()
		case view.EntryFailed:
			out.failed, out.ended, out.cost = true, e.At, e.Cost
			out.cause, out.exit, out.said = e.Cause, e.Exit, e.Said()
		case view.EntryCancelled:
			out.cancelled, out.said = true, e.Said()
		case view.EntryWaiting:
			out.waiting, out.cause = true, e.Cause
		default:
			// Every other kind belongs to the phase without saying
			// anything about whether it ran: a tool call, a note, a gate.
		}
	}

	return out
}

// standingOf is where a phase got to, in one word.
//
// Words and not numbers, for the reason the bands are words: the order these
// are tested in is this package's, and a page keying off it would be reading
// a decision it cannot see.
func standingOf(r ran, t view.Task, name string, gone bool) string {
	switch {
	case r.failed:
		return "failed"
	case r.cancelled:
		return "cancelled"
	case r.waiting:
		return "waiting"
	case r.finished, gone:
		// A phase the run has gone past is done whatever it wrote about
		// itself: a loop writes no finish of its own, and the flow moving on
		// is the only thing that says its block closed.
		return "done"
	case r.turns > 0:
		return "looping"
	case t.Band == view.Running && strings.EqualFold(t.Phase, name):
		return "running"
	}

	return "pending"
}

// past is whether the run has gone beyond this phase: a later one in the
// same list started, waited, finished, broke, was stopped or went round.
func past(entries []view.Entry, phases []flow.Phase, i int) bool {
	for j := i + 1; j < len(phases); j++ {
		later := ranAs(entries, phases[j].Name)
		if !later.began.IsZero() || later.finished || later.waiting ||
			later.failed || later.cancelled || later.turns > 0 {
			return true
		}
	}

	return false
}

// latest keeps what the record said over what the flow asked for, and leaves
// what is known standing when the record said nothing.
func latest(asked, said string) string {
	if said != "" {
		return said
	}

	return asked
}

// at is a time the page can read, and nothing for a clock that never
// answered: a zero time serialises as year one, which draws as a date.
func at(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}

	return &t
}
