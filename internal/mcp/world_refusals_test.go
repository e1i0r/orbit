package mcp

// What the server a model speaks to will not do.
//
// Three refusals, and all three read zero. They are the line between an
// agent that writes down what it learned and an agent that edits its own
// instructions: a model that could change where a rule stands could quietly
// clear away the rules it keeps running into, and one that could ask for a
// cold reading would be making Orbit pay for another model on nobody's
// say-so.
//
// Each refusal says what to run instead, because a tool call is quoted back
// to whoever is watching, and "no" on its own leaves them looking for a bug.

import (
	"context"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/task"
)

// theServer is the world a tool call reaches the machine through.
func theServer(t *testing.T) world {
	t.Helper()

	s, _ := newRoot(t)

	t.Cleanup(func() { _ = s.Close() }) //nolint:errcheck // the test is over

	return Session{}.world(&storeAndBoard{store: s})
}

// TestAModelIsNotAllowedToDecideForAPerson. Each of these is a person's
// decision, and the refusal names the command that is theirs to run.
func TestAModelIsNotAllowedToDecideForAPerson(t *testing.T) {
	f := knowledge.Rule{ID: "aaaa1111", Phrase: "never log a card number"}

	refusals := []struct {
		name  string
		try   func(world) error
		wants []string
	}{
		{
			name:  "opening a pull request",
			try:   func(w world) error { _, err := w.Deliver(context.Background(), task.Task{}, "pr"); return err },
			wants: []string{"person's decision", "orbit pr"},
		},
		{
			name:  "reading what you keep saying",
			try:   func(w world) error { _, err := w.Ask(context.Background(), "", "what do I keep saying"); return err },
			wants: []string{"spends money", "orbit rules draft"},
		},
		{
			name:  "changing where a rule stands",
			try:   func(w world) error { return w.Replace(f, f, learn.Turn{}) },
			wants: []string{"person's decision", "orbit rules"},
		},
		{
			name:  "removing a rule",
			try:   func(w world) error { return w.Forget(f) },
			wants: []string{"person's decision", "orbit rules forget"},
		},
	}

	for _, r := range refusals {
		t.Run(r.name, func(t *testing.T) {
			w := theServer(t)

			err := r.try(w)
			if err == nil {
				t.Fatalf("%s was allowed", r.name)
			}

			for _, want := range r.wants {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("the refusal is %q, want %q in it", err, want)
				}
			}
		})
	}
}

// TestARefusedRuleChangeLeavesTheRuleWhereItWas. The refusal is not a
// message: nothing moves, so an agent that ignores the sentence and asks
// again gets the same answer and the same rules.
func TestARefusedRuleChangeLeavesTheRuleWhereItWas(t *testing.T) {
	w := theServer(t)

	was := knowledge.Rule{
		Scope:  knowledge.Scope{Kind: knowledge.General},
		Source: knowledge.Human,
		Phrase: "never log a card number",
	}

	if err := w.Learn(was); err != nil {
		t.Fatalf("write down what the model learned: %v", err)
	}

	before, err := w.Facts()
	if err != nil {
		t.Fatalf("read the rules: %v", err)
	}

	if len(before) != 1 {
		t.Fatalf("the model wrote %d rules, want the one it learned", len(before))
	}

	now := before[0]
	now.State = knowledge.Off

	if err := w.Replace(before[0], now, learn.Turn{}); err == nil {
		t.Error("a model switched a rule off")
	}

	if err := w.Forget(before[0]); err == nil {
		t.Error("a model removed a rule")
	}

	after, err := w.Facts()
	if err != nil {
		t.Fatalf("read the rules again: %v", err)
	}

	if len(after) != 1 || after[0].State != before[0].State {
		t.Errorf("the rules read back as %+v, want the one that was there untouched", after)
	}
}

// TestWhatAModelLearnsIsWrittenDownUnderANameOfItsOwn, so that what happens
// to it afterwards can be followed — and so the record says a model wrote it
// rather than leaving it indistinguishable from a person's.
func TestWhatAModelLearnsIsWrittenDownUnderANameOfItsOwn(t *testing.T) {
	w := theServer(t)

	one := knowledge.Rule{
		Scope:  knowledge.Scope{Kind: knowledge.General},
		Source: knowledge.FromCode,
		Phrase: "the fuzz corpus lives in testdata",
	}

	if err := w.Learn(one); err != nil {
		t.Fatalf("write it down: %v", err)
	}

	facts, err := w.Facts()
	if err != nil {
		t.Fatalf("read the rules: %v", err)
	}

	if len(facts) != 1 || facts[0].ID == "" {
		t.Fatalf("it was written down as %+v, want a rule with a name", facts)
	}

	turns, err := learn.History(w.sb.store, facts[0].ID)
	if err != nil {
		t.Fatalf("read what happened to it: %v", err)
	}

	if len(turns) != 1 || turns[0].What != learn.Written {
		t.Fatalf("the record says %+v, want the one turn that wrote it", turns)
	}

	if turns[0].By != learn.AModel {
		t.Errorf("the record says %q wrote it, want the model", turns[0].By)
	}
}

// TestARuleAModelCannotStandBehindIsRefused. Validate runs before anything
// is written, because a rule with no sentence in it is a file on somebody's
// disk that says nothing and cannot be answered.
func TestARuleAModelCannotStandBehindIsRefused(t *testing.T) {
	w := theServer(t)

	if err := w.Learn(knowledge.Rule{}); err == nil {
		t.Fatal("a rule with nothing in it was written down")
	}

	facts, err := w.Facts()
	if err != nil {
		t.Fatalf("read the rules: %v", err)
	}

	if len(facts) != 0 {
		t.Errorf("a refused rule was written anyway: %+v", facts)
	}
}
