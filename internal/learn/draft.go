package learn

// Asking a model to write down what somebody keeps saying.
//
// This is the one place in Orbit where a model decides anything, and it is
// here for a reason nothing else in this package needs. Whether one sentence
// is a rule is a question about its shape — a hand-written list of openings
// answers it, costs nothing and never surprises anybody. That is rule.go and
// it stays as it is.
//
// What no list can do is look at four different sentences and see what they
// have in common:
//
//	"add fuzz testing"  ·  "edge cases are missing"  ·  "try it with empty
//	input"  ·  "get the coverage up here"
//
// Nobody writes a word list that groups those four. Here a cheap model earns
// its call.
//
// It is shown the sentences and nothing else — not the code, not the diff,
// not the record. It is a classification and not an analysis, which is what
// keeps it cheap, and what it answers with goes in the tray like everything
// else, to be agreed with or dropped.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/store"
)

// FromAHabit is what By says when a sentence is not something somebody said
// but something they kept saying, written out by a model.
//
// It is neither of the other two. A person did not say this rule — they said
// the four sentences it was drawn from — and no engine found it mid-task
// either: what it knows is what somebody typed, put into one sentence.
const FromAHabit = "habit"

// topics are the kinds of thing a rule can be about, written by hand.
//
// The model chooses from this list and never invents a name. Left to invent
// them it would answer "tests" once, "testing" the next time and "test
// coverage" the time after, and two readings of the same person would not
// add up to anything. A kind that keeps being needed is added here, by
// somebody, on purpose.
var topics = []string{"testing", "style", "dependencies", "security", "process"}

// atMost is how many rules one habit may produce.
//
// Three, by hand. A model asked what four sentences have in common can
// always find a fourth thing to say, and a tray that fills with weak
// proposals is a tray somebody stops opening — which costs more than the
// rule that was missed.
const atMost = 3

// atOnce is how many habits are asked about in one go.
//
// Five, most-repeated first, because this is the first money Orbit spends
// without somebody starting a task. What was said most is what is most worth
// asking about, and the rest are still there next time.
const atOnce = 5

// An Ask is something that can put one short question to a model and answer
// with what it said.
//
// A port and not an engine. This package writes what Orbit knows; starting
// processes, choosing which engine and paying for it belong to whoever calls
// this, and a way in that has no engine says so rather than pretending.
type Ask func(ctx context.Context, question string) (string, error)

// Draft asks a model to write down what somebody keeps telling runs, and
// puts every rule it answers with in the tray.
//
// It answers with what was offered, so the caller can say how much. Nothing
// is applied: these wait to be agreed with exactly as a sentence somebody
// typed does.
func Draft(ctx context.Context, s *store.Store, ask Ask) ([]Said, error) {
	if ask == nil {
		return nil, fmt.Errorf("there is no engine here to read what you keep saying")
	}

	habits, err := Repeated(s)
	if err != nil {
		return nil, err
	}

	d, err := s.Record()
	if err != nil {
		return nil, err
	}

	var (
		out  []Said
		went int
	)

	for _, h := range habits {
		if went >= atOnce {
			break
		}

		answered, err := d.Answered(h.Handle())
		if err != nil {
			return out, err
		}

		if answered {
			continue
		}

		went++

		said, err := drafted(ctx, s, ask, h)
		if err != nil {
			return out, err
		}

		out = append(out, said...)
	}

	return out, nil
}

// drafted is the rules one habit turned into, put in the tray.
func drafted(ctx context.Context, s *store.Store, ask Ask, h Habit) ([]Said, error) {
	answer, err := ask(ctx, question(h))
	if err != nil {
		return nil, fmt.Errorf("reading what you keep saying: %w", err)
	}

	// Two rules written in one breath need two moments, because the moment
	// is what a row in the tray is known by. A nanosecond apart is enough
	// and keeps them in the order the model wrote them.
	at := time.Now().UTC()

	var out []Said

	for _, rule := range rulesIn(answer) {
		one := Said{
			At: at, Text: rule.phrase, By: FromAHabit,
			Repo: h.Repo, Topic: rule.topic, Habit: h.Handle(),
		}

		if err := Propose(s, one); err != nil {
			return out, err
		}

		out = append(out, one)
		at = at.Add(time.Nanosecond)
	}

	return out, nil
}

// question is what the model is shown: the sentences, and what to do with
// them.
//
// Their sentences and nothing around them. What is being asked is what these
// have in common, and the code they were said about would only invite the
// model to write a rule nobody asked for.
func question(h Habit) string {
	var b strings.Builder

	b.WriteString("Someone keeps telling coding agents the same thing. " +
		"Here is what they said, in their own words:\n\n")

	for _, one := range h.Said {
		fmt.Fprintf(&b, "- %s\n", one.Text)
	}

	b.WriteString("\nWrite the standing rule this amounts to, in their words and in " +
		"their language, as close to what they actually said as you can.\n\n" +
		"Answer with one line per rule, and nothing else:\n\n" +
		"  <topic> | <the rule, in one sentence>\n\n" +
		"The topic must be one of: " + strings.Join(topics, ", ") + ".\n\n" +
		"Write nothing at all if these do not amount to a rule. More than one " +
		"line is fine if they amount to more than one rule. Do not explain " +
		"yourself and do not write anything but those lines.")

	return b.String()
}

// A rule is one line of what the model answered.
type rule struct {
	topic  string
	phrase string
}

// rulesIn reads the model's answer, keeping the lines that are rules and
// dropping everything else.
//
// Dropping and not failing. A model that explains itself first, bullets its
// answer, or invents a topic has not broken anything — what it wrote that
// cannot be read is simply not offered, and a run that answered nothing
// usable is a run that proposed nothing.
func rulesIn(answer string) []rule {
	var out []rule

	for _, line := range strings.Split(answer, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "-*• \t")

		topic, phrase, found := strings.Cut(line, "|")
		if !found {
			continue
		}

		topic = strings.ToLower(strings.TrimSpace(topic))
		phrase = strings.TrimSpace(phrase)

		if !aTopic(topic) || phrase == "" || len(phrase) > aRule {
			continue
		}

		out = append(out, rule{topic: topic, phrase: phrase})

		if len(out) == atMost {
			break
		}
	}

	return out
}

// aTopic says whether the model chose from the list rather than inventing.
func aTopic(named string) bool {
	for _, one := range topics {
		if one == named {
			return true
		}
	}

	return false
}
