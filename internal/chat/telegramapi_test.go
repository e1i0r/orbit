package chat

// The bot API, against a stand-in service.
//
// What is under test is never Telegram: it is the offset that stops a
// message being answered twice, the fallback from HTML to plain text, and
// the refusal this service answers with inside a 200 — three pieces of this
// package's own behaviour that are invisible from any other door.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// call is one request the stand-in service was handed.
type call struct {
	method string
	body   map[string]any
}

// service is a stand-in Telegram, and the bot pointed at it. answer is asked
// what to reply with, per method, and every call is kept for the test to
// read afterwards.
type service struct {
	mu     sync.Mutex
	calls  []call
	answer answering
}

// answering is what the stand-in replies with: a status and a body, per
// method and per what was asked. Nil is an empty poll and an ok to
// everything else.
type answering func(method string, body map[string]any) (int, string)

// standIn puts a service in front of a bot and hands back both.
func standIn(t *testing.T, answer answering) (*Telegram, *service) {
	t.Helper()

	s := &service{answer: answer}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read the request body: %v", err)
		}

		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("the request body is not an object: %v", err)
		}

		// /bot<token>/<method> — the last segment is what was asked for.
		parts := strings.Split(r.URL.Path, "/")
		method := parts[len(parts)-1]

		s.mu.Lock()
		s.calls = append(s.calls, call{method: method, body: body})
		s.mu.Unlock()

		status, said := http.StatusOK, `{"ok":true,"result":[]}`
		if s.answer != nil {
			status, said = s.answer(method, body)
		}

		w.WriteHeader(status)

		if _, err := io.WriteString(w, said); err != nil {
			t.Errorf("answer the request: %v", err)
		}
	}))

	t.Cleanup(server.Close)

	bot := Bot("the-token")
	bot.base = server.URL
	bot.http = server.Client()

	return bot, s
}

// seen is every call the stand-in was handed, safe to read while the poll
// loop is still running.
func (s *service) seen() []call {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]call(nil), s.calls...)
}

// TestTheTokenTravelsInThePathAndNowhereElse. It is how this API is built,
// and a token in a header or a query string would be a token in somebody's
// proxy log.
func TestTheTokenTravelsInThePathAndNowhereElse(t *testing.T) {
	bot, _ := standIn(t, nil)

	var path string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path

		if _, err := io.WriteString(w, `{"ok":true}`); err != nil {
			t.Errorf("answer: %v", err)
		}
	}))
	defer server.Close()

	bot.base = server.URL
	bot.http = server.Client()

	if err := bot.Working(context.Background(), "42"); err != nil {
		t.Fatalf("show typing: %v", err)
	}

	if path != "/botthe-token/sendChatAction" {
		t.Errorf("it called %q, want the token in the path before the method", path)
	}
}

// TestAnAnswerIsSentAsHTML. The markup is what makes a listing line up on a
// phone, and it is asked for by name on the first attempt.
func TestAnAnswerIsSentAsHTML(t *testing.T) {
	bot, s := standIn(t, nil)

	if err := bot.Say(context.Background(), "42", "done"); err != nil {
		t.Fatalf("say: %v", err)
	}

	calls := s.seen()
	if len(calls) != 1 {
		t.Fatalf("it made %d calls, want one", len(calls))
	}

	if calls[0].method != "sendMessage" {
		t.Errorf("it called %q, want sendMessage", calls[0].method)
	}

	if calls[0].body["parse_mode"] != "HTML" {
		t.Errorf("it asked for %v, want HTML", calls[0].body["parse_mode"])
	}

	if calls[0].body["chat_id"] != "42" {
		t.Errorf("it answered into %v, want the conversation it was given", calls[0].body["chat_id"])
	}
}

// TestAnAnswerTheServiceRefusesGoesAgainWithNoMarkup. Formatting is worth
// having and it is not worth a message: a send the service rejects is an
// answer nobody sees.
func TestAnAnswerTheServiceRefusesGoesAgainWithNoMarkup(t *testing.T) {
	bot, s := standIn(t, func(_ string, body map[string]any) (int, string) {
		if body["parse_mode"] == "HTML" {
			return http.StatusBadRequest, `{"ok":false,"description":"can't parse entities"}`
		}

		return http.StatusOK, `{"ok":true}`
	})

	if err := bot.Say(context.Background(), "42", "<b>done</b>"); err != nil {
		t.Fatalf("the second attempt was not made: %v", err)
	}

	calls := s.seen()
	if len(calls) != 2 {
		t.Fatalf("it made %d calls, want the refused one and the plain one", len(calls))
	}

	if _, asked := calls[1].body["parse_mode"]; asked {
		t.Errorf("the second attempt still asked for markup: %v", calls[1].body)
	}
}

// TestAnAnswerRefusedTwiceIsReported. Nothing is hidden: the send failed and
// the reader has to be told, rather than the failure ending in a silence
// that reads as an answer delivered.
func TestAnAnswerRefusedTwiceIsReported(t *testing.T) {
	bot, _ := standIn(t, func(string, map[string]any) (int, string) {
		return http.StatusInternalServerError, `{"ok":false,"description":"down"}`
	})

	err := bot.Say(context.Background(), "42", "done")
	if err == nil {
		t.Fatal("a send refused twice answered as if it had landed")
	}

	if strings.Contains(err.Error(), "the-token") {
		t.Errorf("the refusal carries the token: %q", err)
	}
}

