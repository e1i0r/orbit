package supervisor

// The supervisor's thread, cut into conversations.

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/spoken"
	"github.com/e1i0r/orbit/internal/view"
)

// twoConversations is a thread with an old one and a newer one.
func twoConversations(t *testing.T) (State, Env, *held) {
	t.Helper()

	now := fixtureNow
	kept := &held{lines: []view.SupervisorLine{
		spoke(now.Add(-48*time.Hour), "c1", "operator", "Migración a SQLite"),
		spoke(now.Add(-47*time.Hour), "c1", "supervisor", "hecho"),
		spoke(now.Add(-time.Hour), "c2", "operator", "Revisar el módulo de pagos"),
	}}

	e := world(t, kept)
	e.NewID = func() string { return "c3" }

	return Open(0, e), e, kept
}

// TestTheScreenOpensOnTheConversationLastSpokenIn, which is the one somebody
// coming back was in the middle of.
func TestTheScreenOpensOnTheConversationLastSpokenIn(t *testing.T) {
	s, _, _ := twoConversations(t)

	if s.conversation != "c2" {
		t.Errorf("the screen opened on %q", s.conversation)
	}

	// And what it shows is that conversation's lines and no others.
	if len(s.lines) != 1 || s.lines[0].Text != "Revisar el módulo de pagos" {
		t.Errorf("the thread shows %+v", s.lines)
	}
}

// TestTheListTitlesEachConversationByWhatStartedIt, newest first, with how
// long each one is.
func TestTheListTitlesEachConversationByWhatStartedIt(t *testing.T) {
	s, e, _ := twoConversations(t)
	s = s.openConversationList()

	rows := strings.Join(s.conversationRows(20, 90, e), "\n")
	for _, want := range []string{"Revisar el módulo de pagos", "Migración a SQLite", "2 messages"} {
		if !strings.Contains(rows, want) {
			t.Errorf("the list does not say %q:\n%s", want, rows)
		}
	}

	convs := conversationsOf(s.all)
	if len(convs) != 2 || convs[0].id != "c2" || convs[1].turns != 2 {
		t.Errorf("the conversations are %+v", convs)
	}
}

// TestOpeningOneFromTheListReadsIt.
func TestOpeningOneFromTheListReadsIt(t *testing.T) {
	s, e, _ := twoConversations(t)
	s = s.openConversationList()

	s, _ = s.conversationKey(press("down"), e)
	s, _ = s.conversationKey(press("enter"), e)

	if s.conversation != "c1" || s.list {
		t.Fatalf("opening the older one left %q and list=%v", s.conversation, s.list)
	}

	if len(s.lines) != 2 {
		t.Errorf("the older conversation shows %d lines", len(s.lines))
	}
}

// TestANewConversationStartsEmptyAndIsWhatTheNextLineGoesInto.
func TestANewConversationStartsEmptyAndIsWhatTheNextLineGoesInto(t *testing.T) {
	s, e, kept := twoConversations(t)
	s.input = "half a sentence"

	s, _ = s.startConversation(e)
	if s.conversation != "c3" || len(s.lines) != 0 {
		t.Fatalf("the new conversation is %q with %d lines", s.conversation, len(s.lines))
	}

	if s.input != "" {
		t.Errorf("what was half-typed came along: %q", s.input)
	}

	s, _ = s.Say("lo primero que digo", e)

	wrote := ""
	if n := len(kept.wrote); n > 0 {
		wrote = kept.wrote[n-1].Conversation
	}

	if wrote != "c3" {
		t.Errorf("the line went into %q, want the conversation the screen has open", wrote)
	}
}

