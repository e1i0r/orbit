//go:build integration

package integration

// Every flow Orbit ships, walked at least once.
//
// Four of the five were, and the fifth was not, and nothing said so. A flow
// is the one thing here a reader can run without writing any Go at all — it
// is a file in the binary — so a flow nobody ever walks is a feature that
// ships broken and is found by whoever tries it first.
//
// This is the ratchet. It reads the flows out of the binary's own directory
// rather than a list written here, so a sixth flow added tomorrow fails this
// until somebody walks it too.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEveryShippedFlowIsWalked.
func TestEveryShippedFlowIsWalked(t *testing.T) {
	suite := readSuite(t)

	for _, name := range shippedFlows(t) {
		if !strings.Contains(suite, `"-flow", "`+name+`"`) {
			t.Errorf("the %q flow ships and no integration test walks it", name)
		}
	}
}

// shippedFlows is every flow inside the binary, by name.
func shippedFlows(t *testing.T) []string {
	t.Helper()

	dir := filepath.Join(repoRoot(), "internal", "flow", "flows")

	found, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read the flows Orbit ships: %v", err)
	}

	var out []string

	for _, one := range found {
		if name := strings.TrimSuffix(one.Name(), ".json"); name != one.Name() {
			out = append(out, name)
		}
	}

	if len(out) == 0 {
		t.Fatal("no flows were found, so this test would pass for the wrong reason")
	}

	return out
}

// readSuite is every integration test, joined, which is where a flow is
// named if it is walked at all.
func readSuite(t *testing.T) string {
	t.Helper()

	here, err := os.Getwd()
	if err != nil {
		t.Fatalf("find this package: %v", err)
	}

	found, err := os.ReadDir(here)
	if err != nil {
		t.Fatalf("read this package: %v", err)
	}

	var b strings.Builder

	for _, one := range found {
		if !strings.HasSuffix(one.Name(), "_test.go") {
			continue
		}

		body, err := os.ReadFile(filepath.Join(here, one.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", one.Name(), err)
		}

		b.Write(body)
	}

	return b.String()
}
