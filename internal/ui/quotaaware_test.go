package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

// onSubscription is a window whose engine buys a window of use rather than
// an amount of it, which is what every claude and codex plan is.
func onSubscription(m Model) Model {
	m.opts.Quota = func(engine string) QuotaReading {
		return QuotaReading{
			Engine: engine, Money: false, Sourced: true,
			Windows: []QuotaWindow{{Key: "5h", Label: "5h", Pct: 40, ResetsIn: time.Hour}},
		}
	}

	return m
}

// metered is a window whose engine bills for what it used.
func metered(m Model) Model {
	m.opts.Quota = func(engine string) QuotaReading {
		return QuotaReading{Engine: engine, Money: true, Sourced: true}
	}

	return m
}

func running(cost float64, engine string) []view.Task {
	return []view.Task{{
		Repo: "payments", ID: "ACME-1", Title: "in flight", Band: view.Running,
		Engine: engine, Phase: "implement", Since: ago(time.Minute), Cost: cost,
	}}
}

// TestTheCostTabDoesNotPriceASubscription. $0.0000 against a phase that ran
// for twenty minutes reads as "this was free", and under a subscription the
// money left the account in advance: a share of it attributed to one run is
// arithmetic on a charge nobody made.
func TestTheCostTabDoesNotPriceASubscription(t *testing.T) {
	m, _ := onTab(t, tabCost, []view.Entry{
		{Kind: "phase.started", Phase: "implement", At: ago(2 * time.Minute), Engine: "claude", Model: "opus"},
		{Kind: "phase.finished", Phase: "implement", At: ago(time.Minute)},
	})
	m = onSubscription(m)

	lines := strings.Join(m.costLines(), "\n")
	if strings.Contains(ansi.Strip(lines), "$") {
		t.Errorf("the cost tab prices a subscription in dollars:\n%s", ansi.Strip(lines))
	}

	if !strings.Contains(strings.ToLower(ansi.Strip(lines)), "subscription") {
		t.Errorf("the cost tab does not say what the unit is instead:\n%s", ansi.Strip(lines))
	}
}

// TestTheCostTabStillPricesAnEngineThatCharges.
func TestTheCostTabStillPricesAnEngineThatCharges(t *testing.T) {
	m, _ := onTab(t, tabCost, []view.Entry{
		{Kind: "phase.started", Phase: "implement", At: ago(2 * time.Minute), Engine: "codex", Model: "gpt"},
		{Kind: "phase.finished", Phase: "implement", At: ago(time.Minute), Cost: 0.25},
	})
	m = metered(m)

	if lines := ansi.Strip(strings.Join(m.costLines(), "\n")); !strings.Contains(lines, "$0.25") {
		t.Errorf("the cost tab does not price an engine that bills:\n%s", lines)
	}
}

// TestTheHeaderSaysWhatIsRunningCosts. What a reader deciding whether to
// start another run is asking is not what the board has spent since it was
// made — it is what is on the meter right now.
func TestTheHeaderSaysWhatIsRunningCosts(t *testing.T) {
	m, _ := testModel(t, 200, 30)
	m = metered(m)
	m.board = fixtureBoard(running(0.42, "codex"), 1)

	if line := ansi.Strip(m.headerLine(200)); !strings.Contains(line, "$0.42") {
		t.Errorf("the header does not say what the running work has cost:\n%s", line)
	}
}

// TestTheHeaderPricesNoSubscriptionRun. The quota chip beside it is what
// says how much is left, and it is the only honest number there is.
func TestTheHeaderPricesNoSubscriptionRun(t *testing.T) {
	m, _ := testModel(t, 200, 30)
	m = onSubscription(m)
	m.board = fixtureBoard(running(0.42, "claude"), 1)

	line := ansi.Strip(m.headerLine(200))
	if strings.Contains(line, "$0.42") {
		t.Errorf("the header prices a subscription run in dollars:\n%s", line)
	}

	if !strings.Contains(line, "%") {
		t.Errorf("the header says nothing about the window that is the real unit:\n%s", line)
	}
}

// TestARunThatHasCostNothingYetDrawsNoChip. The header at a hundred cells
// is contested space, and "$0.00 running" is not news — what the chip is for
// is the moment the number starts moving.
func TestARunThatHasCostNothingYetDrawsNoChip(t *testing.T) {
	m, _ := testModel(t, 200, 30)
	m = metered(m)
	m.board = fixtureBoard(running(0, "codex"), 1)

	if line := ansi.Strip(m.headerLine(200)); strings.Contains(line, "running") {
		t.Errorf("the header draws a running chip for a run that has cost nothing:\n%s", line)
	}
}

// TestAnIdleBoardSaysNothingAboutRunningCost. A chip that is always there
// says nothing; one that appears when something is running is a fact.
func TestAnIdleBoardSaysNothingAboutRunningCost(t *testing.T) {
	m, _ := testModel(t, 200, 30)
	m = metered(m)
	m.board = fixtureBoard([]view.Task{{
		Repo: "payments", ID: "ACME-2", Title: "done", Band: view.Done, Cost: 0.42, Since: ago(time.Hour),
	}}, 1)

	if line := ansi.Strip(m.headerLine(200)); strings.Contains(line, "running") {
		t.Errorf("the header talks about running work on an idle board:\n%s", line)
	}
}
