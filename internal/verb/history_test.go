package verb

// What the history says this project does, offered as rules.
//
// Nothing here was tested: every function of history.go read zero, which is
// a whole reading of a checkout — the one that spends nothing and is the
// only source that can disagree with what a project wrote about itself.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/learn"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/words"
)

// TestAReadingSaysTheCountThatBacksIt. The count is inside the sentence and
// not beside it, because the sentence is what the agent reads: it has to say
// both what to do and that the project means it.
func TestAReadingSaysTheCountThatBacksIt(t *testing.T) {
	p := words.For("en")

	cases := []struct {
		name  string
		one   repo.Custom
		wants []string
		topic string
		path  string
	}{
		{
			name:  "a folder whose changes bring a test",
			one:   repo.Custom{Kind: repo.TestsTravel, Where: "internal/db", Times: 37, Of: 40},
			wants: []string{"internal/db", "37", "40", "test"},
			topic: "testing",
			path:  "internal/db",
		},
		{
			name:  "commit subjects that keep a form",
			one:   repo.Custom{Kind: repo.MessagesKeepAShape, Where: "a prefix", Times: 45, Of: 50},
			wants: []string{"a prefix", "45", "50"},
			topic: "process",
			// A reading about how the project is worked on is about the
			// whole checkout, so it names no folder.
			path: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			said := asARule(p, c.one)
			for _, want := range c.wants {
				if !strings.Contains(said, want) {
					t.Errorf("the reading is %q, want %q in it", said, want)
				}
			}

			if got := topicOf(c.one); got != c.topic {
				t.Errorf("it is filed under %q, want %q", got, c.topic)
			}

			if got := alongsideIt(c.one); got != c.path {
				t.Errorf("it applies at %q, want %q", got, c.path)
			}
		})
	}
}

// TestWhatTheProjectDoesIsSaidWhetherItHeldOrNot. Both halves on purpose: a
// measurement that held backs a written rule, and one that was taken and did
// not hold is what says a written rule is no longer true.
func TestWhatTheProjectDoesIsSaidWhetherItHeldOrNot(t *testing.T) {
	customs := []repo.Custom{
		{Kind: repo.TestsTravel, Where: "internal/db", Times: 37, Of: 40},
		{Kind: repo.TestsTravel, Where: "internal/ui", Times: 2, Of: 40},
	}

	said := whatItActuallyDoes(words.For("en"), customs)
	if len(said) != 2 {
		t.Fatalf("it said %d things, want one per reading whether it held or not", len(said))
	}

	if !strings.Contains(said[1], "2") || !strings.Contains(said[1], "internal/ui") {
		t.Errorf("the reading that did not hold reads %q, want it to say so with its count", said[1])
	}
}

// TestTheTwoListingsPutTheirOwnColumnFirst. A cold reading leads with what
// kind of thing it is; a gate leads with the command, because the command is
// the whole of why a gate is worth anything — every other source leaves a
// reader deciding what the gate would be.
func TestTheTwoListingsPutTheirOwnColumnFirst(t *testing.T) {
	said := []learn.Said{
		{Topic: "testing", About: "the last 40 commits", Text: "a change here comes with a test"},
		{Topic: "process", About: "CONTRIBUTING.md", Text: "commits are written in English"},
	}

	cold := coldRows(said)
	if lines := strings.Split(cold, "\n"); len(lines) != 2 {
		t.Fatalf("the cold listing is %d lines, want one per reading", len(lines))
	}

	if !strings.HasPrefix(cold, "testing") {
		t.Errorf("the cold listing starts %q, want the topic first", cold)
	}

	if strings.HasSuffix(cold, "\n") {
		t.Errorf("the cold listing ends in a blank line: %q", cold)
	}

	gates := gateRows([]learn.Said{
		{Gate: "make check", Text: "make check has to pass"},
		{Gate: "golangci-lint run", Text: "golangci-lint run has to pass"},
	})

	if !strings.HasPrefix(gates, "make check") {
		t.Errorf("the gate listing starts %q, want the command first", gates)
	}

	if strings.HasSuffix(gates, "\n") {
		t.Errorf("the gate listing ends in a blank line: %q", gates)
	}
}

// TestWhatTheCheckoutAlreadyRefusesWorkOverIsOffered. It spends nothing and
// asks no model: a workflow either parses or it does not, and the sentence
// behind a gate is the command itself.
func TestWhatTheCheckoutAlreadyRefusesWorkOverIsOffered(t *testing.T) {
	w := worldOf(t)
	here := w.gitRepo(t, "acme")

	workflow := filepath.Join(here.Path, ".github", "workflows", "ci.yml")
	if err := os.MkdirAll(filepath.Dir(workflow), 0o750); err != nil {
		t.Fatalf("make the workflow directory: %v", err)
	}

	body := "name: ci\non: [pull_request]\njobs:\n  check:\n    steps:\n      - run: make check\n"
	if err := os.WriteFile(workflow, []byte(body), 0o600); err != nil {
		t.Fatalf("write the workflow: %v", err)
	}

	out := mustAsk(t, w, "rules enforced", In{Args: map[string]string{"repo": here.Path}})

	if !strings.Contains(out.Said, "make check") {
		t.Errorf("it said %q, want the command the pull request has to pass", out.Said)
	}

	offered := offers(t, out)
	if len(offered) == 0 {
		t.Fatal("it offered nothing")
	}

	if offered[0].Gate != "make check" {
		t.Errorf("the offer carries the gate %q, want make check", offered[0].Gate)
	}

	// Asked twice, it offers nothing the second time: the answer to a
	// sentence already put to somebody is that it was put to them.
	again := mustAsk(t, w, "rules enforced", In{Args: map[string]string{"repo": here.Path}})
	if left := offers(t, again); len(left) != 0 {
		t.Errorf("it offered %+v a second time", left)
	}
}

