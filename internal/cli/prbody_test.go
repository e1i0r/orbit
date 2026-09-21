package cli

// The pull request a delivery opens, as whoever reviews it meets it.
//
// This body is read on GitHub by somebody who was not at the terminal. What
// it says about the rest of the task is the only thing telling them there is
// a rest — a task that reached into three repositories opens three pull
// requests, and a reviewer looking at one of them has no other way to find
// the other two.

import (
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/words"
)

// pull is a pull request this delivery opened, or a repository it left as
// it found it.
//
// Named for what it answers rather than for what happened to it: the
// cockpit's own suite has an `opened` of its own, behind the integration
// tag, so two files of this package declared the same name and the package
// only failed to build when that tag was on — which is make check and not
// make test.
func pull(name, url string) aPullRequest {
	return aPullRequest{repo: repo.Repo{Name: name}, url: url}
}

// TestAPullRequestSaysWhatElseTheTaskTouched.
func TestAPullRequestSaysWhatElseTheTaskTouched(t *testing.T) {
	one := task.Task{ID: "ACME-1", Text: "retry the webhook on 5xx"}

	const heading = "### The rest of this task"

	// A task worked in one repository has no rest, and a heading over
	// nothing is a reviewer looking for the other two.
	alone := bodyOf(one, nil, nil)
	if strings.Contains(alone, heading) {
		t.Errorf("a task in one repository heads the section over nothing:\n%s", alone)
	}

	if !strings.Contains(alone, "ACME-1") || !strings.Contains(alone, "retry the webhook on 5xx") {
		t.Errorf("the body does not say what the task is:\n%s", alone)
	}

	// And one worked in three names the other two, with the link for the
	// one that changed something and a word for the one that did not.
	spread := bodyOf(one, []aPullRequest{
		pull("ledger", "https://github.com/e1i0r/ledger/pull/7"),
		pull("scripts", ""),
	}, nil)

	if !strings.Contains(spread, heading) {
		t.Errorf("a task in three repositories does not say so:\n%s", spread)
	}

	for _, want := range []string{
		"`ledger` — https://github.com/e1i0r/ledger/pull/7",
		"`scripts` — joined, nothing to change",
	} {
		if !strings.Contains(spread, want) {
			t.Errorf("the body does not carry %q:\n%s", want, spread)
		}
	}
}

// TestWhatTheOtherRepositoriesAre.
//
// Every repository of the task except the one this pull request is for, in
// the order they joined it — a reviewer reading the list wants the same
// order in every one of the three bodies, or they cannot tell they are
// looking at the same task.
func TestWhatTheOtherRepositoriesAre(t *testing.T) {
	all := []aPullRequest{pull("api", "u1"), pull("ledger", "u2"), pull("scripts", "u3")}

	for i, want := range [][]string{
		{"ledger", "scripts"},
		{"api", "scripts"},
		{"api", "ledger"},
	} {
		got := siblings(all, i)
		if len(got) != len(want) {
			t.Fatalf("the siblings of %d are %d, want %d", i, len(got), len(want))
		}

		for j, name := range want {
			if got[j].repo.Name != name {
				t.Errorf("sibling %d of %d is %q, want %q", j, i, got[j].repo.Name, name)
			}
		}
	}

	// The list it was given is not the list it hands back: a caller that
	// appended to what it got would rewrite the delivery's own record of
	// what it opened.
	if grown := append(siblings(all, 0), pull("intruder", "u4")); len(grown) != 3 {
		t.Fatalf("two siblings and one more came to %d", len(grown))
	}

	if all[1].repo.Name != "ledger" || all[2].repo.Name != "scripts" || len(all) != 3 {
		t.Errorf("appending to the siblings changed the list they came from: %+v", all)
	}
}

// TestARuleKeptWithoutANameIsGivenOne.
//
// The name is coined here rather than left to Save, because what happened to
// the rule is written down in the same breath as the rule and cannot be told
// the name afterwards. Every rule's history starts with somebody keeping it,
// however they kept it — so a rule that arrived without a name and one that
// arrived with its own both end up with a history under the name they keep.
func TestARuleKeptWithoutANameIsGivenOne(t *testing.T) {
	root := t.TempDir()

	s, err := store.New(root)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	t.Cleanup(func() { _ = s.Close() })

	w := newWorld(s, nil, words.For(""))

	bare := knowledge.Rule{
		Phrase: "amounts are always in cents",
		Source: knowledge.Human,
		Scope:  knowledge.Scope{Kind: knowledge.General},
		At:     time.Now().UTC(),
	}

	if err := w.Learn(bare); err != nil {
		t.Fatalf("Learn: %v", err)
	}

	kept, err := knowledge.NewStore(root).Load("")
	if err != nil || len(kept) != 1 {
		t.Fatalf("read it back: %d rules, %v", len(kept), err)
	}

	if kept[0].ID == "" {
		t.Fatal("a rule kept without a name still has none")
	}

	// And its history is under that name, which is the whole reason the
	// name is coined here.
	turns, err := learn.History(s, kept[0].ID)
	if err != nil {
		t.Fatalf("read what happened to it: %v", err)
	}

	if len(turns) != 1 || turns[0].What != learn.Written {
		t.Fatalf("what happened to it reads as %+v", turns)
	}

	// A rule that arrived with a name keeps it: it is the name every other
	// reading of that rule already uses.
	named := bare
	named.ID = "abcd1234"
	named.Phrase = "the commits are written in English"

	if err := w.Learn(named); err != nil {
		t.Fatalf("Learn: %v", err)
	}

	again, err := learn.History(s, "abcd1234")
	if err != nil {
		t.Fatalf("read what happened to it: %v", err)
	}

	if len(again) != 1 {
		t.Errorf("a rule kept under its own name has %d turns of history", len(again))
	}
}
