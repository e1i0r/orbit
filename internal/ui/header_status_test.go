package ui

// header_status_coverage_test.go covers the header line's own shrinking —
// shorten had never run at all — and the status line's five segments,
// including the quota window and the reset-time formatter that draws it.

import (
	"strings"
	"testing"
	"time"
)

func TestShorten(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"~/work/acme/payments", "…/work/acme/payments"},
		{"…/work/acme/payments", "…/acme/payments"},
		{"…/acme/payments", "…/payments"},
		{"…/payments", ""},
		{"payments", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := shorten(tt.in); got != tt.want {
			t.Errorf("shorten(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRule(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	if got := m.rule(0); got != "" {
		t.Errorf("rule(0) = %q, want empty", got)
	}

	if got := m.rule(-3); got != "" {
		t.Errorf("rule(-3) = %q, want empty", got)
	}

	if got := m.rule(10); got == "" {
		t.Error("rule(10) = empty, want a line")
	}
}

// TestHeaderLineShrinksAndRefuses walks the header from a width wide enough
// for every field down to one so narrow even the program's own name and a
// bare root will not fit, which is headerLine's last resort: fit(m.name(), w).
func TestHeaderLineShrinksAndRefuses(t *testing.T) {
	m, _ := testModel(t, 200, 30)
	m.opts.Root = "~/work/acme/payments/service"

	wide := m.headerLine(200)
	if wide == "" {
		t.Fatal("headerLine(200) is empty")
	}

	narrow := m.headerLine(30)
	if narrow == "" {
		t.Fatal("headerLine(30) is empty")
	}

	tiny := m.headerLine(6)
	if tiny == "" {
		t.Fatal("headerLine(6) is empty, want the bare program name")
	}
}

// TestHeaderLeftQueueBadges is the branch that only fires with four bands
// counted and enough room: the full "[orbit]  todo running needs-you done"
// line rather than the plain root path.
func TestHeaderLeftQueueBadges(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	if len(m.board.Counts) < 4 {
		t.Fatalf("fixture board has %d counts, want at least 4", len(m.board.Counts))
	}

	line, _, ok := m.headerLeft(120, false)
	if !ok || !strings.Contains(line, "📋") {
		t.Errorf("headerLeft(120, false) = %q, ok=%v, want the queue badges", line, ok)
	}
	// A width too small for the badges falls back to the root path instead.
	line, _, ok = m.headerLeft(40, true)
	if !ok || strings.Contains(line, "📋") {
		t.Errorf("headerLeft(40, true) = %q, ok=%v, want the root path fallback", line, ok)
	}
	// And a width too small for even that refuses outright.
	_, _, ok = m.headerLeft(1, true)
	if ok {
		t.Error("headerLeft(1, true) succeeded, want a refusal")
	}
}

func TestHeaderFieldsUnreadBrake(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// 1. Below the cap (the fixture's cap of 5 against 3 unread finished
	// tasks): no brake field.
	fields := m.headerFields()
	if strings.Contains(strings.Join(fieldTexts(fields), " "), "brake") {
		t.Errorf("headerFields below the cap mentioned the brake: %v", fields)
	}

	// 2. At the cap: the settings port is swapped for one whose cap the
	// fixture's unread count has already reached — SetUnreadCap on the
	// fixture stub does not persist, so the cap can only move by replacing
	// the port outright.
	m.opts.Settings = &settings{autopilot: true, lang: "en", unread: 1}
	if fields := m.headerFields(); !strings.Contains(strings.Join(fieldTexts(fields), " "), "brake") {
		t.Errorf("headerFields at the cap = %v, want the brake field", fields)
	}

	// 3. The knob chip: the default claude chip with no knobs set, and the
	// chosen engine/model once they are.
	m, _ = testModel(t, 100, 30)
	if fields := m.headerFields(); !strings.Contains(strings.Join(fieldTexts(fields), " "), "claude") {
		t.Errorf("headerFields with no knobs set = %v, want the default claude chip", fields)
	}

	m.knobs.Engine, m.knobs.Model = "codex", "o1"
	if fields := m.headerFields(); !strings.Contains(strings.Join(fieldTexts(fields), " "), "o1") {
		t.Errorf("headerFields with knobs set = %v, want the chip to mention o1", fields)
	}
}

func TestFmtReset(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{-5 * time.Second, "0s"},
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{2*time.Hour + 15*time.Minute, "2h15m"},
	}
	for _, tt := range tests {
		if got := fmtReset(tt.d); got != tt.want {
			t.Errorf("fmtReset(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

// TestStatusLineSegmentsAndQuota walks statusLine from wide enough for every
// segment down to one segment, and through the quota-window branch, which
// only shows when Options.Quota is set.
func TestStatusLineSegmentsAndQuota(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Quota = func() []QuotaWindow {
		return []QuotaWindow{{Label: "5h", Pct: 40, ResetsIn: 90 * time.Minute}}
	}

	wide := m.statusLine(200)
	if !strings.Contains(wide, "%") {
		t.Errorf("statusLine(200) = %q, want the quota segment", wide)
	}

	narrow := m.statusLine(10)
	if narrow == "" {
		t.Error("statusLine(10) is empty, want at least one segment fitted")
	}

	// A read time over the 100ms bound is painted Bad rather than Dim; both
	// are reachable through the same field, so both are worth exercising.
	m.board.Health.Duration = 250 * time.Millisecond
	if got := m.statusLine(200); got == "" {
		t.Error("statusLine with a slow read is empty")
	}

	// No quota port at all: the segment is simply absent.
	m.opts.Quota = nil
	if got := m.statusLine(200); strings.Contains(got, "%") {
		t.Errorf("statusLine with no quota port = %q, want no percentage", got)
	}

	// An empty quota window slice: same as no port, since the branch checks
	// len(windows) > 0.
	m.opts.Quota = func() []QuotaWindow { return nil }
	if got := m.statusLine(200); strings.Contains(got, "%") {
		t.Errorf("statusLine with an empty quota slice = %q, want no percentage", got)
	}
}

func TestStatusRows(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	if rows := m.statusRows(); len(rows) == 0 {
		t.Error("statusRows on a sized frame returned nothing")
	}

	m.frame.Status.H = 0
	if rows := m.statusRows(); rows != nil {
		t.Errorf("statusRows with H=0 = %v, want nil", rows)
	}
}