// offers is what a reading put in front of somebody. Out.Saw is the thing
// itself rather than the sentence about it, and every reading here puts the
// same shape in it.
func offers(t *testing.T, out Out) []learn.Said {
	t.Helper()

	if out.Saw == nil {
		return nil
	}

	said, ok := out.Saw.([]learn.Said)
	if !ok {
		t.Fatalf("a reading answered with %T, want the sentences it offered", out.Saw)
	}

	return said
}

// TestACheckoutThatRefusesNothingSaysSo. Nothing found is an answer, and it
// says where a rule here would have come from rather than leaving a reader
// with an empty listing and no idea what was looked at.
func TestACheckoutThatRefusesNothingSaysSo(t *testing.T) {
	w := worldOf(t)
	here := w.gitRepo(t, "acme")

	out := mustAsk(t, w, "rules enforced", In{Args: map[string]string{"repo": here.Path}})

	if offered := offers(t, out); len(offered) != 0 {
		t.Errorf("a checkout with no gates offered %+v", offered)
	}

	if !strings.Contains(out.Said, "pull request") {
		t.Errorf("it said %q, want it to say where a rule here comes from", out.Said)
	}
}

// worked is a checkout with enough history for a reading to be taken:
// changes under one folder that bring a test, and one that never does.
func worked(t *testing.T, at string) {
	t.Helper()

	run := func(args ...string) {
		t.Helper()

		cmd := exec.Command("git", args...)
		cmd.Dir = at

		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}

	write := func(name, body string) {
		t.Helper()

		path := filepath.Join(at, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("make %s: %v", filepath.Dir(path), err)
		}

		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	// Twenty-two, because a reading needs twenty commits behind it before
	// it says anything at all.
	for i := range 22 {
		body := strings.Repeat("x", i+1)

		write("internal/db/query.go", body)
		write("internal/db/query_test.go", body)

		run("add", "-A")
		run("commit", "-q", "-m", "db: change "+body)
	}
}

// TestAFolderThatAlwaysBringsATestIsOfferedAsARule. The half of a reading
// that no document can give: something nobody wrote down that the repository
// does without fail.
func TestAFolderThatAlwaysBringsATestIsOfferedAsARule(t *testing.T) {
	w := worldOf(t)
	here := w.gitRepo(t, "acme")
	worked(t, here.Path)

	customs, commits, err := here.Customs()
	if err != nil {
		t.Fatalf("read the history: %v", err)
	}

	if commits < 20 {
		t.Fatalf("the checkout has %d commits, want the twenty a reading needs", commits)
	}

	said, err := fromTheHistory(w, here, customs, commits)
	if err != nil {
		t.Fatalf("read what the history says: %v", err)
	}

	if len(said) == 0 {
		t.Fatalf("a folder whose every change brought a test was offered nothing: %+v", customs)
	}

	one := said[0]
	if one.By != learn.FromTheHistory {
		t.Errorf("the offer says it came from %q, want %q", one.By, learn.FromTheHistory)
	}

	if one.Repo != here.Path {
		t.Errorf("the offer is about %q, want the checkout it was read from", one.Repo)
	}

	if !strings.Contains(one.About, "commit") {
		t.Errorf("the offer is backed by %q, want it to name the commits it read", one.About)
	}

	// And the same reading is not offered again, whatever was answered
	// about it: offering a settled question is asking it twice.
	again, err := fromTheHistory(w, here, customs, commits)
	if err != nil {
		t.Fatalf("read it a second time: %v", err)
	}

	if len(again) != 0 {
		t.Errorf("the same reading was offered again: %+v", again)
	}
}

// TestAReadingThatDoesNotHoldIsNotOfferedAsARule. Four in five is the line,
// and a habit the project keeps half the time offered as a rule is Orbit
// starting to lie to the agent about what this project does.
func TestAReadingThatDoesNotHoldIsNotOfferedAsARule(t *testing.T) {
	w := worldOf(t)
	here := w.gitRepo(t, "acme")

	customs := []repo.Custom{{Kind: repo.TestsTravel, Where: "internal/ui", Times: 2, Of: 40}}

	said, err := fromTheHistory(w, here, customs, 40)
	if err != nil {
		t.Fatalf("read what the history says: %v", err)
	}

	if len(said) != 0 {
		t.Errorf("a habit kept two times in forty was offered as a rule: %+v", said)
	}
}
