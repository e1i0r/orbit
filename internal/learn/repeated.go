package learn

// What somebody keeps telling runs.
//
// The rules a person means to lay down they type, and those are the tray
// above. This is the other half: the thing they would never think to say
// because they do not know they do it.
//
// "add fuzz testing here". Said once it is a correction to one task. Said in
// the test phase of the last six tasks it is a rule, and nobody ever
// enunciated it — so the shape test throws it away, rightly, because read on
// its own it is not one. What makes it a rule is the repetition and not the
// words.
//
// Nothing here proposes anything. It reads what is already in the record —
// every directive is written down with its task, and the phase the work was
// in can be read off the events around it — and puts it in a shape somebody,
// or something, can look at.

import (
	"fmt"
	"sort"
	"time"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// A Directive is one thing somebody told a run, and where they were standing
// when they told it.
type Directive struct {
	At   time.Time
	Task string
	// Repo is the checkout the task is worked in, and Phase is the phase
	// the work was in at that moment. Both are what a habit is grouped by:
	// the same words in two projects are two habits, and the same words in
	// the plan phase and the test phase are two rules.
	Repo  string
	Phase string
	Text  string
}

// A Habit is one instruction given to several runs.
type Habit struct {
	Repo  string
	Phase string
	// Words is what every one of them says, which is the whole of why they
	// are grouped together. It is the handle a reader reads first, and what
	// something writing the rule would be given along with the sentences.
	Words []string
	// Said is every time it was said, oldest first. The sentences and not a
	// count: the words a person used are the rule, and a number is only the
	// reason to look at them.
	Said []Directive
}

// Times is how often it was said.
func (h Habit) Times() int { return len(h.Said) }

// Repeated is everything somebody has told runs often enough that it is a
// habit rather than a correction, newest habit first.
//
// It reads and decides nothing. What comes back is material: a person can
// look at it and write the rule themselves, and so can something else.
func Repeated(s *store.Store) ([]Habit, error) {
	told, err := directives(s)
	if err != nil {
		return nil, err
	}

	return habits(told), nil
}

// directives is every instruction a person gave a run, across every task the
// record holds.
func directives(s *store.Store) ([]Directive, error) {
	d, err := s.Record()
	if err != nil {
		return nil, err
	}

	ids, err := d.Tasks()
	if err != nil {
		return nil, err
	}

	var out []Directive

	for _, id := range ids {
		events, err := d.Events(id)
		if err != nil {
			return nil, fmt.Errorf("read what task %s was told: %w", id, err)
		}

		out = append(out, toldIn(id, events)...)
	}

	return out, nil
}

// toldIn is the directives in one task's events, with the checkout and the
// phase each of them was said in.
//
// The phase is the last one that started and not the last one still running.
// Somebody who pauses a run and then types is still talking about the work
// that was in front of them, and a directive filed under no phase at all
// would lose exactly the case this is for.
func toldIn(task string, events []record.Event) []Directive {
	var (
		out   []Directive
		where string
		phase string
	)

	for _, e := range events {
		switch e.Kind {
		case record.TaskCreated:
			where = e.Data["path"]
		case record.PhaseStarted:
			phase = e.Phase
		case record.TaskDialogue:
			if !aPersonSaid(e) || aboutTheFuture(e.Text) {
				continue
			}

			out = append(out, Directive{
				At: e.At, Task: task, Repo: where, Phase: phase, Text: e.Text,
			})
		}
	}

	return out
}

// throughATool is what the record says when a directive came in as a tool
// call, which is a model driving Orbit and not a person.
//
// It is the record's own vocabulary rather than this package's, the way
// itsOwn is. What is being looked for here is what a person does without
// noticing, and a model repeating itself is not that.
const throughATool = "mcp"

// aPersonSaid says whether a directive came from somebody rather than from
// Orbit's own loop or from a model driving it.
//
// A directive with nobody's name on it counts as a person: the way round
// that fails safe is a row somebody looks at, not one quietly dropped.
func aPersonSaid(e record.Event) bool {
	by := e.Data["by"]

	return by != itsOwn && by != throughATool
}

// where is one checkout and one phase of it: the two things that have to
// match before two sentences can be the same rule.
type where struct {
	repo  string
	phase string
}

// habits is the directives grouped: by checkout and phase first, because the
// same words said in two places are two rules, and then by what they say.
func habits(told []Directive) []Habit {
	byPlace := map[where][]Directive{}

	for _, one := range told {
		at := where{repo: one.Repo, phase: one.Phase}
		byPlace[at] = append(byPlace[at], one)
	}

	places := make([]where, 0, len(byPlace))
	for at := range byPlace {
		places = append(places, at)
	}

	// Sorted before anything is read off them, because a map is walked in a
	// different order every time and a listing that reshuffles itself
	// between two readings is one nobody trusts.
	sort.Slice(places, func(i, j int) bool {
		if places[i].repo != places[j].repo {
			return places[i].repo < places[j].repo
		}

		return places[i].phase < places[j].phase
	})

	var out []Habit

	for _, at := range places {
		out = append(out, alike(byPlace[at])...)
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].Times() > out[j].Times() })

	return out
}
