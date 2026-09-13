// Package learn turns what you say into something Orbit knows.
//
// You tell the supervisor "never push a pull request without the tests
// passing". That sentence is already the answer: it is about your code, in
// your words, and you meant it. All that is missing is for Orbit to notice
// it was a rule and ask whether to keep it.
//
// So a sentence waits here between being said and being agreed with. It is
// not knowledge yet and must not be kept as any: a fact is a file, and every
// fact there is goes into every phase's prompt — so something nobody has
// agreed to has to wait somewhere the model cannot see it.
//
// Nothing here decides anything. What it holds is an offer, and an offer
// nobody accepts costs a row.
package learn

import (
	"fmt"
	"strings"
	"time"

	"github.com/e1i0r/orbit/internal/db"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/logger"
	"github.com/e1i0r/orbit/internal/store"
)

// A Said is one sentence waiting to be told whether it was a rule.
type Said struct {
	// At is when it was said, and it is the sentence's name. The thread is
	// append-only and no two turns share an instant, so the moment is what
	// stops one line being offered twice.
	At   time.Time
	Text string
	// By is where it was said: the cockpit, a command, a tool call, the
	// name of whoever typed a correction at a run. The tray shows it, so
	// that a sentence can be placed before it is agreed with.
	By string
	// About is the task it came out of, and empty for the supervisor's
	// thread, which is about the board rather than about one task.
	About string
	// Repo is the checkout it is about, and empty for a sentence that is
	// about everything.
	Repo string
	// Path is the place the sentence arrived with, relative to that
	// checkout, and empty when it arrived with none.
	//
	// It is not a guess about what the sentence meant. Said at a run it is
	// the folder the work was in, which is where somebody was standing;
	// found by a model it is the file or folder that model named. Either
	// way it is known rather than inferred, and it is what the rule turns
	// out to be about often enough that typing it again is work nobody
	// should have to do. Keeping the sentence somewhere else overrides it.
	Path string
}

// From is where it was said: the task it was typed at, or the way in it came
// through when it was about no task.
//
// It is what a reader needs before they can agree with anything. "Never
// merge without the tests passing" said while correcting one run and the
// same words said to the supervisor are the same rule, and which it was is
// how somebody decides whether it was meant that widely.
func (s Said) From() string {
	if s.About != "" {
		return s.About
	}

	return s.By
}

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

// Operator is what By says when it was typed at one of the controls and
// nothing more particular is known. It is what a sentence with no By at all
// becomes: a row somebody has to look at is the way round that fails safe.
const Operator = "operator"

// AModel is what By says when an engine worked the sentence out on its own,
// mid-task, rather than being told it.
//
// Not the engine's name. The tool is served to whoever connected to it, and
// the task's configured engine is a guess about who is calling — while what
// the reader actually needs is the one thing that is certain: nobody said
// this, something deduced it. That is what changes how carefully it is read,
// and what the fact records as its source once it is agreed with.
const AModel = "model"

// itsOwn is what By says when Orbit said it to itself.
//
// The kind cannot tell the supervisor's answers apart from yours: they are
// recorded as the same sort of turn. And when its loop directs a run, what
// it is passing on is something you already said to it — which the thread
// caught the first time. Either way this is the one value that is never a
// person, and never a finding of its own.
const itsOwn = "supervisor"

// Heard is told everything said to Orbit — every line of the supervisor's
// thread, every directive typed at a run — and puts the ones that were rules
// in the tray.
//
// It answers nothing and cannot be made to fail. Saying something is the
// thing that matters at that moment; noticing it was also a rule is a
// question asked about it, and a question that cannot be asked is not a
// reason to lose the sentence — or, at a run, to lose the correction.
//
// What Orbit said to itself is skipped. Everything else is somebody: the
// cockpit, a command, a tool call — and a name nobody has invented yet
// counts as a person rather than being quietly ignored, which is the way
// round that fails safe.
func Heard(s *store.Store, said Said) {
	if s == nil || said.By == itsOwn || !aboutTheFuture(said.Text) {
		return
	}

	if err := Propose(s, said); err != nil {
		logger.Warn("learn", "what was said at %s was not offered back: %v",
			said.At.Format(time.RFC3339), err)
	}
}

// Propose puts a sentence in the tray, and says nothing about one already
// there.
//
// Silence rather than an error, because the caller reads a whole thread and
// most of what it finds it has found before. Proposing is not an event worth
// reporting; deciding is.
func Propose(s *store.Store, said Said) error {
	d, err := s.Record()
	if err != nil {
		return err
	}

	if said.By == "" {
		said.By = Operator
	}

	return d.Propose(db.Proposal{
		SaidAt: said.At, Said: said.Text,
		By: said.By, About: said.About, Repo: said.Repo, Path: said.Path,
	})
}

// Waiting is every sentence nobody has decided about, oldest first.
func Waiting(s *store.Store) ([]Said, error) {
	d, err := s.Record()
	if err != nil {
		return nil, err
	}

	rows, err := d.Waiting()
	if err != nil {
		return nil, err
	}

	out := make([]Said, 0, len(rows))
	for _, row := range rows {
		out = append(out, Said{
			At: row.SaidAt, Text: row.Said,
			By: row.By, About: row.About, Repo: row.Repo, Path: row.Path,
		})
	}

	return out, nil
}

// Keep writes the sentence down as a fact of yours and takes it out of the
// tray.
//
// The text is a parameter rather than the proposal's own, because correcting
// is how most of these are accepted: what you meant is what you typed the
// second time, and a fact in words you just improved is the whole point of
// being asked instead of told.
//
// It is yours, so it reaches every phase's prompt the way every fact does,
// and it refuses work at the gate when you give it a command that can answer
// yes or no without an opinion in it.
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
// Always a person's. Everything that reaches this tray is somebody talking —
// the cockpit, a command, a tool call, a correction typed at a run — and the
// one thing that is not is skipped before it gets here. What a model works
// out on its own goes somewhere else entirely: it is written where it was
// found, about the repository it was found in.
//
// The scope is where the reader put it: a folder inside the checkout, one
// file, the whole checkout, or everything when the sentence was said about
// no checkout at all. A rule almost always belongs somewhere narrower than
// where it was said — you say it while correcting one run and it is true of
// one folder — and this is the moment somebody knows which.
func factOf(said Said, text, check string, where Place) (knowledge.Fact, error) {
	f := knowledge.Fact{
		Scope:  knowledge.Scope{Kind: knowledge.General},
		Source: knowledge.Human,
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

// Drop says it was not a rule. The sentence stays in the thread where you
// said it, which is where it belonged all along.
func Drop(s *store.Store, at time.Time) error {
	d, err := s.Record()
	if err != nil {
		return err
	}

	return d.Decide(at, db.Dropped)
}
