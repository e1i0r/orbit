package engine

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAnEngineIsFoundWhereItsInstallerPutIt.
//
// An engine screen that asks exec.LookPath and nothing else draws an engine
// installed somewhere PATH does not mention as "[setup required]". That is
// not a corner: opencode's installer puts the binary in ~/.opencode/bin and
// adds that line to a shell profile, so any Orbit started from a terminal
// older than the install sees a machine without opencode — while opencode
// runs in the next pane.
func TestAnEngineIsFoundWhereItsInstallerPutIt(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "")

	for _, c := range []struct {
		eng Engine
		dir string
	}{
		{OpenCode{}, ".opencode/bin"},
		{Claude{}, ".local/bin"},
		{Codex{}, ".codex/bin"},
	} {
		name := c.eng.Name()

		if _, err := c.eng.Locate(); err == nil {
			t.Errorf("%s was located on a machine that does not have it", name)
		}

		bin := filepath.Join(home, c.dir, name)
		if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}

		got, err := c.eng.Locate()
		if err != nil {
			t.Errorf("%s installed at %s was not found: %v", name, bin, err)
			continue
		}

		if got != bin {
			t.Errorf("%s located at %q, want %q", name, got, bin)
		}
	}
}

// TestAFileThatIsNotExecutableIsNotAnEngine. A directory named opencode, or
// a half-downloaded file with no execute bit, is not something to draw dials
// for: reporting it as installed trades "[setup required]" for a run that
// dies at exec, which is the worse of the two.
func TestAFileThatIsNotExecutableIsNotAnEngine(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "")

	dir := filepath.Join(home, ".opencode", "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "opencode"), []byte("half a download"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got, err := (OpenCode{}).Locate(); err == nil {
		t.Errorf("a file with no execute bit was reported as an engine at %q", got)
	}

	if err := os.RemoveAll(filepath.Join(dir, "opencode")); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(dir, "opencode"), 0o755); err != nil {
		t.Fatal(err)
	}

	if got, err := (OpenCode{}).Locate(); err == nil {
		t.Errorf("a directory was reported as an engine at %q", got)
	}
}

// TestPathStillWins. The fallback is a fallback: an engine on PATH is found
// there, at the absolute path Run will execute.
func TestPathStillWins(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	t.Setenv("HOME", t.TempDir())

	bin := filepath.Join(dir, "codex")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := (Codex{}).Locate()
	if err != nil {
		t.Fatalf("codex on PATH was not found: %v", err)
	}

	if got != bin {
		t.Errorf("located %q, want the absolute path %q", got, bin)
	}
}
