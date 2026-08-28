package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckUpgradeCmd(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := latestRelease{TagName: "v0.2.0"}
		if err := json.NewEncoder(w).Encode(rel); err != nil {
			t.Errorf("encode release: %v", err)
		}
	}))
	defer ts.Close()

	old := upgradeCheckEndpoint
	upgradeCheckEndpoint = ts.URL

	t.Cleanup(func() { upgradeCheckEndpoint = old })

	msg := checkUpgradeCmd("v0.1.0")()

	upMsg, ok := msg.(upgradeAvailableMsg)
	if !ok {
		t.Fatalf("expected upgradeAvailableMsg, got: %T", msg)
	}

	if upMsg.Version != "v0.2.0" {
		t.Errorf("Version = %q, want v0.2.0", upMsg.Version)
	}
}

// TestCheckUpgradeCmdSaysNothingOnTheLatestVersion is the fix. The banner
// used to light up on whatever tag GitHub answered with, so a build that was
// already current advertised itself: the window said "v0.1.12 available"
// while `orbit upgrade` said it was already on v0.1.12.
func TestCheckUpgradeCmdSaysNothingOnTheLatestVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(latestRelease{TagName: "v0.1.12"}); err != nil {
			t.Errorf("encode release: %v", err)
		}
	}))
	defer ts.Close()

	old := upgradeCheckEndpoint
	upgradeCheckEndpoint = ts.URL

	t.Cleanup(func() { upgradeCheckEndpoint = old })

	// Both spellings, because a tag carries a leading v and the version
	// stamped into a build may not.
	for _, current := range []string{"v0.1.12", "0.1.12"} {
		if msg := checkUpgradeCmd(current)(); msg != nil {
			t.Errorf("running %s, the latest release being v0.1.12: got %+v, want nothing to say", current, msg)
		}
	}
}

// TestWorthOffering is the rule on its own, without a server.
func TestWorthOffering(t *testing.T) {
	for _, c := range []struct {
		current, latest string
		want            bool
		why             string
	}{
		{"v0.1.11", "v0.1.12", true, "a newer release is news"},
		{"0.1.11", "v0.1.12", true, "the leading v is not part of the version"},
		{"v0.1.12", "v0.1.12", false, "the release already running is not news"},
		{"dev", "v0.1.12", false, "a build from source is not told to go backwards"},
		{"", "v0.1.12", false, "a build that cannot say what it is cannot be compared"},
		{"v0.1.12", "", false, "no release to offer"},

		// The direction. This rule was "different from what is running",
		// which answers yes to both directions: the day after a release,
		// everybody working on orbit is ahead of the last published tag
		// and was told to go back to it.
		{"v0.2.0", "v0.1.12", false, "a build ahead of the tag is not told to downgrade"},
		{"v0.10.0", "v0.9.0", false, "ten is after nine, not before it"},
		{"v0.9.0", "v0.10.0", true, "and ten is offered from nine"},
		{"v1.2", "v1.2.0", false, "a missing number is a zero, not a difference"},
		{"v1.2.0-rc1", "v1.2.0", true, "the release a candidate led to is news"},
		{"v1.2.0", "v1.2.0-rc1", false, "a candidate is never offered over the release"},
		{"nightly", "v0.1.12", false, "a version that cannot be ordered is left alone"},
		{"v0.1.12", "latest", false, "and so is a tag that cannot be ordered"},
	} {
		if got := worthOffering(c.current, c.latest); got != c.want {
			t.Errorf("worthOffering(%q, %q) = %v, want %v — %s", c.current, c.latest, got, c.want, c.why)
		}
	}
}

// TestADevBuildNeverReachesGitHub. worthOffering has always left a build
// from source alone, but it was asked after the answer came back: a window
// open all day made one request an hour, waited two seconds for each, and
// discarded every one of them. The question is answerable before the
// request, so it is asked before the request.
func TestADevBuildNeverReachesGitHub(t *testing.T) {
	asked := false

	ts := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		asked = true
	}))
	defer ts.Close()

	old := upgradeCheckEndpoint
	upgradeCheckEndpoint = ts.URL

	t.Cleanup(func() { upgradeCheckEndpoint = old })

	for _, current := range []string{"dev", "", "  "} {
		if cmd := checkUpgradeCmd(current); cmd != nil {
			t.Errorf("a build calling itself %q was given an upgrade check to run", current)
			cmd()
		}
	}

	if asked {
		t.Error("a build with no version to compare asked GitHub for the latest release")
	}
}

func TestCheckUpgradeCmdFailure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	old := upgradeCheckEndpoint
	upgradeCheckEndpoint = ts.URL

	t.Cleanup(func() { upgradeCheckEndpoint = old })

	msg := checkUpgradeCmd("v0.1.0")()
	if msg != nil {
		t.Errorf("expected nil on error, got: %+v", msg)
	}
}

func TestUpgradeTickCmd(t *testing.T) {
	cmd := upgradeTick()
	if cmd == nil {
		t.Fatal("expected non-nil cmd from upgradeTick")
	}
}
