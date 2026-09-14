package repo

// What the history says this project actually does.
//
// The documents say what somebody wanted. The commits say what the team
// keeps doing. If the CONTRIBUTING says every change comes with tests and
// half of last year's commits brought none, that is not a rule — it is
// something somebody wrote once and nobody held to, and offering it is
// starting to lie to the agent.
//
// The other way round too: there are things nobody ever wrote down that the
// repository does without fail. That is knowledge in no document at all.
//
// This reads git and nothing else. No language, no parser, no build, no
// model: a repository of Python and one of Go answer the same way.

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// What the history can say. Each is a measurement and not an opinion: the
// numbers travel with it so that whoever reads it can disagree.
const (
	// TestsTravel is a folder whose changes come with a change to a test.
	TestsTravel = "tests travel"
	// MessagesKeepAShape is the subjects of commits following one form —
	// `feat:`, `fix:`, a ticket key at the front.
	MessagesKeepAShape = "messages keep a shape"
)

// A Custom is one thing the history says this project does, and the count
// that says it.
type Custom struct {
	// Kind is which of the readings above this is.
	Kind string
	// Where is the folder it is about, or the shape the messages keep, and
	// empty when the reading is about the whole repository.
	Where string
	// Times is how often it held and Of is how many times it could have.
	Times int
	Of    int
}

// Ratio is how often it held, between 0 and 1.
func (c Custom) Ratio() float64 {
	if c.Of == 0 {
		return 0
	}

	return float64(c.Times) / float64(c.Of)
}

// Holds says whether this is something the project does rather than
// something it does sometimes.
//
// Four in five: a rule the team keeps most of the time is still the rule,
// and demanding every commit would find nothing in any repository with
// people in it.
func (c Custom) Holds() bool { return c.Ratio() >= holdsUp }

// The shape of the reading.
const (
	// enoughCommits is how much history a reading needs before it says
	// anything. A repository somebody started last week has no history to
	// read, and that is an answer rather than an empty list.
	enoughCommits = 20
	// holdsUp is how often something has to hold before it is worth calling
	// a custom. Four in five: a rule the team keeps most of the time is
	// still the rule, and demanding every commit would find nothing in any
	// repository with people in it.
	holdsUp = 0.8
	// mostCustoms is how many come back. This is material for one or two
	// offers, not a report.
	mostCustoms = 4
)

// Customs is every reading the history could make, whether or not it held.
//
// Both, because they are two different uses. What held is worth offering as a
// rule; what was measured and did not hold is what says a rule somebody wrote
// down is no longer true — and that is the half a document cannot tell you.
// Holds() is the line between them, and it is the caller's to draw.
//
// Empty is an answer: a repository with less history than a reading needs has
// nothing to say yet, and saying so is different from having looked and found
// nothing.
func (r Repo) Customs() ([]Custom, int, error) {
	out, err := git(r.Path, "log", "--name-only",
		fmt.Sprintf("-n%d", commitsRead), "--pretty=format:%x00%H")
	if err != nil {
		return nil, 0, fmt.Errorf("read the history of %s: %w", r.Name, err)
	}

	commits := commitsOf(out)
	if len(commits) < enoughCommits {
		return nil, len(commits), nil
	}

	found := testsTravel(commits)

	shape, err := messageShape(r.Path)
	if err != nil {
		return nil, len(commits), err
	}

	found = append(found, shape...)

	sort.SliceStable(found, func(i, j int) bool { return found[i].Ratio() > found[j].Ratio() })

	if len(found) > mostCustoms {
		found = found[:mostCustoms]
	}

	return found, len(commits), nil
}

