package quota

// Codex writes its own rate limits down after every turn. This reads them
// where it left them.
//
// There is no proxy in front of a codex signed in with a ChatGPT account and
// no verb that prints what is left — /usage is a screen inside its own
// window. What there is: every rollout file codex keeps carries a
// token_count event per turn, and each of those carries the rate limits the
// API answered with. The last one written is the last thing codex was told,
// which is the same number its own screen draws.
//
// It is a reading and not a subscription: the file is written when codex
// runs, so what this reports is as fresh as the last run. Orbit runs codex
// itself, so in the case that matters — a phase that just finished — the
// file was written seconds ago. A window whose reset has since passed is
// dropped rather than shown, because a share of a window that has come back
// is a number about a window that no longer exists.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	// codexTail is how much of a rollout's end is read looking for the last
	// token_count line. The event is written after every turn, so it is
	// near the end of the file; a whole rollout is the session's entire
	// transcript and reading one every half minute to find its last line
	// would be megabytes for two percentages.
	codexTail = 1 << 20

	// codexFiles is how many rollouts are opened before giving up. A run
	// that was interrupted before its first turn leaves a file with no
	// token_count in it at all, and that file is the newest one.
	codexFiles = 5

	// codexDated is how deep the year/month/day directories go under
	// sessions/.
	codexDated = 3
)

// rollouts is codex's session directory, read as a quota source.
type rollouts struct{ dir string }

// newRollouts is a Source over that directory, or nothing when there is no
// directory to read — a machine where codex has never run has nowhere to
// look, which is an answer this package states rather than an error.
func newRollouts(dir string) Source {
	if strings.TrimSpace(dir) == "" {
		return nil
	}

	return rollouts{dir: dir}
}

// codexSessions is where codex keeps its rollouts on this machine. CODEX_HOME
// is codex's own variable for moving them.
func codexSessions() string {
	if home := strings.TrimSpace(os.Getenv("CODEX_HOME")); home != "" {
		return filepath.Join(home, "sessions")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".codex", "sessions")
}

// Read answers with the windows of the last turn codex was told about.
func (r rollouts) Read() ([]Window, error) {
	for _, path := range newestRollouts(r.dir, codexDated, codexFiles) {
		line, err := lastTokenCount(path)
		if err != nil {
			return nil, err
		}

		if line == nil {
			continue
		}

		if windows := line.windows(time.Now()); len(windows) > 0 {
			return windows, nil
		}
	}

	return nil, nil
}

