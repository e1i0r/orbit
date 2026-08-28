package ui

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
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
	if !comparableVersion(current) {
		return nil
	}

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

// comparableVersion is whether the running build has a version a release
// tag could be compared against.
//
// It is asked before the request goes out, not after the answer comes back.
// A build from source calls itself dev and is never offered an upgrade, and
// it used to reach GitHub every hour for an answer worthOffering then threw
// away — one request an hour, for the whole time the window is open, to
// learn something already known before it was sent.
func comparableVersion(current string) bool {
	cur := strings.TrimPrefix(strings.TrimSpace(current), "v")
	return cur != "" && cur != "dev"
}

// worthOffering is whether a released tag is worth putting on the header,
// given what is running.
//
// Nothing to compare against — an empty version, or a build that calls
// itself dev — is not an upgrade anybody can act on, so it is left alone.
// The leading v is stripped from both sides because a tag carries one and a
// version string may not, and "v0.1.12" and "0.1.12" are the same release.
//
// What is offered is a release *after* this one. This was "different from
// this one", which is the same answer for both directions: a build ahead of
// the last published tag — the one anybody working on orbit is running the
// day after a release, and anybody on a release candidate — was told a
// lower version was available, and pressing the banner would have taken
// them backwards.
func worthOffering(current, latest string) bool {
	if !comparableVersion(current) {
		return false
	}

	return laterThan(strings.TrimSpace(latest), strings.TrimSpace(current))
}

// laterThan is whether version a names a release later than b.
//
// A version this cannot read is not offered. Saying nothing about a pair
// that cannot be ordered is the same choice made for a dev build, and for
// the same reason: the cost of a wrong yes is a reader sent backwards, and
// the cost of a wrong no is a banner that does not appear.
func laterThan(a, b string) bool {
	an, apre, aok := versionParts(a)

	bn, bpre, bok := versionParts(b)
	if !aok || !bok {
		return false
	}

	for i := range max(len(an), len(bn)) {
		x, y := numberAt(an, i), numberAt(bn, i)
		if x != y {
			return x > y
		}
	}

	// Same numbers: 1.2.0 is the release the candidates 1.2.0-rc1 were
	// leading to, so it is later than any of them, and no candidate is
	// ever offered over the release itself.
	return apre == "" && bpre != ""
}

// numberAt is the i-th number of a version, counting an absent one as zero, so
// that 1.2 and 1.2.0 are the same release.
func numberAt(nums []int, i int) int {
	if i < len(nums) {
		return nums[i]
	}

	return 0
}

// versionParts splits a version into its numbers and whatever pre-release
// suffix follows them. It reports false for anything it cannot read as a
// dotted run of numbers.
func versionParts(v string) (nums []int, pre string, ok bool) {
	v = strings.TrimPrefix(v, "v")

	// Cut rather than slice: internal/arch forbids slicing a string in this
	// package, and the reason holds here — the separator is found by index
	// and the index is a byte offset.
	for _, sep := range []string{"-", "+"} {
		if head, tail, found := strings.Cut(v, sep); found {
			v, pre = head, sep+tail
			break
		}
	}

	if v == "" {
		return nil, "", false
	}

	for _, part := range strings.Split(v, ".") {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return nil, "", false
		}

		nums = append(nums, n)
	}

	return nums, pre, true
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
