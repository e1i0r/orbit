package cli

// language() and openBoth() (cli.go) branches nothing else in this package
// reaches: RootPath failing because neither $ORBIT_HOME nor $HOME name a
// place to look, and store.Open failing because $ORBIT_HOME cannot be
// created at all — a regular file sitting where the directory would go.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLanguageAnswersEmptyWhenNoHomeCanBeFound(t *testing.T) {
	t.Setenv("ORBIT_HOME", "")
	t.Setenv("HOME", "")

	// Any command exercises language() through Run's own ctx.Words; the
	// usage screen is printed in English (language() answered "") rather
	// than the command failing outright.
	code, out, _ := run(t, "help")
	if code != 0 {
		t.Errorf("help with no home exited %d, want 0", code)
	}
	if out == "" {
		t.Error("usage was not printed even though language() has a fallback")
	}
}

func TestOpenBothFailsWhenTheStateRootCannotBeCreated(t *testing.T) {
	root, _ := workspace(t)
	repoDir := filepath.Join(root, "payments")

	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ORBIT_HOME", filepath.Join(blocker, "orbit"))

	code, _, errOut := run(t, "list", "-repo", repoDir)
	if code == 0 {
		t.Error("list with an unmakeable state root exited 0")
	}
	if errOut == "" {
		t.Error("list failed silently with an unmakeable state root")
	}
}

// language() has two ways to fail after os.Stat(root) has already found
// something there: RootPath itself (covered above, where nothing names a
// home at all) and store.Open, which re-resolves the same root and then
// tries to MkdirAll it. Naming ORBIT_HOME after a plain file rather than a
// directory that merely can't be created under lets os.Stat succeed — the
// file is there — while store.Open's own MkdirAll fails on it.
func TestLanguageAnswersEmptyWhenTheStateRootIsAFile(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ORBIT_HOME", blocker)

	code, out, _ := run(t, "help")
	if code != 0 {
		t.Errorf("help with blocked root exited %d, want 0", code)
	}
	if out == "" {
		t.Error("usage was not printed even though language() has a fallback")
	}
}
