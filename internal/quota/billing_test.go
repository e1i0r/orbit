package quota

// What the meter says about each engine: how it is paid for, whether there
// is anywhere to read its window, and the difference between the two ways of
// having nothing to show.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// bare is the environment with none of the variables the meter reads, which
// is a machine that drives its engines through their own endpoints on a
// subscription login. A test that did not do this would read whatever the
// person running it has exported.
func bare(t *testing.T) {
	t.Helper()

	for _, name := range []string{
		"ANTHROPIC_BASE_URL", "ANTHROPIC_API_KEY",
		"OPENAI_BASE_URL", "OPENAI_API_KEY",
	} {
		t.Setenv(name, "")
	}
}

func TestEveryEngineSaysHowItIsPaidFor(t *testing.T) {
	bare(t)

	m := FromEnv()
	for _, name := range []string{"claude", "codex", "opencode"} {
		if m.Mode(name) == Unstated {
			t.Errorf("%s has no billing mode, and every engine orbit runs must state one", name)
		}
	}
}

func TestAKeyInTheEnvironmentIsWhatMakesAnEngineMetered(t *testing.T) {
	bare(t)

	if mode := FromEnv().Mode("claude"); mode != Subscription || mode.Spends() {
		t.Errorf("claude with no key is %v (spends %v), want a subscription that does not", mode, mode.Spends())
	}

	t.Setenv("ANTHROPIC_API_KEY", "sk-test")

	if mode := FromEnv().Mode("claude"); mode != PerToken || !mode.Spends() {
		t.Errorf("claude with a key is %v (spends %v), want per token, spending", mode, mode.Spends())
	}
}

func TestAnEngineWithNoSourceSaysSoRatherThanShowingZero(t *testing.T) {
	bare(t)

	// opencode drives whichever provider it is configured for, so there is
	// no one endpoint that could answer for it.
	none := FromEnv().Read("opencode", false)
	if none.Sourced {
		t.Error("opencode reports a quota source, and it has none")
	}

	if len(none.Windows) != 0 {
		t.Errorf("opencode answered %d windows, want none at all", len(none.Windows))
	}

	// The distinction the field exists for: nothing to read is not a window
	// read as empty, and a reader is owed the difference.
	if none.Mode != PerToken {
		t.Errorf("opencode is %v, want per token", none.Mode)
	}
}

func TestAnEngineNobodyHasWrittenARowForIsUnstated(t *testing.T) {
	bare(t)

	m := FromEnv()
	if mode := m.Mode("gemini"); mode != Unstated {
		t.Errorf("an unknown engine is %v, want unstated", mode)
	}

	got := m.Read("gemini", true)
	if got.Engine != "gemini" || got.Mode != Unstated || got.Sourced || len(got.Windows) != 0 {
		t.Errorf("an unknown engine reads as %+v, want it named and nothing claimed about it", got)
	}
}

func TestANilMeterClaimsNothingAboutAnybody(t *testing.T) {
	var m *Meter

	if mode := m.Mode("claude"); mode != Unstated {
		t.Errorf("a nil meter says claude is %v, want unstated", mode)
	}

	if got := m.Read("claude", true); got.Sourced || got.Mode != Unstated {
		t.Errorf("a nil meter reads claude as %+v, want nothing claimed", got)
	}
}

func TestEachEngineIsReadFromItsOwnSource(t *testing.T) {
	bare(t)

	asked := make(chan string, 4)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked <- r.URL.Path

		_ = json.NewEncoder(w).Encode([]map[string]any{ //nolint:errcheck // the test reads what arrives
			{"key": "acct", "label": "5h", "pct": 40, "resets_in_s": 900},
		})
	}))
	defer srv.Close()

	// The proxy is codex's here and nobody else's, which is the whole point
	// of a source per engine: the same protocol at whichever base URL an
	// engine is driven through.
	t.Setenv("OPENAI_BASE_URL", srv.URL)

	m := FromEnv()

	codex := m.Read("codex", true)
	if !codex.Sourced || len(codex.Windows) != 1 || codex.Windows[0].Label != "5h" {
		t.Fatalf("codex reads as %+v, want the one window its proxy answered", codex)
	}

	if path := <-asked; path != "/quota" {
		t.Errorf("the proxy was asked for %q, want /quota", path)
	}

	if claude := m.Read("claude", true); claude.Sourced {
		t.Error("claude took codex's proxy for its own")
	}
}
