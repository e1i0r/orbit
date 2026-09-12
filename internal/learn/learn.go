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

// Operator is what By says when it was typed at one of the controls and
// nothing more particular is known. It is what a sentence with no By at all
// becomes: a row somebody has to look at is the way round that fails safe.
const Operator = "operator"

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
		By: said.By, About: said.About, Repo: said.Repo,
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
			By: row.By, About: row.About, Repo: row.Repo,
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
func Keep(s *store.Store, at time.Time, text, check string) error {
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

	if _, err := knowledge.NewStore(s.Root()).Save(factOf(said, text, check)); err != nil {
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
// The scope is the checkout the sentence was said about, and everything when
// it was said about no checkout. Nothing narrower — a file or a symbol is a
// precision nobody has agreed to, and the Knowledge screen is where somebody
// who has read one moves it.
func factOf(said Said, text, check string) knowledge.Fact {
	f := knowledge.Fact{
		Scope:  knowledge.Scope{Kind: knowledge.General},
		Source: knowledge.Human,
		Phrase: text,
		Stops:  check != "",
		Check:  strings.TrimSpace(check),
		Ref:    said.About,
		At:     time.Now().UTC(),
	}

	if said.Repo != "" {
		f.Scope = knowledge.Scope{Kind: knowledge.Repo, Repo: said.Repo}
	}

	return f
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
