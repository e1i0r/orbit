package cli

// `orbit chat` as a command: the flag, the directory it watches, and the
// loop ending with its input.
//
// Apart from chat_test.go because that file is the channel and the gate one
// piece at a time, and this is the whole of it wired together — which is a
// different question and the one that catches a piece wired to the wrong
// neighbour.

import (
	"strings"
	"testing"
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
