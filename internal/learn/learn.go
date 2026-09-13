// Package learn turns what gets said around Orbit into something it knows.
//
// You tell the supervisor "never push a pull request without the tests
// passing". That sentence is already the answer: it is about your code, in
// your words, and you meant it. All that is missing is for Orbit to notice
// it was a rule and ask whether to keep it.
//
// An engine that hits a wall mid-task offers what it found through the same
// door, and waits in the same queue. One flow, and not two: a rule that
// holds for a person and not for the model is not a rule, and a second way
// in that skipped the asking would be exactly the way something nobody read
// ends up in every prompt.
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
	"time"

	"github.com/e1i0r/orbit/internal/db"
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

// From is where it came from: the task it was typed at, or the way in it
// came through when it was about no task — and whether anybody said it at
// all.
//
// It is what a reader needs before they can agree with anything. "Never
// merge without the tests passing" said while correcting one run and the
// same words said to the supervisor are the same rule, and which it was is
// how somebody decides whether it was meant that widely. A sentence a model
// worked out on its own is read differently again, and more carefully, so
// the two are never printed the same.
func (s Said) From() string {
	if s.About == "" {
		return s.By
	}

	if s.By == AModel {
		return s.About + " · " + AModel
	}

	return s.About
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

// Drop says it was not a rule. The sentence stays in the thread where you
// said it, which is where it belonged all along.
func Drop(s *store.Store, at time.Time) error {
	d, err := s.Record()
	if err != nil {
		return err
	}

	return d.Decide(at, db.Dropped)
}
