package cli

// What `orbit web` reads the three non-record screens through.
//
// internal/web knows no store, no engine and no state root — the same rule
// the window is built under — so what it cannot reach arrives through a port
// filled here, where store, knowledge, supervisor and engine may all be
// named. The window's own ports are next door in top.go, learn.go and
// engines.go, and these are built on top of them so the browser and the
// terminal read one answer rather than two.

import (
	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/quota"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/supervisor"
	"github.com/e1i0r/orbit/internal/ui/fact"
	"github.com/e1i0r/orbit/internal/ui/roster"
	"github.com/e1i0r/orbit/internal/web"
	"github.com/e1i0r/orbit/internal/words"
)

// knows fills the facts port off the same reader the cockpit's screen uses.
type knows struct {
	all func() []knowledge.Fact
}

func (k knows) Facts() []web.Fact {
	if k.all == nil {
		return nil
	}

	out := make([]web.Fact, 0)

	for _, f := range k.all() {
		out = append(out, web.Fact{
			Phrase: f.Phrase,
			Scope:  fact.Where(f.Scope),
			Source: sourceName(f.Source),
			Action: actionName(f.Action()),
			Check:  f.Check,
			Ref:    f.Ref,
			Repo:   f.Scope.Repo,
			At:     f.At,
			Used:   f.Used,
			Off:    f.Off,
		})
	}

	return out
}

// sourceName is where a fact came from, in a word a reader reads.
//
// Spelled here and not in internal/knowledge for the reason flow's Origin is
// a value and not a sentence: the words a reader sees are written at the
// call site that shows them.
func sourceName(s knowledge.Source) string {
	switch s {
	case knowledge.FromCode:
		return "read off the code"
	case knowledge.Human:
		return "said by a person"
	case knowledge.FromRecord:
		return "learned from a run"
	case knowledge.FromProduction:
		return "from an incident"
	}

	return "unsourced"
}

// actionName is what a fact does when work reaches its scope — which is not
// always what it was asked to do, and Action is what settles that.
func actionName(a knowledge.Action) string {
	if a == knowledge.Stops {
		return "stops"
	}

	return "warns"
}

// talks fills the supervisor port.
type talks struct {
	store *store.Store
}

func (t talks) Thread() ([]web.Chat, []web.Said, error) {
	events, err := supervisor.Events(t.store)
	if err != nil {
		return nil, nil, err
	}

	chats := make([]web.Chat, 0, 4)
	for _, c := range supervisor.Conversations(events) {
		chats = append(chats, web.Chat{
			ID: c.ID, Title: c.Title, First: c.First, Last: c.Last, Turns: c.Turns,
		})
	}

	said := make([]web.Said, 0, len(events))
	for _, e := range events {
		said = append(said, web.Said{
			At:           e.At,
			Kind:         e.Kind,
			By:           e.Data["by"],
			Channel:      e.Data["channel"],
			Task:         e.Data["task_id"],
			Repo:         e.Data["repo"],
			Text:         e.Text,
			Conversation: record.ConversationOf(e),
		})
	}

	return chats, said, nil
}

// roll fills the engines port off the cockpit's own catalogue and meter.
type roll struct {
	engines func() []roster.Engine
	quota   func(string) roster.Reading
	settled string
	// words is the reader's own, for the one part of this listing that is
	// sentences: what to do about an engine this machine cannot run yet.
	// Those are steps somebody follows, and the rest of the page is already
	// in their language.
	words *words.Printer
}

func (r roll) Settled() string { return r.settled }

// printer is the reader's language, and English for a roll built without
// one. Nothing in the program builds one that way, but the alternative to
// answering here is the nil dereference this replaces.
func (r roll) printer() *words.Printer {
	if r.words == nil {
		return words.For("")
	}

	return r.words
}

func (r roll) Engines() []web.EngineInfo {
	if r.engines == nil {
		return nil
	}

	out := make([]web.EngineInfo, 0)

	for _, e := range r.engines() {
		one := web.EngineInfo{
			Name:      e.Name,
			Available: e.Available,
			Models:    labels(e.Models),
			Efforts:   labels(e.Efforts),
			CanThink:  e.CanThink,
		}

		// Only an engine that cannot run carries these, which is why a nil
		// printer here went unseen for as long as it did: every machine
		// that had the engines installed never reached it, and the one
		// that did not got a server that fell over instead of a page
		// saying how to install them.
		if e.Setup != nil {
			one.Setup = e.Setup(r.printer())
		}

		if r.quota != nil {
			reading := r.quota(e.Name)
			one.Money, one.Sourced = reading.Money, reading.Sourced

			for _, wd := range reading.Windows {
				one.Quota = append(one.Quota, web.Window{
					Label:    wd.Label,
					Pct:      roster.Used(wd),
					ResetsIn: int(wd.ResetsIn.Seconds()),
				})
			}
		}

		out = append(out, one)
	}

	return out
}

// labels is a dial's choices as the page shows them.
func labels(from []roster.Choice) []string {
	out := make([]string, 0, len(from))
	for _, c := range from {
		out = append(out, c.Label)
	}

	return out
}

// webPorts is everything the browser reads through, built once.
func webPorts(
	r *board.Reader, s *store.Store, engines map[string]engine.Engine, dir string, p *words.Printer,
) web.Ports {
	hand := hands{store: s, board: r, words: p}

	return web.Ports{
		Board: r,
		Trees: r,
		Flows: s,
		Knows: knows{all: knowsAllPort(r, s)},
		Talks: talks{store: s},
		// One value fills all three: what can be asked for and what asking
		// does need the same store and the same flow. They are separate
		// ports because two of them change nothing — see
		// internal/web/ports.go.
		Asks:      hand,
		Told:      hand,
		Standings: hand,
		Roster: roll{
			engines: enginesPort(engines),
			quota:   quotaPort(quota.FromEnv(), false),
			settled: settledEngine(s, engines),
			words:   p,
		},
		Root: dir,
	}
}

// settledEngine is the engine a run uses when the dials name none: the
// settings file's, or the first the catalogue lists. The same rule the
// window's own dial follows.
func settledEngine(s *store.Store, engines map[string]engine.Engine) string {
	if cfg, err := s.Settings(); err == nil && cfg.Engine != "" {
		return cfg.Engine
	}

	names := engineNames(engines)
	if len(names) == 0 {
		return ""
	}

	return names[0]
}
