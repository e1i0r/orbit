package cli

// `orbit chat` — the fifth way in, and the wiring behind it.
//
// None of this was tested. The terminal channel is the thing that makes the
// door real before any service is written, and the gate that decides who may
// command a bot read zero: a build that turned the wrong way there is a chat
// anybody who finds it can cancel a run from.

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/chat"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// ctxInto is a command's context writing into a buffer the test can read.
func ctxInto(out *bytes.Buffer) Context {
	return Context{Out: out, Err: out, Words: words.For("en")}
}

// TestTheTerminalHandsOverOneMessagePerLine. It is the same loop, the same
// gate and the same confirmations as a service, with the messages typed — so
// a service adapter that behaves differently from this one is a bug in the
// adapter.
func TestTheTerminalHandsOverOneMessagePerLine(t *testing.T) {
	to := atTheTerminal{in: strings.NewReader("/board\n\nhola\n")}

	var heard []chat.Message

	if err := to.Listen(context.Background(), func(m chat.Message) {
		heard = append(heard, m)
	}); err != nil {
		t.Fatalf("listen: %v", err)
	}

	if len(heard) != 3 {
		t.Fatalf("it heard %d messages, want one per line: %+v", len(heard), heard)
	}

	// One conversation and one account, because a terminal is one person.
	for _, m := range heard {
		if m.Where != "terminal" || m.Who != "operator" {
			t.Errorf("a line arrived as %+v, want the one conversation and the one account", m)
		}
	}

	if heard[0].Text != "/board" || heard[2].Text != "hola" {
		t.Errorf("the lines arrived as %+v", heard)
	}
}

// TestTheReaderLeavingIsNotAFailureOfTheInput. A chat is something somebody
// leaves open, and ending the loop with an error would put a failure in the
// log for somebody pressing ctrl-c.
func TestTheReaderLeavingIsNotAFailureOfTheInput(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	stop()

	to := atTheTerminal{in: strings.NewReader("/board\n")}

	if err := to.Listen(ctx, func(chat.Message) {
		t.Error("a line was handed over after the reader left")
	}); err != nil {
		t.Errorf("the reader leaving read as a failure: %v", err)
	}
}

// TestAnAnswerPrintedHasNoEmphasisInIt. A terminal has one and it belongs to
// whatever is drawing the screen, so a line printed into a pipe carries the
// words and not the markers this package puts round them.
func TestAnAnswerPrintedHasNoEmphasisInIt(t *testing.T) {
	var out bytes.Buffer

	to := atTheTerminal{out: &out}

	if err := to.Say(context.Background(), "terminal",
		chat.Strong+"ACME-1"+chat.Strong+" finished"); err != nil {
		t.Fatalf("say: %v", err)
	}

	said := out.String()
	if strings.Contains(said, chat.Strong) {
		t.Errorf("the emphasis was printed: %q", said)
	}

	if !strings.Contains(said, "ACME-1 finished") {
		t.Errorf("it printed %q, want the words it was handed", said)
	}
}

// TestItIsCalledWhatTheRecordCallsIt, because the name goes into the record
// and into the log.
func TestTheTerminalIsCalledTerminal(t *testing.T) {
	if got := (atTheTerminal{}).Name(); got != "terminal" {
		t.Errorf("the channel calls itself %q, want terminal", got)
	}
}

// TestBeingAtTheTerminalIsThePermission. Whoever is typing already has the
// machine, so there is nobody else for a gate to keep out.
func TestBeingAtTheTerminalIsThePermission(t *testing.T) {
	var out bytes.Buffer

	e := chat.Env{}

	to, err := reachedThrough(ctxInto(&out), "terminal", &e)
	if err != nil {
		t.Fatalf("reach the terminal: %v", err)
	}

	if to.Name() != "terminal" {
		t.Errorf("it reached %q, want the terminal", to.Name())
	}

	if e.Allowed == nil || !e.Allowed("anybody") {
		t.Error("the terminal turned its own reader away")
	}

	if !strings.Contains(out.String(), "/help") {
		t.Errorf("it opened with %q, want it to say how to get the list", out.String())
	}
}

// TestAChatWithNoTokenSaysWhereOneComesFrom. The token is a secret and lives
// in the environment; "no" on its own leaves somebody looking for a settings
// key that does not exist.
func TestAChatWithNoTokenSaysWhereOneComesFrom(t *testing.T) {
	t.Setenv(tokenEnv, "")

	var out bytes.Buffer

	e := chat.Env{}

	_, err := reachedThrough(ctxInto(&out), "telegram", &e)
	if err == nil {
		t.Fatal("a bot with no token was opened")
	}

	if !strings.Contains(err.Error(), tokenEnv) {
		t.Errorf("the refusal is %q, want it to name the variable", err)
	}

	if !strings.Contains(err.Error(), "BotFather") {
		t.Errorf("the refusal is %q, want it to say where a token comes from", err)
	}
}

// TestAChatOrbitDoesNotKnowSaysWhichItKnows, rather than opening nothing and
// answering that something went wrong.
func TestAChatOrbitDoesNotKnowSaysWhichItKnows(t *testing.T) {
	var out bytes.Buffer

	e := chat.Env{}

	_, err := reachedThrough(ctxInto(&out), "whatsapp", &e)
	if err == nil {
		t.Fatal("a channel orbit does not know was opened")
	}

	for _, want := range []string{"whatsapp", "terminal", "telegram"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal is %q, want %q in it", err, want)
		}
	}
}

