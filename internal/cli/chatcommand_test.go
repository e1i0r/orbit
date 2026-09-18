package cli

// `orbit chat` as a command: the flag, the directory it watches, and the
// loop ending with its input.
//
// Apart from chat_test.go because that file is the channel and the gate one
// piece at a time, and this is the whole of it wired together — which is a
// different question and the one that catches a piece wired to the wrong
// neighbour.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// TestTheChatOpensAndClosesWithItsInput. The whole command, from the flag to
// the loop: a terminal chat ends when its input does, which is what makes it
// runnable at all in a suite — and what proves the channel, the gate, the
// world and the supervisor are wired to each other rather than only being
// wired correctly one at a time.
func TestTheChatOpensAndClosesWithItsInput(t *testing.T) {
	root, _ := workspace(t)

	code, out, errs := run(t, "chat", root)
	if code != 0 {
		t.Fatalf("orbit chat answered %d: %s%s", code, out, errs)
	}

	if !strings.Contains(out, "/help") {
		t.Errorf("it opened with %q, want it to say how to get the list", out)
	}
}

// TestAChatOnAServiceThisBuildDoesNotKnowRefusesBeforeItOpens. The refusal
// comes out of the command rather than out of a loop nobody can see, so a
// reader who mistyped the name is told at once.
func TestAChatOnAServiceThisBuildDoesNotKnowRefusesBeforeItOpens(t *testing.T) {
	root, _ := workspace(t)

	code, _, errs := run(t, "chat", "-on", "whatsapp", root)
	if code == 0 {
		t.Fatal("a chat on a service orbit does not know was opened")
	}

	for _, want := range []string{"whatsapp", "terminal", "telegram"} {
		if !strings.Contains(errs, want) {
			t.Errorf("it said %q, want %q in it", errs, want)
		}
	}
}

// TestAChatWatchesMoreThanOneDirectoryOnlyIfToldOne. Two directories is a
// reader meaning something the command cannot do, and picking one of them is
// worse than saying so.
func TestAChatWatchesMoreThanOneDirectoryOnlyIfToldOne(t *testing.T) {
	root, _ := workspace(t)

	if code, _, _ := run(t, "chat", root, root); code == 0 {
		t.Error("a chat over two directories was opened")
	}
}

// TestTheWorldIsOpenedForOneMessageAndClosedAgain.
//
// Fresh each time rather than held: the store has one write lock, and a chat
// left open all afternoon holding it is a chat that stops every run on the
// machine. Opening it costs a file handle and a moment, once per thing
// somebody asks for.
func TestTheWorldIsOpenedForOneMessageAndClosedAgain(t *testing.T) {
	root, _ := workspace(t)

	open := worldFor(root, words.For("en"))

	for range 3 {
		w, done, err := open()
		if err != nil {
			t.Fatalf("open the machine: %v", err)
		}

		if w == nil {
			t.Fatal("it opened nothing")
		}

		// The repositories under the directory are found: a chat that
		// answered about an empty board would be answering about a machine
		// it never looked at.
		b, err := w.Board()
		if err != nil {
			t.Fatalf("read the board: %v", err)
		}

		if len(b.RepoList) == 0 {
			t.Error("the chat looked for repositories and found none")
		}

		done()
	}
}

// TestAWorldOverSomewhereThatIsNotThereSaysSo, rather than answering with an
// empty board that reads as a machine with nothing on it.
func TestAWorldOverSomewhereThatIsNotThereSaysSo(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	_, done, err := worldFor(filepath.Join(t.TempDir(), "nowhere"), words.For("en"))()
	if err == nil {
		done()
		t.Fatal("a chat opened over a directory that is not there")
	}

	if !strings.Contains(err.Error(), "repositories") {
		t.Errorf("it said %q, want it to say what it was doing", err)
	}
}

// TestTheSupervisorAnswersThroughTheEngineTheSettingsName. A chat has no dial
// to turn — the window picks an engine per conversation and a phone has
// nowhere to show the choice — so the standing one is the honest answer.
func TestTheSupervisorAnswersThroughTheEngineTheSettingsName(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	dir := t.TempDir()

	said := `{"type":"result","result":"nothing is stuck","session_id":"s1"}`
	if err := os.WriteFile(filepath.Join(dir, "claude"),
		[]byte("#!/bin/sh\ncat <<'EOF'\n"+said+"\nEOF\n"), 0o755); err != nil {
		t.Fatalf("write the stand-in: %v", err)
	}

	t.Setenv("PATH", dir+":/bin:/usr/bin")

	settingsSet(t, func(c *store.Settings) { c.Engine = "claude" })

	answer, err := supervising()(t.Context(), "what is stuck")
	if err != nil {
		t.Fatalf("ask the supervisor: %v", err)
	}

	if !strings.Contains(answer, "nothing is stuck") {
		t.Errorf("it answered %q, want what the engine said", answer)
	}
}

// TestAChatWithNoEngineChosenSaysSoRatherThanPickingOne. It spends money —
// one model run per sentence somebody types — and an engine nobody chose is
// a bill nobody expected.
func TestAChatWithNoEngineChosenSaysSoRatherThanPickingOne(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())

	if _, err := supervising()(t.Context(), "what is stuck"); err == nil {
		t.Error("a chat with no engine chosen ran one anyway")
	}
}
