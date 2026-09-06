package ui

// The supervisor's thread, cut into conversations.

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// spoke is one line of a conversation.
func spoke(at time.Time, conv, by, text string) view.SupervisorLine {
	return view.SupervisorLine{At: at, Kind: "supervisor.message", By: by, Text: text, Conversation: conv}
}

// twoConversations is a thread with an old one and a newer one.
func twoConversations(t *testing.T) (Model, *threadReader) {
	t.Helper()

	now := time.Now().UTC()
	r := &threadReader{lines: []view.SupervisorLine{
		spoke(now.Add(-48*time.Hour), "c1", "operator", "Migración a SQLite"),
		spoke(now.Add(-47*time.Hour), "c1", "supervisor", "hecho"),
		spoke(now.Add(-time.Hour), "c2", "operator", "Revisar el módulo de pagos"),
	}}

	m, _ := testModel(t, 100, 30)
	m.opts.Reader = r
	m.opts.NewConversation = func() string { return "c3" }

	return m.openSupervisor(), r
}

// TestTheScreenOpensOnTheConversationLastSpokenIn, which is the one somebody
// coming back was in the middle of.
func TestTheScreenOpensOnTheConversationLastSpokenIn(t *testing.T) {
	m, _ := twoConversations(t)

	if m.supervisor.conversation != "c2" {
		t.Errorf("the screen opened on %q", m.supervisor.conversation)
	}

	// And what it shows is that conversation's lines and no others.
	if len(m.supervisor.lines) != 1 || m.supervisor.lines[0].Text != "Revisar el módulo de pagos" {
		t.Errorf("the thread shows %+v", m.supervisor.lines)
	}
}

// TestTheListTitlesEachConversationByWhatStartedIt, newest first, with how
// long each one is.
func TestTheListTitlesEachConversationByWhatStartedIt(t *testing.T) {
	m, _ := twoConversations(t)
	m = m.openConversationList()

	rows := strings.Join(m.conversationRows(20, 90), "\n")
	for _, want := range []string{"Revisar el módulo de pagos", "Migración a SQLite", "2 messages"} {
		if !strings.Contains(rows, want) {
			t.Errorf("the list does not say %q:\n%s", want, rows)
		}
	}

	convs := conversationsOf(m.supervisor.all)
	if len(convs) != 2 || convs[0].id != "c2" || convs[1].turns != 2 {
		t.Errorf("the conversations are %+v", convs)
	}
}

// TestOpeningOneFromTheListReadsIt.
func TestOpeningOneFromTheListReadsIt(t *testing.T) {
	m, _ := twoConversations(t)
	m = m.openConversationList()

	next, _ := m.conversationKey(tea.KeyPressMsg{Code: tea.KeyDown})
	next, _ = asModel(t, next).conversationKey(tea.KeyPressMsg{Code: tea.KeyEnter})

	after := asModel(t, next)
	if after.supervisor.conversation != "c1" || after.supervisor.list {
		t.Fatalf("opening the older one left %q and list=%v", after.supervisor.conversation, after.supervisor.list)
	}

	if len(after.supervisor.lines) != 2 {
		t.Errorf("the older conversation shows %d lines", len(after.supervisor.lines))
	}
}

// TestANewConversationStartsEmptyAndIsWhatTheNextLineGoesInto.
func TestANewConversationStartsEmptyAndIsWhatTheNextLineGoesInto(t *testing.T) {
	m, _ := twoConversations(t)
	m.supervisor.input = "half a sentence"

	m = m.startConversation()
	if m.supervisor.conversation != "c3" || len(m.supervisor.lines) != 0 {
		t.Fatalf("the new conversation is %q with %d lines", m.supervisor.conversation, len(m.supervisor.lines))
	}

	if m.supervisor.input != "" {
		t.Errorf("what was half-typed came along: %q", m.supervisor.input)
	}

	wrote := ""
	m.opts.RecordSupervisor = func(conversation, by, channel, message string) error {
		wrote = conversation

		return nil
	}

	m, _ = m.sendSupervisorMessage("lo primero que digo")

	if wrote != "c3" {
		t.Errorf("the line went into %q, want the conversation the screen has open", wrote)
	}
}

