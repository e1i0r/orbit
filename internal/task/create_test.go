package task

// Which flow a task walks is decided once, when the task is written down,
// and recorded there. These are the tests of that one decision.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// createdFlow is what a task's record says it was written against.
func createdFlow(t *testing.T, s *store.Store, tk Task) string {
	t.Helper()
	return find(t, mustEvents(t, s, tk), record.TaskCreated).Data["flow"]
}

func TestTheFlowATaskIsWrittenAgainstIsRecorded(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "retry the webhook on 5xx", "careful")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if tk.Flow != "careful" {
		t.Errorf("the task says its flow is %q, want careful", tk.Flow)
	}

	if got := createdFlow(t, s, tk); got != "careful" {
		t.Errorf(`task.created Data["flow"] = %q, want careful`, got)
	}
}

// A task written with no flow named is not a task with no flow: the choice
// falls to the user's default, and it is written down as though they had
// typed it, because a run two weeks later must not walk something else
// because a setting moved in between.
func TestAnEmptyFlowNameRecordsTheSettingsDefault(t *testing.T) {
	s, r := fixture(t)
	if err := s.SaveSettings(store.Settings{Flow: "careful"}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if tk.Flow != "careful" || createdFlow(t, s, tk) != "careful" {
		t.Errorf("the task was written against %q, want the settings default careful", tk.Flow)
	}
}

func TestASettingsFileWithNoDefaultRecordsTheTaskFlow(t *testing.T) {
	s, r := fixture(t)
	// A settings file that was saved before there was a flow to set, which
	// is every settings file that already exists.
	if err := s.SaveSettings(store.Settings{Autopilot: true}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if tk.Flow != flow.Default || createdFlow(t, s, tk) != flow.Default {
		t.Errorf("the task was written against %q, want %q", tk.Flow, flow.Default)
	}
}

func TestAStoreThatHasNeverSavedSettingsRecordsTheTaskFlow(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "x", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if tk.Flow != flow.Default || createdFlow(t, s, tk) != flow.Default {
		t.Errorf("the task was written against %q, want %q", tk.Flow, flow.Default)
	}
}

// internal/store may not import internal/flow — it imports nothing of
// Orbit's — so the word "task" is written down in both places. This is what
// keeps the two copies from drifting apart.
func TestTheStoresDefaultFlowIsTheFlowPackagesDefault(t *testing.T) {
	s, _ := fixture(t)

	cfg, err := s.Settings()
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}

	if cfg.Flow != flow.Default {
		t.Errorf("a store with no settings file defaults to flow %q, and internal/flow defaults to %q", cfg.Flow, flow.Default)
	}
}

// Refusing to write a task down because a JSON file is missing would lose
// the sentence the user typed — the one thing in this system nobody can
// regenerate. The name is validated where it is walked, and nowhere else.
func TestNothingIsResolvedWhileTheTaskIsBeingWrittenDown(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "ACME-1", "retry the webhook on 5xx", "nonesuch")
	if err != nil {
		t.Fatalf("Create refused a flow it should not have resolved: %v", err)
	}

	if _, err := flow.Resolve(s, tk.Flow); err == nil {
		t.Error("running the task would have walked something, and the flow does not exist")
	}
}

func TestLoadRecoversTheFlowFromTheRecord(t *testing.T) {
	s, r := fixture(t)
	if _, err := Create(s, r, "ACME-1", "x", "careful"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	tk, err := Load(s, r, "ACME-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if tk.Flow != "careful" {
		t.Errorf("the loaded task says its flow is %q, want careful — a run in another process has only the record", tk.Flow)
	}
}

// A record that cannot be read is not a task that cannot be loaded. The
// written task is the thing worth having; the flow is recovered where it
// can be, and whoever needs one falls back to the default.
func TestATaskWhoseRecordIsGoneStillLoads(t *testing.T) {
	s, r := fixture(t)
	if _, err := Create(s, r, "ACME-1", "retry the webhook on 5xx", "careful"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	breakRecord(t, s)

	tk, err := Load(s, r, "ACME-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if tk.Text != "retry the webhook on 5xx" {
		t.Errorf("the loaded task says %q", tk.Text)
	}

	if tk.Flow != "" {
		t.Errorf("the loaded task claims flow %q from a record that is not there", tk.Flow)
	}
}

// A store satisfies the interface internal/flow declares for the directory
// user flows live in. Nothing else needs to be true for a file dropped in
// $ORBIT_HOME/flows to be a flow.
func TestAStoreIsAFlowSource(t *testing.T) {
	s, _ := fixture(t)

	var src flow.Source = s
	if src.FlowDir() == "" {
		t.Error("a store has no flow directory")
	}
}