// TestRemovingAConversationTakesItOffTheListAndSaysTheRecordKeepsIt.
func TestRemovingAConversationTakesItOffTheListAndSaysTheRecordKeepsIt(t *testing.T) {
	s, e, kept := twoConversations(t)
	s = s.openConversationList()

	// The record marks and does not erase; what the screen sees next is a
	// thread the removed conversation has been folded out of.
	e.Forget = func(id string) error {
		kept.gone = append(kept.gone, id)
		kept.lines = kept.lines[:2]

		return nil
	}

	s, out := s.removeConversation(e)

	if len(kept.gone) != 1 || kept.gone[0] != "c2" {
		t.Fatalf("removed %v", kept.gone)
	}

	if convs := conversationsOf(s.all); len(convs) != 1 || convs[0].id != "c1" {
		t.Errorf("the list still holds %+v", convs)
	}

	// The screen was reading the one that went, so it lands on what is left.
	if s.conversation != "c1" {
		t.Errorf("the screen is on %q", s.conversation)
	}

	wantBand(t, out, "still in the record")
}

// TestTheConversationsAreReachableByGestureAsWellAsByKey.
//
// This screen is a text field: every printable key types into it, so a key
// needs a modifier — and a terminal that decides not to deliver one leaves
// the gesture unreachable. A word typed into the line always arrives.
func TestTheConversationsAreReachableByGestureAsWellAsByKey(t *testing.T) {
	s, e, _ := twoConversations(t)

	listed, _ := s.Say("/chats", e)
	if !listed.list || listed.input != "" {
		t.Errorf("/chats left list=%v input=%q", listed.list, listed.input)
	}

	fresh, _ := s.Say("/new", e)
	if fresh.conversation != "c3" {
		t.Errorf("/new opened %q", fresh.conversation)
	}

	// And both are offered while the word is being typed.
	s.input = "/"

	var found int

	for _, c := range s.completions(e) {
		if c.Text == spoken.ChatsWord || c.Text == spoken.NewWord {
			found++
		}
	}

	if found != 2 {
		t.Errorf("the completion list offers %d of the two conversation gestures", found)
	}
}

// TestTheControlKeysAreTakenInEveryShapeATerminalSendsThem, the bare control
// code included: a terminal speaking no modifier protocol sends 0x0C for ^L
// and sets no modifier at all.
func TestTheControlKeysAreTakenInEveryShapeATerminalSendsThem(t *testing.T) {
	shapes := []tea.KeyPressMsg{
		{Code: 'l', Mod: tea.ModCtrl},
		{Code: 'L', Mod: tea.ModCtrl},
		{Code: 'l', Mod: tea.ModCtrl | tea.ModShift},
		{Code: 12},
	}

	for _, k := range shapes {
		s, e, _ := twoConversations(t)

		after, _ := s.Key(k, e)
		if !after.list {
			t.Errorf("%v did not open the conversations", k)
		}

		if after.input != "" {
			t.Errorf("%v typed %q into the line", k, after.input)
		}
	}
}

// TestBriefAsksTheQuestionForYou, in the reader's own language, and sends it
// the way a typed sentence is sent so the answer lands in the thread.
func TestBriefAsksTheQuestionForYou(t *testing.T) {
	s, e, kept := twoConversations(t)

	asked := ""
	e.Ask = func(conversation, prompt string) tea.Cmd {
		asked = prompt

		return func() tea.Msg { return nil }
	}

	_, out := s.Say("/brief", e)
	if !out.Asking {
		t.Fatal("/brief asked nothing")
	}

	said := ""
	if n := len(kept.wrote); n > 0 {
		said = kept.wrote[n-1].Text
	}

	if !strings.Contains(said, "What happened") || said != asked {
		t.Errorf("the thread got %q and the engine %q", said, asked)
	}

	// What follows the word narrows the question rather than replacing it.
	if _, out = s.Say("/brief the coverage on ACME-1", e); !out.Asking {
		t.Fatal("a narrowed brief asked nothing")
	}

	if !strings.Contains(asked, "In particular: the coverage on ACME-1") {
		t.Errorf("the narrowed question is %q", asked)
	}
}
