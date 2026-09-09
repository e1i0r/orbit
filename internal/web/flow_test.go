package web

// The kinds are written out rather than taken from internal/record. This
// package answers a wire, and a page keys off the string: a test that read
// the constant would go on passing through a rename that broke every reader.

import (
	"net/http"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/view"
)

// phasesIn is the phase list off a flow answer, by name and standing.
func phasesIn(t *testing.T, body map[string]any) map[string]string {
	t.Helper()

	out := map[string]string{}

	for _, raw := range listIn(t, body, "phases") {
		one, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("a phase came back as %T: %v", raw, raw)
		}

		name, named := one["name"].(string)
		if !named {
			t.Fatalf("a phase came back with no name: %v", one)
		}

		standing, said := one["standing"].(string)
		if !said {
			t.Fatalf("the phase %q came back with no standing: %v", name, one)
		}

		out[name] = standing
	}

	return out
}

// TestTheFlowSaysWhereTheRunGotTo. The flow is the shape of the work and the
// record is what became of it; either alone is half an answer, and the page
// cannot join them because only one of the two is on disk here.
func TestTheFlowSaysWhereTheRunGotTo(t *testing.T) {
	now := time.Now()
	s := server(aBoard{
		tasks: []view.Task{{
			ID: "LED-3", Title: "fix the total", Band: view.Running,
			Repo: "ledger", Flow: "task", Phase: "review",
		}},
		entries: []view.Entry{
			{At: now, Kind: "phase.started", Phase: "implement", Engine: "claude"},
			{At: now, Kind: "phase.finished", Phase: "implement", Cost: 1.5},
			{At: now, Kind: "phase.started", Phase: "review", Engine: "claude"},
		},
	}, nowhere{path: "/nowhere"}, "/code", built)

	code, body := ask(t, s, "/api/tasks/LED-3/flow")
	if code != http.StatusOK {
		t.Fatalf("the flow answered %d: %v", code, body)
	}

	if got := body["name"]; got != "task" {
		t.Errorf("the flow is called %v, want the one the task names", got)
	}

	standings := phasesIn(t, body)
	if standings["implement"] != "done" {
		t.Errorf("the phase that finished stands as %q", standings["implement"])
	}

	if standings["review"] != "running" {
		t.Errorf("the phase the task is in stands as %q", standings["review"])
	}
}

// TestAPhaseNobodyReachedIsPending, and the phases carry what the flow set
// them up with whether or not they ran: a reader deciding whether to release
// a gate is reading the phase that has not happened yet.
func TestAPhaseNobodyReachedIsPending(t *testing.T) {
	s := server(aBoard{tasks: []view.Task{
		{ID: "LED-4", Title: "later", Band: view.ToDo, Repo: "ledger", Flow: "task"},
	}}, nowhere{path: "/nowhere"}, "/code", built)

	_, body := ask(t, s, "/api/tasks/LED-4/flow")

	for name, standing := range phasesIn(t, body) {
		if standing != "pending" {
			t.Errorf("%s stands as %q on a task nobody has run", name, standing)
		}
	}

	first, ok := listIn(t, body, "phases")[0].(map[string]any)
	if !ok {
		t.Fatal("the first phase is not an object")
	}

	if first["engine"] != "claude" {
		t.Errorf("a phase that has not run says its engine is %v", first["engine"])
	}
}

// TestAFlowThatIsNotThereIsSaidAndNotRefused. The name is in the record and
// the file behind it is on somebody's disk: "the flow this task names is
// gone" is the whole of what the reader needs, and a 500 would tell them the
// server is broken instead.
func TestAFlowThatIsNotThereIsSaidAndNotRefused(t *testing.T) {
	s := server(aBoard{tasks: []view.Task{
		{ID: "LED-5", Title: "orphan", Band: view.ToDo, Repo: "ledger", Flow: "gone"},
	}}, nowhere{path: "/nowhere"}, "/code", built)

	code, body := ask(t, s, "/api/tasks/LED-5/flow")
	if code != http.StatusOK {
		t.Fatalf("a missing flow answered %d: %v", code, body)
	}

	if body["failed"] == nil || body["failed"] == "" {
		t.Errorf("a missing flow said nothing about why: %v", body)
	}
}

