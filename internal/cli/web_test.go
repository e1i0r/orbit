package cli

// orbit web, up to the moment it starts serving.
//
// Nothing here holds a port open. `orbit web` runs until the terminal it was
// started in ends: there is no shutdown path, because a server whose lifetime
// is a keystroke has no reason to grow one, and Serve returns nil only on a
// close nobody ever asks for. A test that reached it would reach it in a
// goroutine that outlives the test and never stops.
//
// What this file asks is everything before that line — that the flags are
// read, the directory resolved to something a reader on another machine can
// act on, the board read, the window found inside the binary — and that an
// address already taken is refused in words rather than served over. That
// refusal is the interesting one: it is the last thing checked before the
// listen, so every step ahead of it has already succeeded, and a failure
// there is a failure about the address and about nothing else.

import (
	"net"
	"path/filepath"
	"strings"
	"testing"
)

// TestWebRefusesAnAddressItCannotTake.
//
// The address is held for the length of the test and released by cleanup, so
// what is refused is an address that is taken now — not one that happened to
// be free a moment before the binary reached for it.
func TestWebRefusesAnAddressItCannotTake(t *testing.T) {
	root, _ := workspace(t)
	quietLocale(t)

	held, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	t.Cleanup(func() { held.Close() })

	addr := held.Addr().String()

	code, out, errOut := run(t, "web", "-addr", addr, filepath.Join(root, "payments"))
	if code == 0 {
		t.Fatal("web served over an address something else already held")
	}

	if strings.Contains(out, "http://") {
		t.Errorf("web announced a server it never started:\n%s", out)
	}

	if !strings.Contains(errOut, addr) {
		t.Errorf("the refusal does not name the address it could not take:\n%s", errOut)
	}
}

// TestWebRefusesAFlagNobodyGaveAValue.
//
// Refused by the flag set and not by the server: a command that went on to
// listen on the default address because a flag was mistyped would be a
// command that silently ignored what it was told.
func TestWebRefusesAFlagNobodyGaveAValue(t *testing.T) {
	workspace(t)
	quietLocale(t)

	code, out, errOut := run(t, "web", "-addr")
	if code == 0 {
		t.Fatal("web started with a flag nobody gave a value")
	}

	if strings.Contains(out, "http://") {
		t.Errorf("web announced a server it never started:\n%s", out)
	}

	if !strings.Contains(errOut, "addr") {
		t.Errorf("the refusal does not name the flag:\n%s", errOut)
	}
}
