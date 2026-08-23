package cli

// The two answers this layer gives the window about engines.
//
// Nothing in this file runs a command. takeCommand builds the argv for an
// interactive `claude` session, and executing it would open a conversation
// with a real model and charge for it — so the command is inspected and
// never started, which is the only way a test of a command line can be free.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
)

func TestOnlyEnginesThatCarryOnASessionSayTheyCan(t *testing.T) {
	claude := engine.NewClaude()
	fake := engine.NewFake("")
	for _, c := range []struct {
		name    string
		engines map[string]engine.Engine
		want    bool
	}{
		{"the engine this program runs", map[string]engine.Engine{"claude": claude}, true},
		{"one that cannot", map[string]engine.Engine{"fake": fake}, false},
		{"one of each", map[string]engine.Engine{"claude": claude, "fake": fake}, false},
		{"none configured", map[string]engine.Engine{}, false},
		{"a nil in the map", map[string]engine.Engine{"claude": nil}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := canResume(c.engines); got != c.want {
				t.Errorf("canResume is %v, want %v", got, c.want)
			}
		})
	}
}

// The command line the window suspends itself for, asserted and never run.
func TestTakingTheKeyboardResumesAForkAndNotTheRunnersSession(t *testing.T) {
	cmd, err := takeCommand(engine.NewClaude(), "sess-1", "/w/.orbit/worktrees/ACME-1")
	if err != nil {
		t.Fatalf("takeCommand: %v", err)
	}
	if cmd == nil {
		t.Fatal("takeCommand built no command for a session that exists")
	}
	want := []string{"claude", "--resume", "sess-1", "--fork-session"}
	if strings.Join(cmd.Args, " ") != strings.Join(want, " ") {
		t.Errorf("argv is %q, want %q", cmd.Args, want)
	}
	if cmd.Dir != "/w/.orbit/worktrees/ACME-1" {
		t.Errorf("the session would open in %q, want the task's worktree", cmd.Dir)
	}
	if cmd.Process != nil {
		t.Fatal("takeCommand started the command; it must only build one")
	}
}

// --fork-session is the whole reason an interactive session is safe to open
// on a task a runner has already recorded, so it is asserted by name rather
// than only as part of the argv above.
func TestTheResumedSessionIsAlwaysForked(t *testing.T) {
	cmd, err := takeCommand(engine.NewClaude(), "sess-2", "/w")
	if err != nil {
		t.Fatalf("takeCommand: %v", err)
	}
	var forked bool
	for _, a := range cmd.Args {
		if a == "--fork-session" {
			forked = true
		}
	}
	if !forked {
		t.Error("the command would write into the session the runner recorded")
	}
}

func TestTakingTheKeyboardIsRefusedWhereThereIsNothingToResume(t *testing.T) {
	for _, c := range []struct {
		name    string
		eng     engine.Engine
		session string
		dir     string
		wantErr string
	}{
		{"no engine at all", nil, "sess-1", "/w", "needs an engine"},
		{"an engine that cannot resume", engine.NewFake(""), "sess-1", "/w", "cannot resume a session"},
		{"no worktree to open it in", engine.NewClaude(), "sess-1", "", "needs a worktree"},
	} {
		t.Run(c.name, func(t *testing.T) {
			cmd, err := takeCommand(c.eng, c.session, c.dir)
			if err == nil {
				t.Fatalf("takeCommand built %v, want a refusal", cmd)
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("the refusal is %q, want it to mention %q", err, c.wantErr)
			}
			if cmd != nil {
				t.Error("a refused takeCommand still handed back a command")
			}
		})
	}
}

// A task whose engine never reported a session id is not a failure, and the
// window says so in the reader's own language — so this hands back nothing
// twice over rather than an English sentence.
func TestATaskWithNoSessionIsNotARefusal(t *testing.T) {
	cmd, err := takeCommand(engine.NewClaude(), "", "/w")
	if err != nil {
		t.Fatalf("takeCommand: %v", err)
	}
	if cmd != nil {
		t.Errorf("takeCommand built %v for a task with no session", cmd.Args)
	}
}
