package hunch

// The decision engine: what should happen to a run that has stopped, asked
// of something fast enough to answer while nobody is watching.
//
// A run stops far more often than it works. Across the tasks in one reader's
// record the engines were busy for fifty-six minutes and the tasks took
// twenty-five days: the difference is a task sitting at a gate, finished or
// half finished, waiting for a person to say carry on. That wait is what
// this is for. It is not a cheaper supervisor — the money was never the
// problem — it is a supervisor that is awake at four in the morning.
//
// What it answers is deliberately small: one of three words, and how sure it
// is. Sure is the whole design. A verdict nobody can put a number on is a
// verdict that has to be checked by a person, which is where this started;
// a verdict with a calibrated number can be acted on above a line the reader
// draws and handed up below it. Orbit acts on the sure ones, asks a full
// engine or a person about the rest, and writes both down.
//
// Nothing here writes prose. "Again" is worth nothing without a sentence
// saying what is missing, and that sentence is an engine's job — system one
// decides, system two writes, a person settles what neither can.

import "context"

// Choice is the whole vocabulary: three words, because there are three
// things that can be done with a run that stopped.
type Choice string

const (
	// Done is work that answers the task: let it stand, or let the run go on.
	Done Choice = "done"
	// Again is work another run could finish, which is a note and a rerun.
	Again Choice = "again"
	// Human is what no run settles: an ambiguous task, a wrong approach, a
	// decision somebody has to make.
	Human Choice = "human"
)

// Known is whether a word is one of the three. A model that answers
// something else is answering a question this does not know how to act on,
// and the caller treats that as no answer at all.
func (c Choice) Known() bool { return c == Done || c == Again || c == Human }

// Stop is what the decision is made from: the task as it was written, the
// phase that stopped, and what the run said before it did.
//
// This and nothing else. Not the diff — a decision that needs the diff is a
// decision for something that can read code, and that is the tier above.
// Not the record's own kinds either: "phase.failed" is the answer written
// on the question, and a supervisor that reads it learns nothing about
// whether the work is good.
type Stop struct {
	Task  string // the id, for the record rather than for the model
	Asked string // what the task asked for
	Phase string // the phase it stopped in
	Said  string // the last thing the run said
}

// Verdict is one answer and how sure it was, with the model that gave it so
// the record says who decided.
type Verdict struct {
	Choice     Choice
	Confidence float64
	Model      string
}

// Port is how a run asks. A nil Port is no decision engine at all, which is
// what every run had before there was one and what a machine with no key
// still has: see Off.
type Port interface {
	Decide(ctx context.Context, about Stop) (Verdict, error)
}

// Off is the port that answers nothing, for a machine that has not been
// given a key and for every test that is not about deciding.
type Off struct{}

// Decide answers with an unknown verdict, which every caller reads as "ask
// somebody else".
func (Off) Decide(context.Context, Stop) (Verdict, error) { return Verdict{}, nil }

// Acts says whether a verdict may be acted on without asking anybody: it is
// one of the three words, and the model was surer about it than the floor
// the reader set.
//
// The floor is a percentage because that is how the reader sets it — the
// same unit quota-floor uses, and for the same reason: a number somebody
// types has to be a number they can reason about.
func (v Verdict) Acts(floor int) bool {
	return v.Choice.Known() && v.Confidence*100 >= float64(floor)
}
