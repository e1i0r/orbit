package cli

// What `orbit upgrade` does when the release, the network or the archive is
// not what it hoped for.

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestAnArchiveThatIsNotTheOnePublishedIsNotInstalled is the check reaching
// the reader through selfUpdate rather than on its own: the archive arrives
// whole, the release vouches for different bytes, and nothing is written.
func TestAnArchiveThatIsNotTheOnePublishedIsNotInstalled(t *testing.T) {
	name := "orbit_2.0.0_test.tar.gz"
	bin, _ := published(t, name, archiveOf(t, "orbit", []byte("#!/bin/sh\n")))

	sums := asset{Name: "checksums.txt", URL: serving(t, func(w http.ResponseWriter) {
		fmt.Fprintf(w, "%x  %s\n", sha256.Sum256([]byte("a different release")), name)
	})}

	err := selfUpdate(t.Context(), bin, sums)
	if err == nil {
		t.Fatal("an archive the release did not vouch for was installed over orbit")
	}

	if !strings.Contains(err.Error(), "hashes to") {
		t.Errorf("the refusal does not say the archive is not the published one: %v", err)
	}
}

// TestTheReleaseIsAskedOfGithubWhenNothingElseIsSet. updateEndpoint is a
// test's way in and is empty everywhere else, so the address a reader's orbit
// actually asks is only ever exercised here. The context is cancelled so that
// asking it costs nothing.
func TestTheReleaseIsAskedOfGithubWhenNothingElseIsSet(t *testing.T) {
	old := updateEndpoint
	updateEndpoint = ""

	t.Cleanup(func() { updateEndpoint = old })

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := fetchRelease(ctx)
	if err == nil {
		t.Fatal("a cancelled fetch answered a release")
	}

	if !strings.Contains(err.Error(), "api.github.com/repos/"+defaultRepo) {
		t.Errorf("the release was not asked of github: %v", err)
	}
}

// TestAnUpgradeThatCouldNotAskSaysSo. There is nothing to fall back to here:
// without a release there is no version to install, so the reader gets the
// reason and their old orbit.
func TestAnUpgradeThatCouldNotAskSaysSo(t *testing.T) {
	oldEndpoint, oldVersion := updateEndpoint, Version
	updateEndpoint, Version = dead(t), "1.0.0"

	t.Cleanup(func() { updateEndpoint, Version = oldEndpoint, oldVersion })

	t.Setenv("ORBIT_HOME", t.TempDir())

	code, out, errOut := run(t, "upgrade")
	if code == 0 {
		t.Fatalf("an upgrade that never heard back exited 0:\n%s\n%s", out, errOut)
	}

	if !strings.Contains(errOut, "fetch update") {
		t.Errorf("the reader is not told the release could not be fetched:\n%s", errOut)
	}
}

// TestTheFallbackThatWorkedIsStillReported. `go install` writes into GOBIN and
// leaves this binary where it is, so the reader is told which version they
// were moved to and why the archive was not the way it happened.
func TestTheFallbackThatWorkedIsStillReported(t *testing.T) {
	missing := serving(t, func(w http.ResponseWriter) {
		http.Error(w, "no such release asset", http.StatusNotFound)
	})

	api := serving(t, func(w http.ResponseWriter) {
		rel := releaseInfo{
			TagName: "v2.0.0",
			Assets: []asset{
				{Name: fmt.Sprintf("orbit_2.0.0_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH), URL: missing},
				{Name: "checksums.txt", URL: missing},
			},
		}
		if err := json.NewEncoder(w).Encode(rel); err != nil {
			t.Errorf("encode release: %v", err)
		}
	})

	oldEndpoint, oldVersion := updateEndpoint, Version
	updateEndpoint, Version = api, "1.0.0"

	t.Cleanup(func() { updateEndpoint, Version = oldEndpoint, oldVersion })

	t.Setenv("ORBIT_HOME", t.TempDir())
	t.Setenv("PATH", goThatSaysYes(t))

	code, out, errOut := run(t, "upgrade")
	if code != 0 {
		t.Fatalf("an upgrade whose fallback worked exited %d:\n%s\n%s", code, out, errOut)
	}

	// The last line is the one that says it worked. The line above it says
	// which version the upgrade is going to, so looking anywhere in the
	// output would pass even if the success named the version they left.
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if !strings.Contains(lines[len(lines)-1], "v2.0.0") {
		t.Errorf("the version the reader was moved to is not what the success names:\n%s", out)
	}

	if !strings.Contains(errOut, "404") {
		t.Errorf("why the release archive was not used is not reported:\n%s", errOut)
	}
}

// goThatSaysYes is a directory holding a `go` that builds nothing and is
// happy about it, so that the fallback can be taken in a test without
// fetching orbit off the network and writing it into the reader's GOBIN.
func goThatSaysYes(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write the go this test stands in for: %v", err)
	}

	return dir
}