// TestTheMenuIsPublishedWithEveryCommandOnIt. It is what a reader sees the
// moment they type a slash, so a name dropped on the way is a verb that
// exists and cannot be found.
func TestTheMenuIsPublishedWithEveryCommandOnIt(t *testing.T) {
	bot, s := standIn(t, nil)

	all := []Command{{Name: "board", About: "what is on the board"}, {Name: "task", About: "one task"}}

	if err := bot.Announce(context.Background(), all); err != nil {
		t.Fatalf("publish the menu: %v", err)
	}

	calls := s.seen()
	if len(calls) != 1 || calls[0].method != "setMyCommands" {
		t.Fatalf("it made the calls %+v, want one setMyCommands", calls)
	}

	published, ok := calls[0].body["commands"].([]any)
	if !ok || len(published) != len(all) {
		t.Fatalf("it published %v, want both commands", calls[0].body["commands"])
	}

	first, ok := published[0].(map[string]any)
	if !ok || first["command"] != "board" || first["description"] != "what is on the board" {
		t.Errorf("the first command went as %v, want its name and what it is for", published[0])
	}
}

// TestARefusalInsideATwoHundredIsStillARefusal. This API answers "ok": false
// with a 200, so a reader that trusted the status would take an error for an
// empty poll and go round for ever without saying anything.
func TestARefusalInsideATwoHundredIsStillARefusal(t *testing.T) {
	bot, _ := standIn(t, func(string, map[string]any) (int, string) {
		return http.StatusOK, `{"ok":false,"description":"terminated by other getUpdates request"}`
	})

	_, err := bot.poll(context.Background())
	if err == nil {
		t.Fatal("a refusal inside a 200 read as an empty poll")
	}

	if !strings.Contains(err.Error(), "terminated by other getUpdates request") {
		t.Errorf("the refusal is %q, want it to carry what the service said", err)
	}
}

// TestWhatIsListenedForIsMessagesAndTheOnesAfterTheLast. The offset is what
// stops a message being answered twice, and the allowed kinds are what stops
// a bot deciding what to do about somebody joining a group.
func TestWhatIsListenedForIsMessagesAndTheOnesAfterTheLast(t *testing.T) {
	updates := `{"ok":true,"result":[
	  {"update_id":7,"message":{"text":"board","from":{"id":5},"chat":{"id":9}}},
	  {"update_id":8,"message":{"from":{"id":5},"chat":{"id":9}}}
	]}`

	var once sync.Once

	bot, s := standIn(t, func(string, map[string]any) (int, string) {
		said := `{"ok":true,"result":[]}`

		once.Do(func() { said = updates })

		return http.StatusOK, said
	})

	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	heard := make(chan Message, 4)

	done := make(chan error, 1)
	go func() { done <- bot.Listen(ctx, func(m Message) { heard <- m }) }()

	select {
	case m := <-heard:
		if m.Text != "board" || m.Where != "9" || m.Who != "5" {
			t.Errorf("it heard %+v, want the message with its conversation and its sender", m)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("nothing was heard")
	}

	// A second update with no text is not a message: a bot told about a
	// photo has nothing to answer, and handing an empty one to the desk
	// would have it reply to silence.
	select {
	case m := <-heard:
		t.Errorf("an update with no text was heard as %+v", m)
	case <-time.After(200 * time.Millisecond):
	}

	stop()

	if err := <-done; err != nil {
		t.Errorf("the reader leaving read as a failure: %v", err)
	}

	calls := s.seen()
	if len(calls) < 2 {
		t.Fatalf("it polled %d times, want a second poll after the first answered", len(calls))
	}

	kinds, ok := calls[0].body["allowed_updates"].([]any)
	if !ok || len(kinds) != 1 || kinds[0] != "message" {
		t.Errorf("it asked for %v, want messages only", calls[0].body["allowed_updates"])
	}

	// Nine, because the last update seen was eight: asking for the one
	// after it is how this service is told the eight was dealt with.
	offset, ok := calls[1].body["offset"].(float64)
	if !ok || offset != 9 {
		t.Errorf("the second poll asked from %v, want 9", calls[1].body["offset"])
	}
}

// TestAServiceThatIsMerelyUnreachableDoesNotEndTheLoop. Orbit is a program
// somebody leaves running, and a channel that stopped because a laptop
// closed its lid is a channel silently off for the rest of the afternoon.
func TestAServiceThatIsMerelyUnreachableDoesNotEndTheLoop(t *testing.T) {
	bot, _ := standIn(t, func(string, map[string]any) (int, string) {
		return http.StatusBadGateway, `{"ok":false,"description":"bad gateway"}`
	})

	ctx, stop := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer stop()

	nothing := func(Message) {
		t.Error("an unreachable service handed over a message")
	}

	if err := bot.Listen(ctx, nothing); err != nil {
		t.Errorf("an unreachable service ended the loop with %v, want it to keep waiting", err)
	}
}

// TestItIsCalledWhatTheRecordCallsIt. The name goes into the record and into
// the log, so it is one word and it is this one.
func TestItIsCalledWhatTheRecordCallsIt(t *testing.T) {
	if got := Bot("the-token").Name(); got != "telegram" {
		t.Errorf("the channel calls itself %q, want telegram", got)
	}
}