// TestAMachineWithNoReaderYetAnswersNobodyAndSaysWhoYouAre.
//
// The one message worth answering a stranger with. Without it, setting this
// up means reading an HTTP API by hand to find a number — and a bot that
// argues with everybody who finds it is a bot telling strangers it is here.
func TestAMachineWithNoReaderYetAnswersNobodyAndSaysWhoYouAre(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	var out bytes.Buffer

	e := chat.Env{}

	said, err := allowedChat(ctxInto(&out), &e)
	if err != nil {
		t.Fatalf("read the gate: %v", err)
	}

	if !strings.Contains(said, "chat-id") {
		t.Errorf("it said %q, want it to say no chat-id is set", said)
	}

	if e.Allowed == nil || e.Allowed("12345") {
		t.Error("a machine with nobody allowed let somebody command it")
	}

	if e.Stranger == nil {
		t.Fatal("a machine with nobody allowed says nothing to the first person who writes")
	}

	to := e.Stranger(chat.Message{Who: "12345"})
	if !strings.Contains(to, "12345") || !strings.Contains(to, "settings set chat-id") {
		t.Errorf("it answered a stranger %q, want their id and the command that keeps it", to)
	}
}

// TestOnlyTheOneAccountMayCommandIt. A chat anybody can join is a chat
// anybody can cancel a run from.
func TestOnlyTheOneAccountMayCommandIt(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())
	settingsSet(t, func(c *store.Settings) { c.ChatID = "12345" })

	var out bytes.Buffer

	e := chat.Env{}

	said, err := allowedChat(ctxInto(&out), &e)
	if err != nil {
		t.Fatalf("read the gate: %v", err)
	}

	if !strings.Contains(said, "12345") {
		t.Errorf("it said %q, want it to name the account it answers", said)
	}

	if !e.Allowed("12345") {
		t.Error("the one account it answers was turned away")
	}

	if e.Allowed("54321") {
		t.Error("an account that is not the one was let in")
	}

	if e.Stranger != nil {
		t.Error("a machine that knows its reader still argues with strangers")
	}

	// Notify is off by default, so nothing speaks unasked.
	if e.Watch != nil || e.Tells != "" || e.Also != nil {
		t.Errorf("a chat with notify off still speaks unasked: tells=%q", e.Tells)
	}
}

// TestBeingToldThingsIsOneSwitchForTheLot. Whether you want to be
// interrupted is the question; where it reaches is a fact about what you
// have set up, and a switch per channel would ask the same question twice.
func TestBeingToldThingsIsOneSwitchForTheLot(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())
	settingsSet(t, func(c *store.Settings) { c.ChatID = "12345"; c.Notify = true })

	var out bytes.Buffer

	e := chat.Env{}

	if _, err := allowedChat(ctxInto(&out), &e); err != nil {
		t.Fatalf("read the gate: %v", err)
	}

	if e.Watch == nil {
		t.Error("a chat told to speak has nothing to watch")
	}

	// The same conversation the gate lets command it: one decision, not two.
	if e.Tells != "12345" {
		t.Errorf("it tells %q, want the account it answers", e.Tells)
	}

	if e.Also == nil {
		t.Error("the desktop half of the switch is off")
	}
}

// TestWatchingStartsWhereTheRecordIsNow. A chat opened this afternoon
// telling you about a task that finished on Tuesday is a chat that has
// taught you to scroll past it.
func TestWatchingStartsWhereTheRecordIsNow(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	appendEvent(t, "ACME-1", record.Event{Kind: record.TaskCreated, Text: "before the chat opened"})

	watch, err := watching()
	if err != nil {
		t.Fatalf("start watching: %v", err)
	}

	got, err := watch(context.Background())
	if err != nil {
		t.Fatalf("ask what happened: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("it opened holding %d things that happened before it: %+v", len(got), got)
	}

	appendEvent(t, "ACME-1", record.Event{Kind: record.TaskFinished, Text: "after"})

	got, err = watch(context.Background())
	if err != nil {
		t.Fatalf("ask again: %v", err)
	}

	if len(got) != 1 || got[0].Task != "ACME-1" || got[0].Event.Kind != record.TaskFinished {
		t.Fatalf("it answered %+v, want the one thing written since", got)
	}

	// And asked again with nothing written since, it answers nothing rather
	// than the same thing twice.
	again, err := watch(context.Background())
	if err != nil {
		t.Fatalf("ask a third time: %v", err)
	}

	if len(again) != 0 {
		t.Errorf("it said the same thing twice: %+v", again)
	}
}

// settingsSet writes the settings this machine runs with.
func settingsSet(t *testing.T, change func(*store.Settings)) {
	t.Helper()

	s, err := store.Open()
	if err != nil {
		t.Fatalf("open the store: %v", err)
	}

	defer func() {
		if err := s.Close(); err != nil {
			t.Errorf("close the store: %v", err)
		}
	}()

	cfg, err := s.Settings()
	if err != nil {
		t.Fatalf("read the settings: %v", err)
	}

	change(&cfg)

	if err := s.SaveSettings(cfg); err != nil {
		t.Fatalf("write the settings: %v", err)
	}
}

// appendEvent writes one event into the record of this machine.
func appendEvent(t *testing.T, id string, e record.Event) {
	t.Helper()

	s, err := store.Open()
	if err != nil {
		t.Fatalf("open the store: %v", err)
	}

	defer func() {
		if err := s.Close(); err != nil {
			t.Errorf("close the store: %v", err)
		}
	}()

	d, err := s.Record()
	if err != nil {
		t.Fatalf("open the record: %v", err)
	}

	if err := d.Append(id, e); err != nil {
		t.Fatalf("append %s: %v", e.Kind, err)
	}
}