// newestRollouts is the newest rollout files under a tree of dated
// directories, newest first.
//
// The tree is walked rather than listed: sessions/ holds every session codex
// has ever run, and sorting all of them by modification time to take the
// first five would stat the lot. The directories are named by date and the
// files by timestamp, so reading each level backwards arrives at the newest
// file having opened one directory per level.
func newestRollouts(dir string, depth, want int) []string {
	if want <= 0 {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	slices.Reverse(entries)

	var out []string

	for _, e := range entries {
		switch {
		case depth > 0 && e.IsDir():
			out = append(out, newestRollouts(filepath.Join(dir, e.Name()), depth-1, want-len(out))...)
		case depth == 0 && !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl"):
			out = append(out, filepath.Join(dir, e.Name()))
		}

		if len(out) >= want {
			break
		}
	}

	return out
}

// codexLine is one rollout line, of which this reads two fields: what the
// event is and what the API said about the limits when it happened.
type codexLine struct {
	At      time.Time `json:"timestamp"`
	Payload struct {
		Type   string       `json:"type"`
		Limits *codexLimits `json:"rate_limits"`
	} `json:"payload"`
}

// codexLimits is the pair of windows a plan has. secondary is null on a plan
// with only one.
type codexLimits struct {
	Primary   *codexWindow `json:"primary"`
	Secondary *codexWindow `json:"secondary"`
}

// codexWindow is one of them.
//
// Two spellings of the countdown are in the files on this machine: older
// codex wrote the seconds left when the line was written, newer codex writes
// the instant it comes back. Both are read, and the relative one is measured
// from the line's own timestamp — seconds left, five months ago, is not
// seconds left.
type codexWindow struct {
	Pct      float64 `json:"used_percent"`
	Minutes  int     `json:"window_minutes"`
	ResetsAt int64   `json:"resets_at"`
	ResetsIn int64   `json:"resets_in_seconds"`
}

// windows is what one token_count line says about the plan, in the shape
// every other source here answers in.
func (l codexLine) windows(now time.Time) []Window {
	if l.Payload.Limits == nil {
		return nil
	}

	var out []Window

	for _, w := range []*codexWindow{l.Payload.Limits.Primary, l.Payload.Limits.Secondary} {
		if win, ok := w.window(l.At, now); ok {
			out = append(out, win)
		}
	}

	return out
}

// window is one limit as a Window, and whether there is one to report.
func (w *codexWindow) window(at, now time.Time) (Window, bool) {
	if w == nil || w.Minutes <= 0 {
		return Window{}, false
	}

	resets := w.resets(at)
	if !resets.IsZero() && !resets.After(now) {
		// The window came back after codex wrote this down, so what it
		// says was used has since been given back.
		return Window{}, false
	}

	out := Window{Label: codexLabel(w.Minutes), Pct: w.Pct}
	if !resets.IsZero() {
		out.ResetsIn = resets.Sub(now)
	}

	return out, true
}

// resets is when this window comes back, from whichever of the two spellings
// the file carries.
func (w *codexWindow) resets(at time.Time) time.Time {
	if w.ResetsAt > 0 {
		return time.Unix(w.ResetsAt, 0)
	}

	if w.ResetsIn > 0 && !at.IsZero() {
		return at.Add(time.Duration(w.ResetsIn) * time.Second)
	}

	return time.Time{}
}

// codexLabel names a window by how long it is, in the unit it is nearest.
//
// Codex counts a week as 10079 minutes and a month as 43200, so the unit is
// rounded to rather than divided into: a label of "6d" on a seven-day window
// would be a rounding error drawn as a fact about the plan.
func codexLabel(minutes int) string {
	switch {
	case minutes >= 60*24:
		return fmt.Sprintf("%dd", (minutes+60*12)/(60*24))
	case minutes >= 60:
		return fmt.Sprintf("%dh", (minutes+30)/60)
	default:
		return fmt.Sprintf("%dm", minutes)
	}
}

// lastTokenCount is the last token_count line of a rollout, or nothing when
// it has none.
func lastTokenCount(path string) (*codexLine, error) {
	data, err := readTail(path, codexTail)
	if err != nil {
		return nil, err
	}

	lines := bytes.Split(data, []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		if !bytes.Contains(lines[i], []byte(`"token_count"`)) {
			continue
		}

		var line codexLine
		if json.Unmarshal(lines[i], &line) != nil {
			// The first line of a tail is half a line, and a rollout
			// written by a codex this does not know is not an error worth
			// stopping a status bar for.
			continue
		}

		if line.Payload.Type == "token_count" && line.Payload.Limits != nil {
			return &line, nil
		}
	}

	return nil, nil
}

// readTail is the last n bytes of a file, which is where the last line of it
// is.
func readTail(path string, n int64) (data []byte, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening the codex rollout %s: %w", path, err)
	}

	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing the codex rollout %s: %w", path, cerr)
		}
	}()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("measuring the codex rollout %s: %w", path, err)
	}

	if start := info.Size() - n; start > 0 {
		if _, err := f.Seek(start, io.SeekStart); err != nil {
			return nil, fmt.Errorf("seeking the codex rollout %s: %w", path, err)
		}
	}

	data, err = io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading the codex rollout %s: %w", path, err)
	}

	return data, nil
}
