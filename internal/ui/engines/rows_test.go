package engines

// The list drawn: the setup steps, what a disabled engine says, what each
// cell answers to a click, and the quota carried beside an engine's name.

import (
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/ui/point"
	"github.com/e1i0r/orbit/internal/ui/roster"
)

func TestEnginesRowsShowingSetup(t *testing.T) {
	s, e := open(t)
	s.showingSetup, s.setupEngine = true, "codex"

	rows := drawn(s, e)

	joined := strings.Join(rows, "\n")
	if !strings.Contains(joined, "codex") {
		t.Errorf("expected the setup screen to name codex")
	}

	if !strings.Contains(joined, "install codex") {
		t.Errorf("expected codex's own setup steps listed")
	}
}

func TestEnginesRowsSelectedAndDisabled(t *testing.T) {
	_, e := open(t)
	s := Open(0, Knobs{Engine: "claude", Model: "opus"})

	rows := drawn(s, e)

	joined := strings.Join(rows, "\n")
	if !strings.Contains(joined, "setup required") {
		t.Errorf("expected the disabled codex row to say setup is required")
	}
}

func TestHitEnginesOutsideBody(t *testing.T) {
	s, e := open(t)
	if got := s.hit(10, 0, e); got.Kind != point.None {
		t.Errorf("hitEngines outside the body = %+v, want the zero point.Target", got)
	}
}

func TestHitEnginesPastLastRow(t *testing.T) {
	s, e := open(t)
	if got := s.hit(10, e.Frame.Body.Y+999, e); got.Kind != point.None {
		t.Errorf("hitEngines past the last row = %+v, want the zero point.Target", got)
	}
}

func TestHitEnginesSkipsHeaders(t *testing.T) {
	s, e := open(t)

	// The very first body row is the "Engine & Model" section header,
	// which Hit steps over rather than treating as a target.
	if got := s.hit(10, e.Frame.Body.Y+4, e); got.Kind != point.None {
		t.Errorf("hitEngines on a header row = %+v, want the zero point.Target", got)
	}
}

// The screen where the engine is chosen says what is left of each one, and
// says it beside the engine rather than beside a model: a window belongs to
// the engine, and a model with a percentage next to it would be claiming a
// cap of its own.
func TestTheEnginePickerCarriesEachEnginesWindows(t *testing.T) {
	s, e := open(t)

	e.Quota = func(engine string) roster.Reading {
		if engine != "claude" {
			return roster.Reading{Engine: engine, Sourced: false}
		}

		return roster.Reading{
			Engine:  engine,
			Sourced: true,
			Windows: []roster.Window{{Label: "5h", Pct: 2}, {Label: "7d", Pct: 77}},
		}
	}

	rows := s.View(30, 100, e)

	var claude, model string

	for _, row := range rows {
		if strings.Contains(row, "claude") {
			claude = row
		}

		if strings.Contains(row, "sonnet") {
			model = row
		}
	}

	if claude == "" {
		t.Fatal("no claude row on the engine screen")
	}

	for _, want := range []string{"2% 5h", "77% 7d"} {
		if !strings.Contains(claude, want) {
			t.Errorf("engine row %q does not carry %q", claude, want)
		}
	}

	if model != "" && strings.Contains(model, "%") {
		t.Errorf("model row %q carries a percentage, which is the engine's and not its own", model)
	}

	// An engine that is not installed carries nothing: that row is about the
	// setup it still needs, and a quota beside it is an answer to a question
	// nobody standing there is asking.
	for _, row := range rows {
		if strings.Contains(row, "setup required") && strings.Contains(row, "%") {
			t.Errorf("row %q carries a quota for an engine that is not installed", row)
		}
	}

	// An engine nobody can read says so where it is chosen, for the same
	// reason it says so on the status line: silence there is read as zero.
	unsourced := s.engineQuota(engineRow{kind: rowEngine, engine: "codex"}, e)
	if !strings.Contains(unsourced, "no quota source") {
		t.Errorf("unsourced engine = %q, want it to say it has no source", unsourced)
	}

	// And one paid per token says that instead: there is no window to be at
	// the end of, which is an answer and not an absence.
	e.Quota = func(engine string) roster.Reading {
		return roster.Reading{Engine: engine, Money: true}
	}

	if got := s.engineQuota(engineRow{kind: rowEngine, engine: "opencode"}, e); !strings.Contains(got, "per token") {
		t.Errorf("metered engine = %q, want it to say it is paid per token", got)
	}

	// A source that is there and answers nothing is a third thing, and the
	// one worth going to look at: a base URL nothing serves /quota on read
	// exactly like an engine with no proxy at all.
	e.Quota = func(engine string) roster.Reading {
		return roster.Reading{Engine: engine, Sourced: true}
	}

	if got := s.engineQuota(engineRow{kind: rowEngine, engine: "codex"}, e); !strings.Contains(got, "answered nothing") {
		t.Errorf("silent source = %q, want it to say the source answered nothing", got)
	}
}
