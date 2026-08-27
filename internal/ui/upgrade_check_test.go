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

	msg := checkUpgradeCmd()
	upMsg, ok := msg.(upgradeAvailableMsg)
	if !ok {
		t.Fatalf("expected upgradeAvailableMsg, got: %T", msg)
	}
	if upMsg.Version != "v0.2.0" {
		t.Errorf("Version = %q, want v0.2.0", upMsg.Version)
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

	msg := checkUpgradeCmd()
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
