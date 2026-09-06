package record

// The supervisor thread, cut into conversations.
//
// It was one list with no ends: every line anybody had ever said to the
// supervisor, in one column, growing. What it is now is a list of
// conversations, each one titled by the sentence that started it — because
// that sentence is what somebody remembers about it, and because a thread
// with no ends is one nobody scrolls back through twice.
//
// Cutting them apart is safe for one reason: nothing permanent lives in
// them. The moment something was worth keeping, /rule or /aware sent it to
// what Orbit knows. A conversation is deliberately disposable.

import (
	"sort"
	"strings"
	"time"
)

// titleCut is how much of the first sentence a title carries. Enough to
// recognise the conversation by, and short enough that a list of them is a
// list rather than a wall.
const titleCut = 60

// Conversation is one of them, as a list shows it.
type Conversation struct {
	ID    string    // what its turns carry; empty is the thread that came before there were ids
	Title string    // the first thing said in it
	First time.Time // when it started
	Last  time.Time // when it was last spoken in
	Turns int       // how many turns still stand in it
}

// ConversationOf is the conversation a turn belongs to.
func ConversationOf(e Event) string { return e.Data["conversation"] }

// NewConversationID is the id a conversation started right now carries.
//
// It is the moment it began, stamped the way every other time in this record
// is: unique enough for a person's thread, sorted by the same order the
// turns are, and traceable back to the line that opened it.
func NewConversationID(at time.Time) string { return Stamp(at) }

// Conversations is the thread grouped, most recently spoken in first.
//
// Turns that were taken back do not count and do not title anything: a
// retracted line is one the reader decided was not said to the supervisor,
// and a list that titled a conversation with it would put it back on screen.
func Conversations(events []Event) []Conversation {
	var (
		gone  = Retracted(events)
		off   = RemovedConversations(events)
		by    = map[string]*Conversation{}
		order []string
	)

	for _, e := range events {
		if !counts(e, gone) {
			continue
		}

		id := ConversationOf(e)
		if off[id] {
			continue
		}

		held, seen := by[id]
		if !seen {
			held = &Conversation{ID: id, Title: title(e), First: e.At}
			by[id] = held
			order = append(order, id)
		}

		held.Last = e.At
		held.Turns++
	}

	out := make([]Conversation, 0, len(order))
	for _, id := range order {
		out = append(out, *by[id])
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].Last.After(out[j].Last) })

	return out
}

// InConversation is one conversation's turns, in the order they were said.
//
// The retractions come with them: this is the thread as the screen draws it,
// and folding what was taken back is the screen's own rule, applied where
// every other reader applies it.
func InConversation(id string, events []Event) []Event {
	off := RemovedConversations(events)
	if off[id] {
		return nil
	}

	var out []Event

	for _, e := range events {
		if ConversationOf(e) == id || (e.Kind == SupervisorRetracted && retractedIn(id, e, events)) {
			out = append(out, e)
		}
	}

	return out
}

// RemovedConversations is every conversation somebody has taken off the
// list.
func RemovedConversations(events []Event) map[string]bool {
	off := map[string]bool{}

	for _, e := range events {
		if e.Kind != SupervisorConversationRemoved {
			continue
		}

		if id, held := e.Data["conversation"]; held {
			off[id] = true
		}
	}

	return off
}

// counts is whether a turn is one of the conversation's own: something said,
// still standing, and not the bookkeeping that took something back.
func counts(e Event, gone map[string]bool) bool {
	switch e.Kind {
	case SupervisorRetracted, SupervisorConversationRemoved:
		return false
	}

	return !gone[Stamp(e.At)] && strings.TrimSpace(e.Text) != ""
}

// retractedIn is whether a retraction takes back a turn of this conversation.
func retractedIn(id string, retraction Event, events []Event) bool {
	at := retraction.Data["at"]
	if at == "" {
		return false
	}

	for _, e := range events {
		if Stamp(e.At) == at {
			return ConversationOf(e) == id
		}
	}

	return false
}

// title is the first line of what was said, cut to something a list can show.
func title(e Event) string {
	line := strings.TrimSpace(e.Text)
	if cut, _, held := strings.Cut(line, "\n"); held {
		line = strings.TrimSpace(cut)
	}

	runes := []rune(line)
	if len(runes) <= titleCut {
		return line
	}

	return strings.TrimSpace(string(runes[:titleCut])) + "…"
}
