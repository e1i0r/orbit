package quota

// Reading codex's windows out of the rollouts codex writes them into.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// codexRollout writes one rollout file where codex would keep it, and
// answers with the CODEX_HOME it was written under.
func codexRollout(t *testing.T, day string, name string, lines ...string) string {
	t.Helper()

	home := t.TempDir()
	writeRollout(t, home, day, name, lines...)
	t.Setenv("CODEX_HOME", home)

	return home
}

func writeRollout(t *testing.T, home, day, name string, lines ...string) {
	t.Helper()

	dir := filepath.Join(home, "sessions", filepath.FromSlash(day))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("making the sessions directory: %v", err)
	}

	body := ""
	for _, line := range lines {
		body += line + "\n"
	}

	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatalf("writing the rollout: %v", err)
	}
}

// tokenCount is a token_count line as codex 0.153 writes one, with the reset
// as an instant.
func tokenCount(at time.Time, primaryPct float64, primaryMinutes int, resetsAt time.Time) string {
	return fmt.Sprintf(`{"timestamp":%q,"type":"event_msg","payload":{"type":"token_count",`+
		`"info":{"total_token_usage":{"total_tokens":150942}},`+
		`"rate_limits":{"limit_id":"codex","primary":{"used_percent":%v,"window_minutes":%d,"resets_at":%d},`+
		`"secondary":null,"plan_type":"free"}}}`,
		at.Format(time.RFC3339), primaryPct, primaryMinutes, resetsAt.Unix())
}

// TestCodexWindowsComeOffItsOwnRollouts. There is no proxy in front of a
// codex signed in with a ChatGPT account and no verb that prints what is
// left, and this is the same number its own /usage screen draws.
func TestCodexWindowsComeOffItsOwnRollouts(t *testing.T) {
	now := time.Now()
	codexRollout(t, "2026/09/03", "rollout-2026-09-03T20-08-38-aaa.jsonl",
		`{"timestamp":"2026-09-03T20:08:38Z","type":"session_meta","payload":{"id":"aaa"}}`,
		tokenCount(now.Add(-time.Minute), 33, 43200, now.Add(571*time.Hour)))

	windows, err := newRollouts(codexSessions()).Read()
	if err != nil {
		t.Fatalf("reading codex's rollouts: %v", err)
	}

	if len(windows) != 1 {
		t.Fatalf("codex answered %+v, want the one window its plan has", windows)
	}

	if windows[0].Label != "30d" || windows[0].Pct != 33 {
		t.Errorf("the window is %+v, want 33%% of a thirty-day window", windows[0])
	}

	if windows[0].ResetsIn < 570*time.Hour || windows[0].ResetsIn > 571*time.Hour {
		t.Errorf("the window comes back in %v, want the countdown to the instant codex wrote down", windows[0].ResetsIn)
	}
}

// TestTheLastThingCodexWasToldIsTheReading. A file holds a token_count per
// turn, and the percentage from the first turn of a session is the one that
// is out of date.
func TestTheLastThingCodexWasToldIsTheReading(t *testing.T) {
	now := time.Now()
	codexRollout(t, "2026/09/03", "rollout-2026-09-03T20-08-38-aaa.jsonl",
		tokenCount(now.Add(-time.Hour), 12, 43200, now.Add(500*time.Hour)),
		tokenCount(now.Add(-time.Minute), 33, 43200, now.Add(500*time.Hour)))

	windows, err := newRollouts(codexSessions()).Read()
	if err != nil {
		t.Fatalf("reading codex's rollouts: %v", err)
	}

	if len(windows) != 1 || windows[0].Pct != 33 {
		t.Errorf("codex answered %+v, want the last turn's share and not the first's", windows)
	}
}

