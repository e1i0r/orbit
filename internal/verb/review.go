package verb

// What a rule has put you through, said in words rather than counted.
//
// A score was considered and thrown out, for one reason: a rule that works
// perfectly never stops anything. The model reads it and obeys it. So "it
// refused work zero times" means two opposite things and no number tells
// them apart.
//
// A rule is good until it annoys you. Silence is the good case and is not
// measured; what is written down is the friction, and the friction is all
// gestures of yours.
//
// And there is a case no score would have understood: you asked for 90%
// coverage and the repository has never been past 80. The rule is not wrong
// — it arrived early. Only you know that, which is why this shows and does
// not decide.

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/words"
)

// reviewed is one rule with everything that has happened to it, in the words
// somebody decides by.
func reviewed(w World, in In) (Out, error) {
	f, err := ruleNamed(w, in)
	if err != nil {
		return Out{}, err
	}

	turns, err := learn.History(w.Store(), f.ID)
	if err != nil {
		return Out{}, err
	}

	var b strings.Builder

	fmt.Fprintf(&b, "%-8s %-20s %s\n", f.ID, scopeOf(f), f.Phrase)

	if f.Why != "" {
		fmt.Fprintf(&b, "%-8s %s\n", "", f.Why)
	}

	b.WriteString("\n")

	for _, line := range Story(w.Words(), f, turns) {
		fmt.Fprintf(&b, "  %s\n", line)
	}

	return Out{Said: strings.TrimRight(b.String(), "\n"), Saw: turns}, nil
}

// Story is what happened to a rule, one sentence at a time.
//
// Sentences and not a table, because what a reader is doing here is deciding
// whether to keep something — and "it refused work four times in the test
// phase and you got past it every time" is an argument, where a row of four
// is a number they have to make one out of.
//
// Exported because the window shows the same sentences. What a rule has put
// you through is one reading, and two surfaces telling it two ways would be
// two accounts of the same evidence — which is exactly the thing a person is
// about to decide on.
func Story(p *words.Printer, f knowledge.Fact, turns []learn.Turn) []string {
	var told []string

	if kept := first(turns, learn.Written); kept != nil {
		told = append(told, p.T("verb.rules.review.kept", "you kept it on {when}",
			words.Arg{Name: "when", Value: kept.At.Local().Format("2 January")}))
	}

	told = append(told, friction(p, turns)...)

	if paused := last(turns, learn.Paused); paused != nil {
		told = append(told, p.T("verb.rules.review.paused", "you paused it on {when}: {why}",
			words.Arg{Name: "when", Value: paused.At.Local().Format("2 January")},
			words.Arg{Name: "why", Value: paused.Was}))
	}

	if len(told) == 0 || (len(told) == 1 && f.Tells()) {
		told = append(told, p.T("verb.rules.review.quiet",
			"it has not stopped anything since, which is what a rule that is working looks like"))
	}

	return told
}

// friction is what the rule has cost you, one line per phase it happened in.
//
// By phase because that is the axis a pattern shows on: "I always get past
// this one in the test phase" is a pattern, and the same four refusals spread
// over four tasks are four accidents.
func friction(p *words.Printer, turns []learn.Turn) []string {
	refused, skipped := map[string]int{}, map[string]int{}
	where := []string{}

	for _, one := range turns {
		switch one.What {
		case learn.Failed:
			if refused[one.Phase] == 0 {
				where = append(where, one.Phase)
			}

			refused[one.Phase]++
		case learn.Skipped:
			skipped[one.Phase]++
		}
	}

	sort.Strings(where)

	told := make([]string, 0, len(where))
	for _, phase := range where {
		told = append(told, refusals(p, phase, refused[phase], skipped[phase]))
	}

	return told
}

// refusals is one phase's line: how often it stopped the work there, and how
// much of that you walked past.
func refusals(p *words.Printer, phase string, refused, skipped int) string {
	at := p.P("verb.rules.review.refused", refused,
		"it stopped the work once in {phase}", "it stopped the work {n} times in {phase}",
		words.Arg{Name: "phase", Value: phaseNamed(p, phase)})

	switch {
	case skipped == 0:
		return at + p.T("verb.rules.review.all_fixed", ", and every one was fixed")
	case skipped >= refused:
		return at + p.T("verb.rules.review.all_skipped", ", and you got past it every time")
	default:
		return at + p.P("verb.rules.review.some_skipped", skipped,
			", and you got past one of them", ", and you got past {n} of them")
	}
}

