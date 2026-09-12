package record

// The thread cut into conversations.

import (
	"testing"
	"time"
)

func said(at time.Time, id, text string) Event {
	return Event{
		At:   at,
		Kind: SupervisorMessage,
		Text: text,
		Data: map[string]string{"conversation": id},
	}
}

// TestAConversationIsTitledByWhatStartedIt, which is what somebody remembers
// it by.
func TestAConversationIsTitledByWhatStartedIt(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	events := []Event{
		said(now, "c1", "Revisar el módulo de pagos"),
		said(now.Add(time.Minute), "c1", "y de paso los tests"),
		said(now.Add(time.Hour), "c2", "Migración a SQLite"),
	}

	got := Conversations(events)
	if len(got) != 2 {
		t.Fatalf("three turns became %d conversations", len(got))
	}

	// Most recently spoken in first.
	if got[0].ID != "c2" || got[0].Title != "Migración a SQLite" {
		t.Errorf("the newest is %+v", got[0])
	}

	if got[1].Turns != 2 || !got[1].First.Equal(now) {
		t.Errorf("the older one is %+v", got[1])
	}
}

// TestTheThreadThatCameBeforeIdsIsAConversation. Every line said before
// conversations existed carries no id, and it is one conversation rather
// than none: dropping it would hide the whole thread the day this shipped.
func TestTheThreadThatCameBeforeIdsIsAConversation(t *testing.T) {
	now := time.Now().UTC()
	events := []Event{
		{At: now, Kind: SupervisorMessage, Text: "lo de siempre"},
	}

	got := Conversations(events)
	if len(got) != 1 || got[0].ID != "" || got[0].Title != "lo de siempre" {
		t.Errorf("the thread with no ids came back as %+v", got)
	}
}

// TestARetractedLineTitlesNothing. It is a line the reader decided was not
// said, and a list that titled a conversation with it would put it back on
// the screen it was taken off.
func TestARetractedLineTitlesNothing(t *testing.T) {
	now := time.Date(2026, 9, 5, 9, 0, 0, 0, time.UTC)
	events := []Event{
		said(now, "c1", "pegué esto sin querer"),
		said(now.Add(time.Minute), "c1", "lo que quería decir"),
		{At: now.Add(2 * time.Minute), Kind: SupervisorRetracted, Data: map[string]string{"at": Stamp(now)}},
	}

	got := Conversations(events)
	if len(got) != 1 {
		t.Fatalf("got %d conversations", len(got))
	}

	if got[0].Title != "lo que quería decir" || got[0].Turns != 1 {
		t.Errorf("the retracted line still counts: %+v", got[0])
	}
}

// TestRemovingAConversationTakesItOutOfTheListAndOutOfTheContext, and out of
// nothing else: the turns stay where they are.
func TestRemovingAConversationTakesItOutOfTheListAndOutOfTheContext(t *testing.T) {
	now := time.Now().UTC()
	events := []Event{
		said(now, "c1", "una"),
		said(now.Add(time.Minute), "c2", "otra"),
		{
			At:   now.Add(2 * time.Minute),
			Kind: SupervisorConversationRemoved,
			Data: map[string]string{"conversation": "c1"},
		},
	}

	got := Conversations(events)
	if len(got) != 1 || got[0].ID != "c2" {
		t.Errorf("the removed conversation is still listed: %+v", got)
	}

	if turns := InConversation("c1", events); turns != nil {
		t.Errorf("a removed conversation still answers with %d turns", len(turns))
	}

	// And the record still holds every line of it.
	if len(events) != 3 {
		t.Error("removing a conversation changed the record")
	}
}

// TestInIsOneConversationsOwnTurns, with the retractions that belong to it
// so the screen can fold them where it folds every other.
func TestInIsOneConversationsOwnTurns(t *testing.T) {
	now := time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC)
	events := []Event{
		said(now, "c1", "una"),
		said(now.Add(time.Minute), "c2", "otra"),
		{At: now.Add(2 * time.Minute), Kind: SupervisorRetracted, Data: map[string]string{"at": Stamp(now)}},
	}

	got := InConversation("c1", events)
	if len(got) != 2 {
		t.Fatalf("c1 came back with %d turns, want its line and the retraction of it", len(got))
	}

	if got[1].Kind != SupervisorRetracted {
		t.Errorf("the retraction did not come with it: %+v", got)
	}

	if other := InConversation("c2", events); len(other) != 1 {
		t.Errorf("c2 came back with %d turns", len(other))
	}
}

// TestATitleIsOneLineAndNotAParagraph, because the list has one row per
// conversation.
func TestATitleIsOneLineAndNotAParagraph(t *testing.T) {
	long := "una primera línea que es bastante más larga de lo que cabe en una lista de conversaciones\ny una segunda"

	got := Conversations([]Event{said(time.Now().UTC(), "c1", long)})
	if len(got) != 1 {
		t.Fatal("no conversation")
	}

	if title := got[0].Title; len([]rune(title)) > titleCut+1 || title[len(title)-3:] != "…" {
		t.Errorf("the title is %q", title)
	}
}

// TestNewConversationIDNamesTheInstant. The stamp is the name: two turns
// are never typed in the same nanosecond.
func TestNewConversationIDNamesTheInstant(t *testing.T) {
	at := time.Now().UTC()

	if got := NewConversationID(at); got != Stamp(at) {
		t.Errorf("a new conversation id reads %q, want the stamp", got)
	}
}
