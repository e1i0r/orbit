package ui

import (
	"context"
	"encoding/json"
	"net/http"
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

// checkUpgradeCmd performs a lightweight, non-blocking check for the latest release.
func checkUpgradeCmd() tea.Msg {
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

	return upgradeAvailableMsg{Version: rel.TagName}
}
