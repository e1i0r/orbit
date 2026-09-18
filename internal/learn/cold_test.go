package learn

// Reading what a project already says about itself.

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// aProject is a checkout with the files a project keeps its expectations in.
func aProject(t *testing.T, files map[string]string) string {
	t.Helper()

	repo := t.TempDir()

	for name, body := range files {
		at := filepath.Join(repo, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
			t.Fatalf("make room for %s: %v", name, err)
		}

		if err := os.WriteFile(at, []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	return repo
}

// heldUp is one line of an answer whose quote really is in the file below.
const heldUp = "testing | a change comes with a test | " +
	"Every change ships with a test that fails without it."

const aContributing = `# Contributing

This project is written in Go.

Every change ships with a test that fails without it.
Never push on a Friday afternoon.
`

// TestARuleHasToPointAtALineOfTheFile.
//
// The whole reason this can be trusted. A model asked to summarise two years
// of CONTRIBUTING will produce plausible rules nobody ever wrote, and the
// only way to tell those from the real ones without reading it yourself is to
// make it point at the line.
func TestARuleHasToPointAtALineOfTheFile(t *testing.T) {
	s := root(t)
	repo := aProject(t, map[string]string{"CONTRIBUTING.md": aContributing})

	answer := heldUp + "\n" + "process | deploys go out on Tuesdays | Deploys go out on Tuesdays."

	got, err := Read(context.Background(), s, answering(answer, nil), repo, nil)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("the answer became %d rules: %+v", len(got), got)
	}

	if !strings.Contains(got[0].Text, "comes with a test") {
		t.Errorf("the one that held up reads %q", got[0].Text)
	}

	// And it says where to go and look, which is half of why a rule from a
	// document can be trusted at all.
	if got[0].About != "CONTRIBUTING.md:5" {
		t.Errorf("it says it came from %q", got[0].About)
	}

	if got[0].By != FromAPaper {
		t.Errorf("it says it came from %q", got[0].By)
	}
}

// TestAFileAlreadyReadIsNotReadAgain, until it changes. Reading the same
// CONTRIBUTING twice offers the same rules twice, and the second time they
// were already answered.
func TestAFileAlreadyReadIsNotReadAgain(t *testing.T) {
	s := root(t)
	repo := aProject(t, map[string]string{"CONTRIBUTING.md": aContributing})

	if _, err := Read(context.Background(), s, answering(heldUp, nil), repo, nil); err != nil {
		t.Fatalf("reading: %v", err)
	}

	again, err := Read(context.Background(), s, answering(heldUp, nil), repo, nil)
	if err != nil {
		t.Fatalf("reading again: %v", err)
	}

	if len(again) != 0 {
		t.Errorf("a file nobody has touched was read again: %+v", again)
	}

	// Rewritten, it is read again: what changed may be a rule.
	rewritten := aContributing + "\nAlways wrap an error with what you were doing.\n"
	at := filepath.Join(repo, "CONTRIBUTING.md")

	if err := os.WriteFile(at, []byte(rewritten), 0o600); err != nil {
		t.Fatal(err)
	}

	after, err := Read(context.Background(), s, answering(heldUp, nil), repo, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(after) == 0 {
		t.Error("a file somebody rewrote was not read again")
	}
}

// TestARuleInAFolderIsAboutThatFolder, because a rule in backend/README is
// about backend.
func TestARuleInAFolderIsAboutThatFolder(t *testing.T) {
	if got := alongside("backend/README.md"); got != "backend" {
		t.Errorf("a rule in backend/README.md is about %q", got)
	}

	if got := alongside("CONTRIBUTING.md"); got != "" {
		t.Errorf("a rule at the root is about %q", got)
	}
}

// TestAQuoteTheFileWrappedIsStillTheQuote.
//
// A model copying a sentence out of a wrapped paragraph rejoins it with one
// space where the file had a newline, and refusing that would throw away real
// quotes for a difference nobody can see.
func TestAQuoteTheFileWrappedIsStillTheQuote(t *testing.T) {
	body := "# Contributing\n\nEvery change ships with a test\nthat fails without it.\n"

	at, there := lineOf(body, "Every change ships with a test that fails without it.")
	if !there || at != 3 {
		t.Errorf("the wrapped sentence reads as line %d, there=%v", at, there)
	}

	if _, there := lineOf(body, "deploys go out on Tuesdays"); there {
		t.Error("a sentence that is not in the file was found in it")
	}
}

// TestReadingNeedsAnEngineAndACheckout, and says which is missing rather than
// answering with nothing.
func TestReadingNeedsAnEngineAndACheckout(t *testing.T) {
	s := root(t)

	if _, err := Read(context.Background(), s, nil, t.TempDir(), nil); err == nil {
		t.Error("reading with no engine answered as though it had one")
	}

	if _, err := Read(context.Background(), s, answering("", nil), "", nil); err == nil {
		t.Error("reading no checkout answered as though there were one")
	}
}

// aRuled is a file with more standing sentences in it than one reading is
// allowed to offer.
const aRuled = `# Contributing

Every change ships with a test that fails without it.
Never push on a Friday afternoon.
Always wrap an error with what you were doing.
Deploys go out on Tuesdays.
Amounts are written in cents.
Comments explain rather than judge.
`

// theSentencesOf is what aRuled says, in the order it says it.
var theSentencesOf = []string{
	"Every change ships with a test that fails without it.",
	"Never push on a Friday afternoon.",
	"Always wrap an error with what you were doing.",
	"Deploys go out on Tuesdays.",
	"Amounts are written in cents.",
	"Comments explain rather than judge.",
}

// anAnswerOf is a model's reply holding n rules, each pointing at its own
// line of aRuled.
func anAnswerOf(n int) string {
	lines := make([]string, 0, n)
	for i := range n {
		lines = append(lines, "testing | what line "+strconv.Itoa(i)+" asks for | "+theSentencesOf[i])
	}

	return strings.Join(lines, "\n")
}

// asAsked answers each file with whatever it was given for it, and keeps
// every question, so that which files were read is something a test can ask
// about rather than infer from what came back.
func asAsked(answers map[string]string, asked *[]string) Ask {
	return func(_ context.Context, question string) (string, error) {
		*asked = append(*asked, question)

		for paper, answer := range answers {
			if strings.Contains(question, "Here is "+paper+" from") {
				return answer, nil
			}
		}

		return "", nil
	}
}

// TestOnlySoManyOfAProjectsFilesAreReadInOneGo.
//
// Every file read is a model call paid for, and a project that keeps a note
// for each of four engines plus a CONTRIBUTING and a README would spend six
// of them on one press of a key. The cap is what makes this safe to offer as
// a button rather than as something somebody budgets for.
func TestOnlySoManyOfAProjectsFilesAreReadInOneGo(t *testing.T) {
	s := root(t)
	repo := aProject(t, map[string]string{
		"CLAUDE.md":       aContributing,
		"AGENTS.md":       aContributing,
		"GEMINI.md":       aContributing,
		".cursorrules":    aContributing,
		"CONTRIBUTING.md": aContributing,
	})

	var asked []string

	answers := map[string]string{}
	for _, paper := range papers {
		answers[paper] = "testing | " + paper + " asks for a test | " +
			"Every change ships with a test that fails without it."
	}

	got, err := Read(context.Background(), s, asAsked(answers, &asked), repo, nil)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}

	if len(asked) != atMostPapers {
		t.Fatalf("%d of the five files were read, want %d", len(asked), atMostPapers)
	}

	if len(got) != atMostPapers {
		t.Fatalf("the reading offered %d rules: %+v", len(got), got)
	}

	// And they are the four the list puts first, because the engine notes
	// are where somebody has already sat down and explained this project.
	if strings.Contains(strings.Join(asked, "\n"), "Here is CONTRIBUTING.md from") {
		t.Error("the fifth file was read, and the four before it are the ones worth paying for")
	}
}

// TestAWholeReadingOffersOnlySoManyRules.
//
// The cap is on the reading and not on the file, so a second file is asked
// for what is left rather than for the whole of it again — and once there is
// nothing left, no further file is read at all. Forty weak offers is a tray
// somebody stops opening, and they share it with three other sources.
func TestAWholeReadingOffersOnlySoManyRules(t *testing.T) {
	s := root(t)
	repo := aProject(t, map[string]string{
		"CLAUDE.md": aRuled,
		"AGENTS.md": aRuled,
		"GEMINI.md": aRuled,
	})

	var asked []string

	got, err := Read(context.Background(), s, asAsked(map[string]string{
		"CLAUDE.md": anAnswerOf(3),
		"AGENTS.md": anAnswerOf(6),
		"GEMINI.md": anAnswerOf(6),
	}, &asked), repo, nil)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}

	if len(got) != atMostCold {
		t.Fatalf("a reading of three files offered %d rules, want at most %d", len(got), atMostCold)
	}

	if len(asked) != 2 {
		t.Fatalf("%d files were read, want the two it took to fill the tray", len(asked))
	}

	// The second is asked for the room that is left: three came out of the
	// first, so there is room for two. Asking for five again would mean the
	// model writes three nobody can be offered.
	if !strings.Contains(asked[1], "Write at most 2 lines") {
		t.Error("the second file was asked for something other than the room left for it")
	}
}