// TestTheNewestRolloutIsTheOneRead, across the dated directories codex files
// them under.
func TestTheNewestRolloutIsTheOneRead(t *testing.T) {
	now := time.Now()
	home := codexRollout(t, "2026/09/03", "rollout-2026-09-03T20-08-38-aaa.jsonl",
		tokenCount(now.Add(-time.Minute), 33, 43200, now.Add(500*time.Hour)))
	writeRollout(t, home, "2026/08/29", "rollout-2026-08-29T22-14-09-bbb.jsonl",
		tokenCount(now.Add(-120*time.Hour), 9, 43200, now.Add(500*time.Hour)))

	windows, err := newRollouts(codexSessions()).Read()
	if err != nil {
		t.Fatalf("reading codex's rollouts: %v", err)
	}

	if len(windows) != 1 || windows[0].Pct != 33 {
		t.Errorf("codex answered %+v, want the newest session's reading", windows)
	}
}

// TestARolloutWithNothingToSayIsNotTheAnswer: a session cancelled before its
// first turn leaves a file with no token_count in it, and that file is the
// newest one on the machine.
func TestARolloutWithNothingToSayIsNotTheAnswer(t *testing.T) {
	now := time.Now()
	home := codexRollout(t, "2026/09/03", "rollout-2026-09-03T21-00-00-ccc.jsonl",
		`{"timestamp":"2026-09-03T21:00:00Z","type":"session_meta","payload":{"id":"ccc"}}`)
	writeRollout(t, home, "2026/09/03", "rollout-2026-09-03T20-08-38-aaa.jsonl",
		tokenCount(now.Add(-time.Hour), 33, 43200, now.Add(500*time.Hour)))

	windows, err := newRollouts(codexSessions()).Read()
	if err != nil {
		t.Fatalf("reading codex's rollouts: %v", err)
	}

	if len(windows) != 1 || windows[0].Pct != 33 {
		t.Errorf("codex answered %+v, want the newest rollout that had a reading in it", windows)
	}
}

// TestAWindowThatHasComeBackIsNotReported. The file is written when codex
// runs; a share of a window that has since reset is a number about a window
// that no longer exists.
func TestAWindowThatHasComeBackIsNotReported(t *testing.T) {
	now := time.Now()
	codexRollout(t, "2026/09/03", "rollout-2026-09-03T20-08-38-aaa.jsonl",
		tokenCount(now.Add(-72*time.Hour), 88, 300, now.Add(-48*time.Hour)))

	windows, err := newRollouts(codexSessions()).Read()
	if err != nil {
		t.Fatalf("reading codex's rollouts: %v", err)
	}

	if len(windows) != 0 {
		t.Errorf("codex answered %+v, want nothing about a window that has already come back", windows)
	}
}

// TestTheOlderSpellingOfTheCountdown: codex used to write the seconds left
// when the line was written rather than the instant it comes back, and both
// spellings are in the files on a machine that has had it a while.
func TestTheOlderSpellingOfTheCountdown(t *testing.T) {
	at := time.Now().Add(-time.Hour)
	codexRollout(t, "2025/10/14", "rollout-2025-10-14T06-21-26-ddd.jsonl",
		fmt.Sprintf(`{"timestamp":%q,"type":"event_msg","payload":{"type":"token_count","info":{},`+
			`"rate_limits":{"primary":{"used_percent":3.0,"window_minutes":299,"resets_in_seconds":17566},`+
			`"secondary":{"used_percent":1.0,"window_minutes":10079,"resets_in_seconds":604366}}}}`,
			at.Format(time.RFC3339)))

	windows, err := newRollouts(codexSessions()).Read()
	if err != nil {
		t.Fatalf("reading codex's rollouts: %v", err)
	}

	if len(windows) != 2 {
		t.Fatalf("codex answered %+v, want both windows the plan has", windows)
	}

	if windows[0].Label != "5h" || windows[1].Label != "7d" {
		t.Errorf("the windows are labelled %q and %q, want 5h and 7d", windows[0].Label, windows[1].Label)
	}

	// An hour of the five-hour window's countdown has gone by since the line
	// was written, and the seconds on it were counted from then.
	if want := 17566*time.Second - time.Hour; windows[0].ResetsIn > want+time.Minute || windows[0].ResetsIn < want-time.Minute {
		t.Errorf("the window comes back in %v, want about %v — counted from the line, not from now", windows[0].ResetsIn, want)
	}
}