// testsTravel is, per folder, how often a commit that changed something in it
// also changed a test.
//
// By the top folder and not by the file, because that is the grain a rule is
// written at: "changes under internal/db come with a test" is a rule, and the
// same thing said about one file is a note about that file.
func testsTravel(commits [][]string) []Custom {
	touched, withTest := map[string]int{}, map[string]int{}

	for _, files := range commits {
		if len(files) > crowdedCommit {
			continue
		}

		tested := false

		for _, f := range files {
			if aTest(f) {
				tested = true
			}
		}

		for where := range topFolders(files) {
			touched[where]++

			if tested {
				withTest[where]++
			}
		}
	}

	var out []Custom

	for where, n := range touched {
		if n < enoughCommits {
			continue
		}

		out = append(out, Custom{
			Kind: TestsTravel, Where: where, Times: withTest[where], Of: n,
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Where < out[j].Where })

	return out
}

// topFolders is the top-level folders one commit touched, leaving out the
// tests themselves: a commit that only changed tests says nothing about
// whether tests travel with code.
func topFolders(files []string) map[string]bool {
	where := map[string]bool{}

	for _, f := range files {
		if aTest(f) {
			continue
		}

		if top, _, nested := strings.Cut(path.Clean(f), "/"); nested {
			where[top] = true
		}
	}

	return where
}

// aTest says whether a path is a test, in the spellings the languages Orbit
// is likely to meet use.
//
// A list and not a language: a repository whose tests are named in a way
// nobody put here simply reads as having none, which is the way round that
// fails safe — it offers no rule rather than a wrong one.
func aTest(f string) bool {
	low := strings.ToLower(f)

	for _, mark := range []string{
		"_test.", ".test.", "_spec.", ".spec.", "/test/", "/tests/", "/spec/", "test_",
	} {
		if strings.Contains(low, mark) {
			return true
		}
	}

	return false
}

// shapes are the forms a project's commit subjects keep, in the spellings
// that are common enough to be worth looking for.
//
// Hand-written and meant to be argued with. A project whose messages keep a
// shape nobody put here reads as keeping none, which offers no rule rather
// than a wrong one.
var shapes = []struct {
	name string
	is   func(subject string) bool
}{
	{"a type and a colon, like `fix: …`", conventional},
	{"the ticket at the front, like `ABC-12 …`", ticketed},
}

// messageShape is how often this project's commit subjects keep one form.
func messageShape(dir string) ([]Custom, error) {
	out, err := git(dir, "log", fmt.Sprintf("-n%d", commitsRead), "--pretty=format:%s")
	if err != nil {
		return nil, fmt.Errorf("read the commit messages of %q: %w", dir, err)
	}

	subjects := splitLines(out)
	if len(subjects) < enoughCommits {
		return nil, nil
	}

	var found []Custom

	for _, shape := range shapes {
		kept := 0

		for _, one := range subjects {
			// A merge commit is git's sentence and not the team's, so it
			// says nothing about the form they keep.
			if strings.HasPrefix(one, "Merge ") {
				continue
			}

			if shape.is(one) {
				kept++
			}
		}

		found = append(found, Custom{
			Kind: MessagesKeepAShape, Where: shape.name, Times: kept, Of: len(subjects),
		})
	}

	return found, nil
}

// conventional is `fix: something`, `feat(ui): something` — a word, maybe a
// scope in brackets, then a colon.
func conventional(subject string) bool {
	head, _, found := strings.Cut(subject, ":")
	if !found || head == "" || len(head) > 24 {
		return false
	}

	head, _, _ = strings.Cut(head, "(")

	for _, r := range head {
		if r < 'a' || r > 'z' {
			return false
		}
	}

	return true
}

// ticketed is `ABC-12 something` — letters, a dash, digits, at the front.
func ticketed(subject string) bool {
	head, _, found := strings.Cut(subject, " ")
	if !found {
		return false
	}

	key, number, found := strings.Cut(strings.TrimSuffix(head, ":"), "-")
	if !found || key == "" || number == "" {
		return false
	}

	for _, r := range key {
		if r < 'A' || r > 'Z' {
			return false
		}
	}

	for _, r := range number {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}
