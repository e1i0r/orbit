package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAFailedCommandIsWrittenDown.
//
// Sixty-three logger calls lived in this package and sixty-one of them went
// nowhere: only top and mcp opened the log, so `orbit run`, cancel, merge,
// pr and reconcile each wrote their failures to a nil global logger, which
// drops them without a word. The command that puts an agent on a repository
// left no trace of why it did not.
//
// Both files are checked, because the errors file is the one a reader tails.
func TestAFailedCommandIsWrittenDown(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ORBIT_HOME", home)

	var out, errOut bytes.Buffer
	if code := Run([]string{"cancel", "-repo", filepath.Join(home, "nowhere"), "ABC-1"}, &out, &errOut); code == 0 {
		t.Fatal("cancel answered 0 for a repository that is not there")
	}

	for _, name := range []string{"orbit.log", "errors.log"} {
		path := filepath.Join(home, "logs", name)

		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}

		if !strings.Contains(string(b), "cli/cancel") {
			t.Errorf("%s does not say the command failed: %q", name, b)
		}
	}
}

// TestACommandThatWorksSaysNothingToTheErrorsFile. The errors file is only
// worth tailing if a green run leaves it alone.
func TestACommandThatWorksSaysNothingToTheErrorsFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ORBIT_HOME", home)

	var out, errOut bytes.Buffer
	if code := Run([]string{"repos", t.TempDir()}, &out, &errOut); code != 0 {
		t.Fatalf("repos answered %d: %s", code, errOut.String())
	}

	b, err := os.ReadFile(filepath.Join(home, "logs", "errors.log"))
	if err != nil {
		t.Fatalf("read errors.log: %v", err)
	}

	if len(bytes.TrimSpace(b)) != 0 {
		t.Errorf("a command that worked wrote to the errors file: %q", b)
	}

	all, err := os.ReadFile(filepath.Join(home, "logs", "orbit.log"))
	if err != nil {
		t.Fatalf("read orbit.log: %v", err)
	}

	if !strings.Contains(string(all), "[INFO] [cli/repos] ran") {
		t.Errorf("the log does not say which command was run: %q", all)
	}
}

// TestACommandThatLogsNothingOfItsOwnIsStillWrittenDown.
//
// Nine of the twenty-two commands have no logger call anywhere in them —
// flows, list, read, repos, resume, settings, show, top and upgrade — so
// before the dispatcher wrote their failures down, those nine failed and
// left no file anywhere any the wiser. show is one of them.
func TestACommandThatLogsNothingOfItsOwnIsStillWrittenDown(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ORBIT_HOME", home)

	var out, errOut bytes.Buffer
	if code := Run([]string{"show", "-repo", filepath.Join(home, "nowhere"), "ABC-1"}, &out, &errOut); code == 0 {
		t.Fatal("show answered 0 for a repository that is not there")
	}

	b, err := os.ReadFile(filepath.Join(home, "logs", "errors.log"))
	if err != nil {
		t.Fatalf("read errors.log: %v", err)
	}

	if !strings.Contains(string(b), "[ERROR] [cli/show]") {
		t.Errorf("the errors file does not say show failed: %q", b)
	}
}