// TestAMachineWhereCodexHasNeverRunIsNotAFailure.
func TestAMachineWhereCodexHasNeverRunIsNotAFailure(t *testing.T) {
	t.Setenv("CODEX_HOME", t.TempDir())

	windows, err := newRollouts(codexSessions()).Read()
	if err != nil {
		t.Fatalf("asking a machine with no codex: %v", err)
	}

	if len(windows) != 0 {
		t.Errorf("a machine with no codex answered %+v", windows)
	}

	if newRollouts("") != nil {
		t.Error("a source was built over nowhere to look")
	}
}

// TestAWindowIsNamedByHowLongItIs, in the unit it is nearest: codex counts a
// week as 10079 minutes and a five-hour window as 299, so a label that
// divided rather than rounded would report a six-day week.
func TestAWindowIsNamedByHowLongItIs(t *testing.T) {
	for minutes, want := range map[int]string{43200: "30d", 10079: "7d", 299: "5h", 60: "1h", 45: "45m"} {
		if got := codexLabel(minutes); got != want {
			t.Errorf("codexLabel(%d) = %q, want %q", minutes, got, want)
		}
	}
}

// TestCodexIsSourcedByItsOwnFiles: with no proxy named, the meter reads what
// codex wrote down rather than reporting an engine nobody can see.
func TestCodexIsSourcedByItsOwnFiles(t *testing.T) {
	bare(t)

	now := time.Now()
	codexRollout(t, "2026/09/03", "rollout-2026-09-03T20-08-38-aaa.jsonl",
		tokenCount(now.Add(-time.Minute), 33, 43200, now.Add(500*time.Hour)))

	reading := FromEnv().Read("codex", true)
	if !reading.Sourced {
		t.Fatal("codex reads as an engine with nowhere to look, and its own rollouts are somewhere")
	}

	if len(reading.Windows) != 1 || reading.Windows[0].Pct != 33 {
		t.Errorf("codex reads as %+v, want the window off its rollout", reading)
	}
}

// TestAProxyThatCannotAnswerForCodexFallsThroughToTheFile. A base URL is
// often one proxy in front of every engine, and a proxy that speaks for
// another engine has no route for this one — the reader is owed the number
// that is on their own disk rather than a 404 drawn as an unreadable engine.
func TestAProxyThatCannotAnswerForCodexFallsThroughToTheFile(t *testing.T) {
	bare(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "no such route", http.StatusNotFound)
	}))
	defer srv.Close()

	t.Setenv("OPENAI_BASE_URL", srv.URL)

	now := time.Now()
	codexRollout(t, "2026/09/03", "rollout-2026-09-03T20-08-38-aaa.jsonl",
		tokenCount(now.Add(-time.Minute), 33, 43200, now.Add(500*time.Hour)))

	reading := FromEnv().Read("codex", true)
	if len(reading.Windows) != 1 || reading.Windows[0].Pct != 33 {
		t.Errorf("codex reads as %+v, want the window off its rollout once the proxy had nothing", reading)
	}
}

// TestAProxyThatDoesAnswerIsThePreferredReading, because it is live and the
// file is as fresh as the last run.
func TestAProxyThatDoesAnswerIsThePreferredReading(t *testing.T) {
	bare(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{ //nolint:errcheck // the test reads what arrives
			{"key": "acct", "label": "5h", "pct": 40, "resets_in_s": 900},
		})
	}))
	defer srv.Close()

	t.Setenv("OPENAI_BASE_URL", srv.URL)

	now := time.Now()
	codexRollout(t, "2026/09/03", "rollout-2026-09-03T20-08-38-aaa.jsonl",
		tokenCount(now.Add(-time.Minute), 33, 43200, now.Add(500*time.Hour)))

	reading := FromEnv().Read("codex", true)
	if len(reading.Windows) != 1 || reading.Windows[0].Pct != 40 {
		t.Errorf("codex reads as %+v, want the proxy's live window and not the file's", reading)
	}
}
