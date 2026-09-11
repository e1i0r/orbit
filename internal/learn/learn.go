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
	"github.com/e1i0r/orbit/internal/store"
)

// A Said is one sentence waiting to be told whether it was a rule.
type Said struct {
	// At is when it was said, and it is the sentence's name. The thread is
	// append-only and no two turns share an instant, so the moment is what
	// stops one line being offered twice.
	At   time.Time
	Text string
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

	return d.Propose(db.Proposal{SaidAt: said.At, Said: said.Text})
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
		out = append(out, Said{At: row.SaidAt, Text: row.Said})
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

	fact := knowledge.Fact{
		Scope:  knowledge.Scope{Kind: knowledge.General},
		Source: knowledge.Human,
		Phrase: text,
		Stops:  check != "",
		Check:  strings.TrimSpace(check),
		At:     time.Now().UTC(),
	}

	if _, err := knowledge.NewStore(s.Root()).Save(fact); err != nil {
		return err
	}

	return d.Decide(at, db.Kept)
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