// TestRemovingAConversationTakesItOffTheListAndSaysTheRecordKeepsIt.
func TestRemovingAConversationTakesItOffTheListAndSaysTheRecordKeepsIt(t *testing.T) {
	m, r := twoConversations(t)
	m = m.openConversationList()

	gone := ""
	m.opts.RemoveConversation = func(id string) error {
		gone = id
		// The record marks and does not erase; what the screen sees next is
		// a thread the removed conversation has been folded out of.
		r.lines = r.lines[:2]

		return nil
	}

	m = m.removeConversation()

	if gone != "c2" {
		t.Fatalf("removed %q", gone)
	}

	if convs := conversationsOf(m.supervisor.all); len(convs) != 1 || convs[0].id != "c1" {
		t.Errorf("the list still holds %+v", convs)
	}

	// The screen was reading the one that went, so it lands on what is left.
	if m.supervisor.conversation != "c1" {
		t.Errorf("the screen is on %q", m.supervisor.conversation)
	}

	if !strings.Contains(m.message, "record") {
		t.Errorf("the bar says %q, want it to say the record still has it", m.message)
	}
}

// TestTheConversationsAreReachableByGestureAsWellAsByKey.
//
// This screen is a text field: every printable key types into it, so a key
// needs a modifier — and a terminal that decides not to deliver one leaves
// the gesture unreachable. A word typed into the line always arrives.
func TestTheConversationsAreReachableByGestureAsWellAsByKey(t *testing.T) {
	m, _ := twoConversations(t)

	listed, _ := m.sendSupervisorMessage("/chats")
	if !listed.supervisor.list || listed.supervisor.input != "" {
		t.Errorf("/chats left list=%v input=%q", listed.supervisor.list, listed.supervisor.input)
	}

	fresh, _ := m.sendSupervisorMessage("/new")
	if fresh.supervisor.conversation != "c3" {
		t.Errorf("/new opened %q", fresh.supervisor.conversation)
	}

	// And both are offered while the word is being typed.
	m.supervisor.input = "/"

	var found int

	for _, c := range m.completions() {
		if c.Text == chatsWord || c.Text == newWord {
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
		m, _ := twoConversations(t)

		next, _ := m.supervisorKey(k)

		after := asModel(t, next)
		if !after.supervisor.list {
			t.Errorf("%v did not open the conversations", k)
		}

		if after.supervisor.input != "" {
			t.Errorf("%v typed %q into the line", k, after.supervisor.input)
		}
	}
}

// TestBriefAsksTheQuestionForYou, in the reader's own language, and sends it
// the way a typed sentence is sent so the answer lands in the thread.
func TestBriefAsksTheQuestionForYou(t *testing.T) {
	m, _ := twoConversations(t)

	said := ""
	m.opts.RecordSupervisor = func(conversation, by, channel, message string) error {
		said = message

		return nil
	}

	asked := ""
	m.opts.AskSupervisor = func(engineName, conversation, prompt string) (string, error) {
		asked = prompt

		return "", nil
	}

	next, cmd := m.sendSupervisorMessage("/brief")
	if cmd == nil {
		t.Fatal("/brief asked nothing")
	}

	cmd()

	if !strings.Contains(said, "What happened") || said != asked {
		t.Errorf("the thread got %q and the engine %q", said, asked)
	}

	if !next.supervisorBusy {
		t.Error("the window does not say it is waiting on an answer")
	}

	// What follows the word narrows the question rather than replacing it.
	if _, cmd = m.sendSupervisorMessage("/brief the coverage on ACME-1"); cmd != nil {
		cmd()
	}

	if !strings.Contains(said, "In particular: the coverage on ACME-1") {
		t.Errorf("the narrowed question is %q", said)
	}
}
