package learn

// What the project already says about itself.
//
// Orbit starts knowing nothing, and a repository with two years behind it
// already has half of what Orbit is going to learn written down and sitting
// there: the CONTRIBUTING, the README, and the notes each engine keeps in a
// file of its own.
//
// Those last ones are the point. You explain everything to one engine, then
// try another and have to explain it all again — which is exactly what this
// package's own reason for existing says it is for. The silos are already in
// the repository; reading them is taking the knowledge out of one engine and
// putting it where every engine sees it.
//
// The risk is one, and it is the whole of the design: flooding the tray. A
// CONTRIBUTING from two years ago says things nobody does any more, and forty
// weak offers is a tray somebody stops opening — which would take the other
// three sources down with it, because they share that tray. So this brings
// few and good rather than everything it can find.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/e1i0r/orbit/internal/store"
)

// papers are the files a project keeps what it expects in, in the order they
// are worth reading.
//
// Hand-written and meant to be argued with, like every other list here. The
// engine silos come first because they are where somebody has already sat
// down and explained this project to a model — which is the closest thing to
// a rule anybody has written — and the README comes last because most of it
// describes rather than expects.
var papers = []string{
	"CLAUDE.md", "AGENTS.md", "GEMINI.md", ".cursorrules",
	"CONTRIBUTING.md", "CONTRIBUTING", "docs/CONTRIBUTING.md",
	"README.md", "README",
}

// atMostPapers is how many of them are read in one go, and atMostCold how
// many rules the whole reading may offer.
//
// Five rules, by hand. It is better to bring five that are still kept than
// forty somebody wanted once: the second is a tray nobody opens, and a tray
// nobody opens is every source turned off at once.
const (
	atMostPapers = 4
	atMostCold   = 5
)

// aPaper is how much of one file is shown to the model.
//
// Sixty thousand characters is a long README and a very long CONTRIBUTING. A
// file past it is read from the top, because what a project expects is said
// near the front and what is at the end is usually history.
const aPaper = 60000

// Read offers rules out of what the project already says about itself.
//
// It runs when somebody asks and not on every run: this is money spent on
// files that have not changed, and the answer would be the same every time.
// A file already read is not read again until it changes.
func Read(ctx context.Context, s *store.Store, ask Ask, repo string) ([]Said, error) {
	if ask == nil {
		return nil, fmt.Errorf("there is no engine here to read what the project says")
	}

	if repo == "" {
		return nil, fmt.Errorf("say which checkout to read")
	}

	d, err := s.Record()
	if err != nil {
		return nil, err
	}

	var (
		out  []Said
		read int
	)

	for _, paper := range papers {
		if read >= atMostPapers || len(out) >= atMostCold {
			break
		}

		body, mark, ok := paperAt(repo, paper)
		if !ok {
			continue
		}

		answered, err := d.Answered(mark)
		if err != nil {
			return out, err
		}

		if answered {
			continue
		}

		read++

		said, err := outOf(ctx, s, ask, repo, paper, body, mark, atMostCold-len(out))
		if err != nil {
			return out, err
		}

		out = append(out, said...)
	}

	return out, nil
}

// paperAt reads one of them, and answers with what it says and the name it is
// remembered by.
//
// The name carries the file and what was in it, so that a file somebody has
// since rewritten is read again and one they have not is left alone. Reading
// the same CONTRIBUTING twice would offer the same rules twice, and the
// second time they were already answered.
func paperAt(repo, paper string) (body, mark string, ok bool) {
	at := filepath.Join(repo, filepath.FromSlash(paper))

	read, err := os.ReadFile(at)
	if err != nil {
		return "", "", false
	}

	text := string(read)
	if len(text) > aPaper {
		text = text[:aPaper]
	}

	sum := sha256.Sum256(read)

	return text, paper + "@" + hex.EncodeToString(sum[:8]), true
}

// outOf is the rules one file turned into, put in the tray.
func outOf(
	ctx context.Context, s *store.Store, ask Ask,
	repo, paper, body, mark string, room int,
) ([]Said, error) {
	answer, err := ask(ctx, aboutThisProject(paper, body, room))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", paper, err)
	}

	at := time.Now().UTC()

	var out []Said

	for _, one := range quoted(answer, body, room) {
		said := Said{
			At: at, Text: one.phrase, By: FromAPaper,
			Repo: repo, Path: alongside(paper), Topic: one.topic,
			Habit: mark, About: paper + ":" + one.line,
		}

		if err := Propose(s, said); err != nil {
			return out, err
		}

		out = append(out, said)
		at = at.Add(time.Nanosecond)
	}

	return out, nil
}

// alongside is where a rule found in a file applies: the folder the file is
// in, and the whole checkout for one at the root.
//
// A rule in backend/README is about backend. It is the same reading the rest
// of this package makes of a place, and it is right far more often than
// filing everything against the whole project would be.
func alongside(paper string) string {
	if dir := filepath.ToSlash(filepath.Dir(paper)); dir != "." {
		return dir
	}

	return ""
}
