package task

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/repo"
)

// lastValue answers the way exec does: a later assignment wins over an
// earlier one for the same name.
func lastValue(env []string, name string) string {
	value := ""

	for _, entry := range env {
		if rest, ok := strings.CutPrefix(entry, name+"="); ok {
			value = rest
		}
	}

	return value
}

// The command line is asserted rather than run. Spawning orbit would need a
// built binary, a real repository and a real engine, and the thing worth
// pinning is exactly this: which subcommand a started run is, what it is
// told, and that it is put in a group of its own.
func TestAStartedRunIsTheSameSubcommandAPersonWouldType(t *testing.T) {
	tk := Task{ID: "ACME-1", Repo: repo.Repo{Name: "app", Path: "/repos/app"}}
	cmd := runCommand("/usr/local/bin/orbit", "/state", tk, "review", "")

	want := []string{"/usr/local/bin/orbit", "run", "-repo", "/repos/app", "-flow", "review", "ACME-1"}
	if len(cmd.Args) != len(want) {
		t.Fatalf("argv = %q, want %q", cmd.Args, want)
	}

	for i := range want {
		if cmd.Args[i] != want[i] {
			t.Fatalf("argv = %q, want %q", cmd.Args, want)
		}
	}

	if cmd.Path != "/usr/local/bin/orbit" {
		t.Errorf("the run would be spawned from %q, want the orbit binary that started it", cmd.Path)
	}

	if cmd.Dir != "/repos/app" {
		t.Errorf("working directory = %q, want the repository the task is against", cmd.Dir)
	}
}

func TestAStartedRunIsToldWhereTheStateIs(t *testing.T) {
	// A caller whose own root came from somewhere else entirely: what the
	// child must be told is the root the caller is reading, not the one the
	// environment happens to carry.
	t.Setenv("ORBIT_HOME", "/somewhere/else")

	tk := Task{ID: "ACME-1", Repo: repo.Repo{Name: "app", Path: "/repos/app"}}

	cmd := runCommand("/usr/local/bin/orbit", "/state", tk, "task", "")

	if got := lastValue(cmd.Env, "ORBIT_HOME"); got != "/state" {
		t.Errorf("ORBIT_HOME = %q, want /state — a run writing into another root is a task that vanished", got)
	}

	if lastValue(cmd.Env, "PATH") == "" {
		t.Error("the rest of the environment was dropped; a run with no PATH cannot find its engine")
	}
}

func TestAStartedRunGetsAProcessGroupOfItsOwn(t *testing.T) {
	tk := Task{ID: "ACME-1", Repo: repo.Repo{Name: "app", Path: "/repos/app"}}

	cmd := runCommand("/usr/local/bin/orbit", "/state", tk, "task", "")

	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setpgid {
		t.Error("the run would share the window's process group, so closing the window would SIGHUP it away mid-phase")
	}
}

// TestAnEngineOverrideReachesTheRun. A task handed to another engine is run
// with -engine, and a run that is not handed over carries no such flag: an
// empty one would mean "the flow's own", which is what its absence means.
func TestAnEngineOverrideReachesTheRun(t *testing.T) {
	tk := Task{ID: "ACME-9", Repo: repo.Repo{Path: "/code/ledger", Name: "ledger"}}

	handed := strings.Join(runCommand("/bin/orbit", "/state", tk, "task", "codex").Args, " ")
	if !strings.Contains(handed, "-engine codex") {
		t.Errorf("a handed run does not name the engine: %s", handed)
	}

	plain := strings.Join(runCommand("/bin/orbit", "/state", tk, "task", "").Args, " ")
	if strings.Contains(plain, "-engine") {
		t.Errorf("an ordinary run names an engine anyway: %s", plain)
	}
}
