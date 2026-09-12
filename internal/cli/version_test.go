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

	want := "    _____      orbit dev\n" +
		"   /     \\     orbit — a cockpit for supervising coding agents\n" +
		"--(   o   )--\n" +
		"   \\_____/\n"
	if out != want {
		t.Errorf("version printed %q, want %q", out, want)
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

	if out != "    _____      orbit 1.2.3\n"+
		"   /     \\     orbit — a cockpit for supervising coding agents\n"+
		"--(   o   )--\n"+
		"   \\_____/\n" {
		t.Errorf("version printed %q, want the banner with %q", out, "orbit 1.2.3")
	}
}

func TestVersionBannerCarriesTheMark(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	_, out, _ := run(t, "version")

	// The body with the rings on. This holds the shape rather than the
	// exact drawing: re-aligning the art must stay green, dropping the
	// mark must not.
	if !strings.Contains(out, "--(   o   )--") {
		t.Errorf("the version banner lost the rings around it:\n%s", out)
	}

	if !strings.HasPrefix(out, "    _____") {
		t.Errorf("the version banner lost the body the rings go around:\n%s", out)
	}
}

// TestVersionBannerMeasuresTheSameEverywhere. A glyph the terminal renders
// wider than the code counts slides its line out of column — the banner
// once drew its body as ◉, which reads two cells on fonts that render it
// so, and the line beside it came out shifted. Everything before the words
// on a line is plain ASCII, the one width every terminal agrees on.
func TestVersionBannerMeasuresTheSameEverywhere(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	_, out, _ := run(t, "version")

	for _, line := range strings.Split(out, "\n") {
		i := strings.Index(line, "orbit")
		if i < 0 {
			continue
		}

		for _, r := range line[:i] {
			if r > 127 {
				t.Errorf("line %q draws %q before its words: ASCII only, or it will not line up", line, r)
			}
		}
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
