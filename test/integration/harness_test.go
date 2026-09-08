//go:build integration

package integration

// The whole of Orbit, driven the way a person drives it.
//
// These tests build the real binary, put a stand-in engine on PATH under the
// name the flows ask for, make a real git repository with a real Go module
// in it, and then run `orbit` against it. Nothing is stubbed inside the
// program: the record is SQLite, the worktrees are git's, the gates are
// shell commands that really run. What is faked is the model, because a
// model is neither free nor the same twice.
//
// They are behind a build tag so that `go test ./...` stays a second's work.
// `make integration` is what runs them, and `make check` calls it.

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// built is where the binaries under test live for the whole run: one build
// for every test in the package rather than one per test.
var built struct {
	orbit string
	bin   string // the directory PATH is given, holding the stand-in engine
}

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "orbit-integration-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "make a directory for the binaries:", err)
		os.Exit(1)
	}

	if err := build(dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.RemoveAll(dir) //nolint:errcheck // going away either way
		os.Exit(1)
	}

	code := m.Run()

	os.RemoveAll(dir) //nolint:errcheck // a temporary directory the OS will take back
	os.Exit(code)
}

// build compiles orbit and the stand-in engine into one directory.
//
// The stand-in is installed under every name a shipped flow asks for, so a
// flow that names claude and one that names codex are both answered by it
// without the flow being edited for the test — a flow edited to be testable
// is a flow nobody tested.
func build(dir string) error {
	built.bin = dir
	built.orbit = filepath.Join(dir, "orbit")

	if err := goBuild(built.orbit, "./cmd/orbit"); err != nil {
		return err
	}

	engine := filepath.Join(dir, "fakeengine")
	if err := goBuild(engine, "./test/integration/fakeengine"); err != nil {
		return err
	}

	for _, name := range []string{"claude", "codex", "opencode", "agy"} {
		if err := os.Link(engine, filepath.Join(dir, name)); err != nil {
			return fmt.Errorf("install the stand-in engine as %q: %w", name, err)
		}
	}

	return nil
}

func goBuild(out, pkg string) error {
	cmd := exec.Command("go", "build", "-o", out, pkg)
	cmd.Dir = repoRoot()

	if said, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("build %s: %w: %s", pkg, err, said)
	}

	return nil
}

// repoRoot is the module's own directory, which is two above this package.
func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}

	return filepath.Dir(filepath.Dir(wd))
}

// orbit runs one orbit command against a board and answers what it printed.
//
// The environment is built from nothing rather than inherited: a test that
// reads the operator's own ~/.orbit, or their PATH, is a test that passes on
// one machine.
func (b board) orbit(t *testing.T, args ...string) (string, error) {
	t.Helper()

	cmd := exec.Command(built.orbit, args...)
	cmd.Dir = b.repo
	cmd.Env = b.env()

	var out bytes.Buffer

	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()

	return out.String(), err
}

// must runs a command that has to work for the test to mean anything.
func (b board) must(t *testing.T, args ...string) string {
	t.Helper()

	said, err := b.orbit(t, args...)
	if err != nil {
		t.Fatalf("orbit %s: %v\n%s", strings.Join(args, " "), err, said)
	}

	return said
}

// start runs a command that is meant to outlive the call, which is what a
// run held at a gate is: `orbit run` parks there and waits for the word,
// exactly as it does for a person who has walked away.
func (b board) start(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()

	cmd := exec.Command(built.orbit, args...)
	cmd.Dir = b.repo
	cmd.Env = b.env()

	var out bytes.Buffer

	cmd.Stdout, cmd.Stderr = &out, &out

	if err := cmd.Start(); err != nil {
		t.Fatalf("start orbit %s: %v", strings.Join(args, " "), err)
	}

	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill() //nolint:errcheck // the test is over either way
			_ = cmd.Wait()         //nolint:errcheck // and so is the process
		}
	})

	return cmd
}

// waitFor blocks until the record carries an event of one kind, and fails
// the test rather than hanging when it never does.
func (b board) waitFor(t *testing.T, id, kind string, within time.Duration) {
	t.Helper()

	for waited := time.Duration(0); waited < within; waited += pollEvery {
		if holds(b.record(t, id), kind) {
			return
		}

		time.Sleep(pollEvery)
	}

	t.Fatalf("no %s in the record of %s after %v:\n%v", kind, id, within, kinds(b.record(t, id)))
}

// pollEvery is how often the record is asked, and gateWait how long a test
// waits for a run to reach a gate before calling it hung.
const (
	pollEvery = 200 * time.Millisecond
	gateWait  = 60 * time.Second
)

// env is what every orbit in these tests is given: nothing of the
// operator's, and everything the run needs.
func (b board) env() []string {
	return []string{
		"PATH=" + built.bin + string(os.PathListSeparator) + os.Getenv("PATH"),
		"HOME=" + b.home,
		"ORBIT_HOME=" + b.state,
		"ORBIT_FAKE_SCRIPT=" + b.script,
		"ORBIT_FAKE_STATE=" + b.turns,
		"GOCACHE=" + os.Getenv("GOCACHE"),
		"GOPATH=" + os.Getenv("GOPATH"),
		"GOMODCACHE=" + os.Getenv("GOMODCACHE"),
	}
}