// phaseNamed is a phase to a reader, and the words for one that has no name —
// which is what a refusal recorded before phases were written down carries.
func phaseNamed(p *words.Printer, phase string) string {
	if phase == "" {
		return p.T("verb.rules.review.nowhere", "a phase nobody named")
	}

	return phase
}

// first and last are the turns that bracket a story: when it began, and what
// was last done to it.
func first(turns []learn.Turn, what string) *learn.Turn {
	for i := range turns {
		if turns[i].What == what {
			return &turns[i]
		}
	}

	return nil
}

func last(turns []learn.Turn, what string) *learn.Turn {
	for i := len(turns) - 1; i >= 0; i-- {
		if turns[i].What == what {
			return &turns[i]
		}
	}

	return nil
}

// corrected is the decision this screen exists for: the rule, said better,
// or narrowed to where it was actually true.
//
// Narrowing is the one that was almost always wanted. A rule that annoys you
// in `docs` and earns its keep in `payments` is not a rule to switch off —
// it is a rule about `payments` that somebody wrote too wide, and until it
// could be moved the only answer was to lose it.
//
// Whatever is not named is left alone. Correcting the sentence should not
// quietly drop the command that makes it stop the work.
func corrected(w World, in In) (Out, error) {
	was, err := ruleNamed(w, in)
	if err != nil {
		return Out{}, err
	}

	now := was
	now.State, now.Why, now.Review = knowledge.Active, "", false

	if text := strings.TrimSpace(in.Arg("text")); text != "" {
		now.Phrase = text
	}

	if check, said := in.Args["check"]; said {
		now.Check = strings.TrimSpace(check)
		now.Stops = now.Check != ""
	}

	if where, said := in.Args["in"]; said {
		if now.Scope, err = whereItGoes(w, was, where); err != nil {
			return Out{}, err
		}
	}

	if err := w.Replace(was, now, learn.Turn{By: learn.Operator}); err != nil {
		return Out{}, err
	}

	return Out{Said: w.Words().T("verb.rules.corrected", "{rule} now reads: {text}",
		words.Arg{Name: "rule", Value: was.ID},
		words.Arg{Name: "text", Value: now.Phrase})}, nil
}

// whereItGoes is the place a corrected rule now applies to: a folder or a
// file inside its own checkout, and the whole checkout for a dot.
//
// Its own checkout and no other. A rule moved into a repository it was never
// about is a rule steering code nobody meant it to, and the reader moving it
// is deciding how narrow it should be — not which project it belongs to.
func whereItGoes(w World, was knowledge.Fact, where string) (knowledge.Scope, error) {
	if was.Scope.Repo == "" {
		return knowledge.Scope{}, errors.New(w.Words().T("verb.rules.nowhere_to_narrow",
			"{rule} is about no checkout, so there is nothing for {path} to be inside",
			words.Arg{Name: "rule", Value: was.ID},
			words.Arg{Name: "path", Value: where}))
	}

	return knowledge.At(was.Scope.Repo, where)
}

// switchedOff is a rule you decided against, with everything it has put you
// through in front of you.
//
// It stays and stops being told. Disagreeing with a rule and losing the
// record that it existed are different things, and this is the one place the
// first is offered — because it is the one place the second is impossible to
// do by accident.
func switchedOff(w World, in In) (Out, error) {
	was, err := ruleNamed(w, in)
	if err != nil {
		return Out{}, err
	}

	now := was
	now.State, now.Why, now.Review = knowledge.Off, "", false

	if err := w.Replace(was, now, learn.Turn{By: learn.Operator}); err != nil {
		return Out{}, err
	}

	return Out{Said: w.Words().T("verb.rules.switched_off",
		"{rule} is off; it stays where it is and nothing is told it",
		words.Arg{Name: "rule", Value: was.ID})}, nil
}
