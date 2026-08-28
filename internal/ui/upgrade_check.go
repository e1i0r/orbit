package ui

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

var upgradeCheckEndpoint = "https://api.github.com/repos/e1i0r/orbit/releases/latest"

// upgradeAvailableMsg notifies the TUI that a newer version of orbit was detected.
type upgradeAvailableMsg struct {
	Version string
}

type latestRelease struct {
	TagName string `json:"tag_name"`
}

// checkUpgradeCmd performs a lightweight, non-blocking check for the latest
// release, and says nothing when that release is the one already running.
//
// The comparison is the whole point. This used to announce whatever tag
// GitHub answered with, so the banner lit up on a build that was already the
// latest and stayed lit for ever: the window said "v0.1.12 available" while
// `orbit upgrade` said "orbit is already on the latest version (v0.1.12)".
// Two readers of one fact, disagreeing — which is exactly what the window is
// not allowed to do.
//
// current is the running build's version, handed in through Options because
// internal/ui cannot name internal/cli, where it lives. The rule it applies
// is deliberately the same one cli/upgrade.go applies, down to leaving a dev
// build alone: a build with no version cannot be compared to a tag, and
// telling somebody who built from source to upgrade to the last release is
// usually advice to go backwards.
func checkUpgradeCmd(current string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, upgradeCheckEndpoint, nil)
		if err != nil {
			return nil
		}

		req.Header.Set("User-Agent", "orbit-cockpit")
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}

			return nil
		}
		defer resp.Body.Close()

		var rel latestRelease
		if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil || rel.TagName == "" {
			return nil
		}

		if !worthOffering(current, rel.TagName) {
			return nil
		}

		return upgradeAvailableMsg{Version: rel.TagName}
	}
}

// worthOffering is whether a released tag is worth putting on the header,
// given what is running.
//
// Nothing to compare against — an empty version, or a build that calls
// itself dev — is not an upgrade anybody can act on, so it is left alone.
// The leading v is stripped from both sides because a tag carries one and a
// version string may not, and "v0.1.12" and "0.1.12" are the same release.
func worthOffering(current, latest string) bool {
	cur := strings.TrimPrefix(strings.TrimSpace(current), "v")

	rel := strings.TrimPrefix(strings.TrimSpace(latest), "v")
	if cur == "" || cur == "dev" || rel == "" {
		return false
	}

	return cur != rel
}

// upgradeCheckInterval is the cadence at which Orbit checks for new releases.
const upgradeCheckInterval = 1 * time.Hour

// upgradeTickMsg triggers periodic release checks.
type upgradeTickMsg time.Time

// upgradeTick schedules the next periodic check.
func upgradeTick() tea.Cmd {
	return tea.Tick(upgradeCheckInterval, func(t time.Time) tea.Msg {
		return upgradeTickMsg(t)
	})
}
