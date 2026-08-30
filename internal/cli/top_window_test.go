package cli

// window()'s own early exits, quotaPort's window-building loop (never
// reached in ports_settings_test.go because quota.FromEnv()
// there is nil or empty), and interactive()'s character-device check, which
// no other test reaches with os.Stdout itself rather than a buffer.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/e1i0r/orbit/internal/quota"
)

func TestWindowRefusesADirectoryThatIsNotOne(t *testing.T) {
	t.Setenv("ORBIT_HOME", t.TempDir())

	if _, _, err := window(Context{}, filepath.Join(t.TempDir(), "does-not-exist"), ""); err == nil {
		t.Error("window over a directory that does not exist succeeded")
	}
}

func TestWindowFailsWhenTheStateRootCannotBeCreated(t *testing.T) {
	repoRoot := t.TempDir()

	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("ORBIT_HOME", filepath.Join(blocker, "orbit"))

	if _, _, err := window(Context{}, repoRoot, ""); err == nil {
		t.Error("window with an unmakeable state root succeeded")
	}
}

// window() falls back to $HOME when os.UserHomeDir() cannot answer — the
// standard library's own version of that call reads $HOME on this platform,
// so unsetting it makes both the primary lookup and the fallback answer the
// same empty string, but it is what exercises the branch rather than
// short-circuiting past it.
func TestWindowFallsBackToHOMEWhenUserHomeDirFails(t *testing.T) {
	repoRoot := t.TempDir()
	t.Setenv("ORBIT_HOME", t.TempDir())
	t.Setenv("HOME", "")

	if _, _, err := window(Context{}, repoRoot, ""); err != nil {
		t.Fatalf("window: %v", err)
	}
}

func TestQuotaPortMapsRealWindows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{ //nolint:errcheck
			{"key": "acct-1", "label": "5h", "pct": 42.5, "resets_in_s": 3600},
		})
	}))
	defer srv.Close()

	qc := quota.New(srv.URL)

	qp := quotaPort(qc, true)
	if qp == nil {
		t.Fatal("quotaPort(client, true) returned nil")
	}

	windows := qp()
	if len(windows) != 1 {
		t.Fatalf("got %d windows, want 1", len(windows))
	}

	if windows[0].Key != "acct-1" || windows[0].Label != "5h" {
		t.Errorf("unexpected window: %+v", windows[0])
	}
}

func TestInteractiveOnRealStdout(t *testing.T) {
	// Whatever the answer is under a test binary's stdout (rarely a
	// terminal), the function must not panic and must exercise the
	// character-device Stat check rather than short-circuiting on the
	// "not os.Stdout" comparison every other test in this package takes.
	_ = interactive(os.Stdout)
}
