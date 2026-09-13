package learn

// Saying yes: what a sentence in the tray becomes once somebody agrees with
// it, and where it goes.

import (
	"fmt"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/db"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/store"
)

// A Place is where a kept rule belongs: the checkout it is about, and the
// folder or file inside it that somebody named.
//
// The checkout is here because the commonest way a rule gets said is the
// supervisor, and the supervisor is about the board rather than about one
// task — so the sentence arrives knowing no repository at all. What it is
// about is where the reader was standing when they agreed with it.
type Place struct {
	Repo string
	Path string
}

// Keep writes the sentence down as a fact and takes it out of the tray.
//
// The text is a parameter rather than the proposal's own, because correcting
// is how most of these are accepted: what you meant is what you typed the
// second time, and a fact in words you just improved is the whole point of
// being asked instead of told.
//
// From then on it reaches every phase's prompt the way every fact does, and
// it refuses work at the gate when you give it a command that can answer yes
// or no without an opinion in it. The command is asked for here and nowhere
// else: a gate runs on every future phase in that repository, and an engine
// that could ask for one would be deciding an hour of somebody's task.
//
// The fact is written before the tray is cleared. The other order loses the
// sentence when the write fails — an offer nobody can accept twice and a
// fact that was never saved — and this way the worst case is being offered
// something you already kept.
func Keep(s *store.Store, at time.Time, text, check string, where Place) error {
	d, err := s.Record()
	if err != nil {
		return err
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("a rule with no sentence says nothing")
	}

	said, err := waitingAt(s, at)
	if err != nil {
		return err
	}

	fact, err := factOf(said, text, check, where)
	if err != nil {
		return err
	}

	if _, err := knowledge.NewStore(s.Root()).Save(fact); err != nil {
		return err
	}

	return d.Decide(at, db.Kept)
}

// waitingAt is the sentence in the tray said at that moment.
//
// It is read before anything is written, because what the fact says about
// itself is on that row: who said it, and what it was about. A number off a
// listing read a minute ago can name a row something has since decided, and
// that is a refusal rather than a fact filed under a guess.
func waitingAt(s *store.Store, at time.Time) (Said, error) {
	waiting, err := Waiting(s)
	if err != nil {
		return Said{}, err
	}

	for _, one := range waiting {
		if one.At.Equal(at) {
			return one, nil
		}
	}

	return Said{}, fmt.Errorf("nothing said at %s is waiting to be kept",
		at.Format(time.RFC3339))
}

// factOf is the fact a kept sentence becomes.
//
// The source is whoever found it and not whoever agreed with it. Agreeing
// with something an engine worked out mid-task does not make it something
// you said — the screen that lists facts says where each one came from, and
// the whole of why one can be trusted is that it can be traced back to where
// it actually came from.
//
// The scope is where the reader put it: a folder inside the checkout, one
// file, the whole checkout, or everything when the sentence was said about
// no checkout at all. A rule almost always belongs somewhere narrower than
// where it was said — you say it while correcting one run and it is true of
// one folder — and this is the moment somebody knows which.
func factOf(said Said, text, check string, where Place) (knowledge.Fact, error) {
	f := knowledge.Fact{
		Scope:  knowledge.Scope{Kind: knowledge.General},
		Source: foundBy(said),
		Phrase: text,
		Stops:  check != "",
		Check:  strings.TrimSpace(check),
		Ref:    said.About,
		At:     time.Now().UTC(),
	}

	// The task's checkout first: a rule said while correcting one run is
	// about that run's code, wherever the reader happens to be standing
	// when they get round to agreeing with it. Then the checkout they are
	// standing in, which is the only thing a sentence said to the
	// supervisor has to go on.
	repo := said.Repo
	if repo == "" {
		repo = where.Repo
	}

	// What the reader typed, then where the work was when the sentence was
	// said. A rule said in the middle of one folder is usually about that
	// folder, and having to retype a path Orbit already watched being worked
	// in is the kind of small tax that ends with nobody placing rules at
	// all. Typing `.` is how somebody says the whole checkout instead.
	path := strings.TrimSpace(where.Path)
	if path == "" {
		path = said.Path
	}

	if repo == "" {
		if path != "" {
			return knowledge.Fact{}, fmt.Errorf(
				"%q is about no checkout, so there is nothing for %q to be inside", said.Text, path)
		}

		return f, nil
	}

	// Nothing named is the whole checkout for a rule that came out of one,
	// and everywhere for one said to the supervisor from nowhere in
	// particular: agreeing with a sentence is not the same as saying it is
	// about wherever the terminal happened to be.
	if path == "" && said.Repo == "" {
		return f, nil
	}

	scope, err := knowledge.At(repo, path)
	if err != nil {
		return knowledge.Fact{}, err
	}

	f.Scope = scope

	return f, nil
}

// foundBy is where a kept sentence came from, in the words a fact records it
// in: the record when an engine worked it out on its own, and a person for
// everything else — the cockpit, a command, a channel, a correction typed at
// a run.
func foundBy(said Said) knowledge.Source {
	if said.By == AModel {
		return knowledge.FromRecord
	}

	return knowledge.Human
}
