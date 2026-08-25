package cli

import (
	"strings"
	"testing"
)

// TestVersionPrintsOrbitAndTheVersion is the default a checkout with no
// build-time override produces: `go build` and `go run` both leave Version
// at "dev", so that is what a contributor's own binary says.
func TestVersionPrintsOrbitAndTheVersion(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())
	code, out, errOut := run(t, "version")
	if code != 0 {
		t.Fatalf("version exited %d: %s", code, errOut)
	}
	if out != "orbit dev\n" {
		t.Errorf("version printed %q, want %q", out, "orbit dev\n")
	}
}

// TestVersionPrintsAnOverriddenVersion covers a release built with
// -ldflags "-X ...cli.Version=...": the command prints whatever the build
// set Version to, not a string it works out for itself.
func TestVersionPrintsAnOverriddenVersion(t *testing.T) {
	old := Version
	Version = "1.2.3"
	t.Cleanup(func() { Version = old })

	t.Setenv("ORBIT_HOME", t.TempDir())
	code, out, errOut := run(t, "version")
	if code != 0 {
		t.Fatalf("version exited %d: %s", code, errOut)
	}
	if out != "orbit 1.2.3\n" {
		t.Errorf("version printed %q, want %q", out, "orbit 1.2.3\n")
	}
}

func TestVersionHelpFlagShowsTheShapeOfTheCommand(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())
	code, out, errOut := run(t, "version", "-h")
	if code != 0 {
		t.Errorf("version -h exited %d, want 0: %s", code, errOut)
	}
	if errOut != "" {
		t.Errorf("version -h wrote to stderr: %s", errOut)
	}
	if !strings.Contains(out, "orbit version") {
		t.Errorf("version -h does not show the shape of the command:\n%s", out)
	}
}

func TestVersionRejectsAnUnknownFlag(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())
	code, _, errOut := run(t, "version", "-repo", ".")
	if code == 0 {
		t.Error("version -repo . exited 0, want a refusal — version takes no flags")
	}
	if !strings.Contains(errOut, "-repo") {
		t.Errorf("the error does not name the flag that was wrong:\n%s", errOut)
	}
}