// TestImpactHasNothingToLookInForATaskNobodyRan. The same fact the diff
// route answers, in the same shape: a task with no checkout is the ordinary
// state of the To Do band, not a failure.
func TestImpactHasNothingToLookInForATaskNobodyRan(t *testing.T) {
	s := server(aBoard{tasks: []view.Task{
		{ID: "LED-6", Title: "later", Band: view.ToDo, Repo: "ledger"},
	}}, nowhere{path: "/nowhere"}, "/code", built)

	code, body := ask(t, s, "/api/tasks/LED-6/impact")
	if code != http.StatusOK {
		t.Fatalf("the impact answered %d: %v", code, body)
	}

	if body["missing"] != true {
		t.Errorf("a task with no checkout does not say so: %v", body)
	}
}

// TestImpactKeepsTheThreeClaimsApart. The coupling is the history's word,
// the contracts are sentences somebody wrote as test names, and the delta is
// the engine's account of its own work. They carry different weight, so they
// travel as three fields and not as one list of findings.
func TestImpactKeepsTheThreeClaimsApart(t *testing.T) {
	got := impactOf("LED-9", repo.Impact{
		Changed:   []string{"pay/total.go"},
		Commits:   40,
		Coupled:   []repo.Coupled{{File: "pay/total_test.go", With: "pay/total.go", Times: 9, Of: 10}},
		Contracts: []repo.Contract{{File: "pay/total_test.go", Says: "rejects negative amounts"}},
	}, []view.Entry{
		{Kind: "task.delta", Delta: &view.Delta{Needs: []string{"call it with cents"}}},
		{Kind: "phase.finished", Phase: "review"},
	})

	if len(got.Coupled) != 1 || got.Coupled[0].Ratio != 0.9 {
		t.Errorf("the coupling came back as %v", got.Coupled)
	}

	if len(got.Contracts) != 1 || got.Contracts[0].Says != "rejects negative amounts" {
		t.Errorf("the contracts came back as %v", got.Contracts)
	}

	if got.Delta == nil || len(got.Delta.Needs) != 1 {
		t.Errorf("the engine's own account came back as %v", got.Delta)
	}

	if got.Commits != 40 {
		t.Errorf("the reading says it read %d commits", got.Commits)
	}
}

// TestAFileOutsideTheWorktreeIsNotServed. The path arrives in a query
// string, and a reader who can ask for a file can ask for one two
// directories up; it answers as a file that is not there rather than as a
// refusal, so nothing about what is outside can be learned from it either.
func TestAFileOutsideTheWorktreeIsNotServed(t *testing.T) {
	s := server(aBoard{tasks: []view.Task{
		{ID: "LED-7", Title: "escape", Band: view.Running, Repo: "ledger"},
	}}, nowhere{path: "/nowhere"}, "/code", built)

	code, body := ask(t, s, "/api/tasks/LED-7/file?path=../../../etc/passwd")
	if code != http.StatusOK {
		t.Fatalf("the file route answered %d: %v", code, body)
	}

	if body["missing"] != true || body["text"] != nil {
		t.Errorf("a path outside the worktree came back with something: %v", body)
	}
}

// TestTheFileRouteSaysWhichFileItNeeds.
func TestTheFileRouteSaysWhichFileItNeeds(t *testing.T) {
	s := server(aBoard{tasks: []view.Task{
		{ID: "LED-8", Title: "no path", Band: view.Running, Repo: "ledger"},
	}}, nowhere{path: "/nowhere"}, "/code", built)

	code, body := ask(t, s, "/api/tasks/LED-8/file")
	if code != http.StatusBadRequest {
		t.Fatalf("a request with no path answered %d: %v", code, body)
	}
}
