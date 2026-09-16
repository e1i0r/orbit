package verb

// Where your rules stand, and moving one.
//
// One listing and not two. Without a filter it is what needs an answer from
// you — the sentences nobody has decided about, and the rules somebody sent
// to be looked at again — because that is what you came to the screen for.
// With one it is the rules standing wherever you asked about. Two screens
// would be two to remember to look at.

import (
	"errors"
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/words"
)

// standing is the states a reader can ask about, in the words they type.
//
// review is on this list and is not one of the three a rule stands in: it is
// the other question — whether the rule is waiting for a decision — and a
// reader asking "what do I have to look at" means that one. Keeping them in
// one vocabulary is how that stays one flag to remember rather than two.
var standing = map[string]knowledge.State{
	"active": knowledge.Active,
	"paused": knowledge.Paused,
	"off":    knowledge.Off,
}

// inReview is what a reader asks for when they mean the rules waiting for
// them rather than a state a rule stands in.
const inReview = "review"

// where lists the rules standing wherever the reader asked.
func where(w World, asked string) (Out, error) {
	if _, known := standing[asked]; !known && asked != inReview {
		return Out{}, errors.New(w.Words().T("verb.rules.no_such_state",
			"{state} is not a state a rule can be in; they are active, paused, off and review",
			words.Arg{Name: "state", Value: asked}))
	}

	facts, err := w.Facts()
	if err != nil {
		return Out{}, err
	}

	var kept []knowledge.Rule

	for _, f := range knowledge.Every(facts) {
		if (asked == inReview && f.Review) || (asked != inReview && f.State == standing[asked]) {
			kept = append(kept, f)
		}
	}

	if len(kept) == 0 {
		return Out{Said: w.Words().T("verb.rules.none_standing", "no rule is {state}",
			words.Arg{Name: "state", Value: asked}), Saw: kept}, nil
	}

	return Out{Said: ruleRows(kept), Saw: kept}, nil
}

// ruleRows is the rules, one to a line, with the reason under a paused one.
//
// The reason gets its own line rather than a column, because it is a
// sentence somebody wrote and a sentence in a column is a sentence cut off.
func ruleRows(facts []knowledge.Rule) string {
	var b strings.Builder

	for _, f := range facts {
		fmt.Fprintf(&b, "%-8s %-20s %s\n", f.ID, scopeOf(f), f.Phrase)

		if f.Why != "" {
			fmt.Fprintf(&b, "%-8s %s\n", "", f.Why)
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

// paused stops a rule applying, and sends it to be looked at again.
//
// The reason is not optional, and that is the whole difference between this
// and a switch: "skip this one while we get the coverage up" is what you
// will read when you come back, and the only thing that will tell you
// whether it made sense or you were wrong.
func paused(w World, in In) (Out, error) {
	was, err := ruleNamed(w, in)
	if err != nil {
		return Out{}, err
	}

	why := strings.TrimSpace(in.Arg("why"))

	now := was
	now.State, now.Why, now.Review = knowledge.Paused, why, true

	// Where it was in the way, when it was. A pause typed from a terminal
	// is about the rule and about no run; one taken while a task sat
	// blocked is the beginning of a pattern, and the pattern is what makes
	// a rule worth reconsidering.
	at, phase := whileWorkingOn(w, in)

	where := learn.Turn{By: learn.Operator, Task: at, Phase: phase}
	if err := w.Replace(was, now, where); err != nil {
		return Out{}, err
	}

	return Out{Said: w.Words().T("verb.rules.paused", "{rule} is paused: {why}",
		words.Arg{Name: "rule", Value: was.ID},
		words.Arg{Name: "why", Value: why})}, nil
}

// whileWorkingOn is the task a pause was taken at and the phase it was
// stopped in, and nothing when the reader named no task.
//
// The phase is read the way a skip reads it: the last refusal in the record
// is the one the run is waiting at. A task named with no refusal behind it
// still says which task, which is most of the answer.
func whileWorkingOn(w World, in In) (at, phase string) {
	named := strings.TrimSpace(in.Arg("task"))
	if named == "" {
		return "", ""
	}

	t, _, err := w.Find(named, in.Repo)
	if err != nil {
		return named, ""
	}

	_, phase = inTheWay(w, t)

	return named, phase
}

// resumed has a rule apply again and stops asking about it, which is the
// answer to both ways it got here: paused on purpose, or skipped once and
// left waiting.
func resumed(w World, in In) (Out, error) {
	was, err := ruleNamed(w, in)
	if err != nil {
		return Out{}, err
	}

	now := was
	now.State, now.Why, now.Review = knowledge.Active, "", false

	if err := w.Replace(was, now, learn.Turn{By: learn.Operator}); err != nil {
		return Out{}, err
	}

	return Out{Said: w.Words().T("verb.rules.resumed", "{rule} applies again",
		words.Arg{Name: "rule", Value: was.ID})}, nil
}

// ruleNamed is the rule a reader meant by its name.
//
// A rule somebody wrote by hand has no name, so it cannot be reached this
// way — and the refusal says where the names are printed rather than only
// that this one was not found.
func ruleNamed(w World, in In) (knowledge.Rule, error) {
	name := strings.TrimSpace(in.Arg("rule"))

	facts, err := w.Facts()
	if err != nil {
		return knowledge.Rule{}, err
	}

	for _, f := range facts {
		if f.ID != "" && f.ID == name {
			return f, nil
		}
	}

	return knowledge.Rule{}, errors.New(w.Words().T("verb.rules.no_such_name",
		"there is no rule called {rule}; orbit knowledge prints their names",
		words.Arg{Name: "rule", Value: name}))
}

// unanswered is everything that wants a decision from you: the sentences
// nobody has answered, and the rules that were sent to be looked at again.
//
// Both under one command because they are one question — "what do I have to
// decide" — asked of a person who has just sat down.
func unanswered(w World) (Out, error) {
	said, err := learn.Waiting(w.Store())
	if err != nil {
		return Out{}, err
	}

	facts, err := w.Facts()
	if err != nil {
		return Out{}, err
	}

	var again []knowledge.Rule

	for _, f := range knowledge.Every(facts) {
		if f.Review {
			again = append(again, f)
		}
	}

	if len(said) == 0 && len(again) == 0 {
		return Out{Said: w.Words().T("verb.rules.nothing_waiting",
			"nothing is waiting for an answer from you")}, nil
	}

	return Out{Said: bothHalves(said, again), Saw: said}, nil
}

// bothHalves is the two lists under one another, with a blank line between
// them: the sentences are answered by their number and the rules by their
// name, and running them together would leave a reader guessing which.
func bothHalves(said []learn.Said, again []knowledge.Rule) string {
	parts := make([]string, 0, 2)
	if len(said) > 0 {
		parts = append(parts, numbered(said))
	}

	if len(again) > 0 {
		parts = append(parts, ruleRows(again))
	}

	return strings.Join(parts, "\n\n")
}
