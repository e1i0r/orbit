package mcp

// What the tools answer, and the four things the first version of this
// package got wrong: the band counts were read off the wrong indices, the
// band filter matched nothing, a task id was minted from a process id, and
// every question was asked of the working directory.

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/record"
)

// at is a fixed clock, so a record a test writes reads back in the order it
// was written whatever the machine's is doing.
func at(n int) time.Time {
	return time.Date(2026, 8, 27, 12, 0, n, 0, time.UTC)
}

// TestTheCountsAreOfTheBandsTheyName is the finding. view.Band's iota is
// ToDo, NeedsYou, Running, Done, and the first version of this package read
// Counts[1] as running and Counts[2] as needs you — so a board with one
// failed task reported one running task, and a supervisor watching for work
// that needed it saw none.
func TestTheCountsAreOfTheBandsTheyName(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-1", record.Event{At: at(1), Kind: record.TaskCreated, Text: "written and never started"})
	addTask(t, s, r, "PAY-2",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "failed"},
		record.Event{At: at(2), Kind: record.TaskStarted},
		record.Event{At: at(3), Kind: record.TaskFailed, Text: "the build broke"})
	addTask(t, s, r, "PAY-3",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "finished"},
		record.Event{At: at(2), Kind: record.TaskStarted},
		record.Event{At: at(3), Kind: record.TaskFinished})

	counts, ok := call(t, sn, "orbit_get_board_summary", nil)["counts"].(map[string]any)
	if !ok {
		t.Fatal("the summary carries no counts")
	}

	for band, want := range map[string]float64{"todo": 1, "needs_you": 1, "done": 1, "running": 0} {
		if counts[band] != want {
			t.Errorf("counts[%q] = %v, want %v — the whole counts map is %v", band, counts[band], want, counts)
		}
	}
}

// TestTheBandFilterFindsTheBandItNames is the second finding: the schema
// declared needs_you and the filter compared against view.Band.String(),
// which says "needs you" with a space, so every filtered call answered with
// nothing at all and looked like an empty board.
func TestTheBandFilterFindsTheBandItNames(t *testing.T) {
	s, sn, r := oneRepo(t)
	addTask(t, s, r, "PAY-1", record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"})
	addTask(t, s, r, "PAY-2",
		record.Event{At: at(1), Kind: record.TaskCreated, Text: "failed"},
		record.Event{At: at(2), Kind: record.TaskStarted},
		record.Event{At: at(3), Kind: record.TaskFailed, Text: "broke"})

	got := call(t, sn, "orbit_list_tasks", map[string]any{"band": "needs_you"})

	tasks, ok := got["tasks"].([]any)
	if !ok || len(tasks) != 1 {
		t.Fatalf("the needs_you band answered %v, want exactly PAY-2", got["tasks"])
	}

	row0, ok := tasks[0].(map[string]any)
	if !ok || row0["id"] != "PAY-2" {
		t.Fatalf("the needs_you band answered %v, want PAY-2", tasks[0])
	}

	if row0["band"] != "needs_you" {
		t.Errorf("the row names its band %q, want the same token the filter takes", row0["band"])
	}
}

// TestAnUnknownBandIsRefusedWithTheOnesThatWork. A filter nobody spelled
// right must not answer "no tasks", which is indistinguishable from a quiet
// board.
func TestAnUnknownBandIsRefusedWithTheOnesThatWork(t *testing.T) {
	_, sn, _ := oneRepo(t)

	said := refused(t, sn, "orbit_list_tasks", map[string]any{"band": "in_progress"})
	for _, name := range bandNames() {
		if !strings.Contains(said, name) {
			t.Errorf("the refusal does not offer %q: %s", name, said)
		}
	}
}

// TestTasksAreFoundThroughTheRepositoriesOrbitKnows is the third finding,
// and the one that made the server useless where it is actually run: every
// question was asked of the working directory, and a desktop application
// spawns this process in its own bundle. A session with no root of its own
// asks the state root instead.
func TestTasksAreFoundThroughTheRepositoriesOrbitKnows(t *testing.T) {
	s, rooted, r := oneRepo(t)
	addTask(t, s, r, "PAY-1", record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"})

	// The same store, and a session pointed nowhere. The working directory
	// is the test binary's, which holds no repository of this fixture's.
	rootless := Session{Version: "test"}
	got := call(t, rootless, "orbit_list_tasks", nil)

	tasks, ok := got["tasks"].([]any)
	if !ok || len(tasks) != 1 {
		t.Fatalf("a session with no root found %v, want the one task in the repository orbit knows", got["tasks"])
	}

	alsoFound, ok := call(t, rooted, "orbit_list_tasks", nil)["tasks"].([]any)
	if !ok || len(alsoFound) != 1 {
		t.Errorf("the rooted session found %v, and the rootless one found the task", alsoFound)
	}
}

// TestTheAnswerSaysWhereItLooked. An empty board and a board looked for in
// the wrong place read identically, and the second is the failure a client
// actually has.
func TestTheAnswerSaysWhereItLooked(t *testing.T) {
	_, sn, _ := oneRepo(t)

	roots, ok := call(t, sn, "orbit_list_repos", nil)["roots"].([]any)
	if !ok || len(roots) == 0 {
		t.Fatal("orbit_list_repos does not say which directories it looked in")
	}

	if roots[0] != sn.Root {
		t.Errorf("it says it looked in %v, and it was pointed at %q", roots[0], sn.Root)
	}
}

func TestListReposNamesTheRepositoryAndItsPath(t *testing.T) {
	_, sn, r := oneRepo(t)

	repos, ok := call(t, sn, "orbit_list_repos", nil)["repos"].([]any)
	if !ok || len(repos) != 1 {
		t.Fatalf("orbit_list_repos found %v, want the one repository in the workspace", repos)
	}

	got, ok := repos[0].(map[string]any)
	if !ok || got["name"] != r.Name || got["path"] != r.Path {
		t.Errorf("orbit_list_repos answered %v, want name %q at %q", repos[0], r.Name, r.Path)
	}
}

func TestListFlowsCarriesThePhasesEachOneWalks(t *testing.T) {
	_, sn, _ := oneRepo(t)

	flows, ok := call(t, sn, "orbit_list_flows", nil)["flows"].([]any)
	if !ok || len(flows) == 0 {
		t.Fatal("orbit_list_flows found no flows, and orbit ships several")
	}

	for _, entry := range flows {
		f, ok := entry.(map[string]any)
		if !ok || f["name"] == "" {
			t.Fatalf("a flow came back as %#v", entry)
		}

		if f["error"] != nil {
			continue
		}

		phases, ok := f["phases"].([]any)
		if !ok || len(phases) == 0 {
			t.Errorf("flow %v carries no phases, which is the only reason to ask for the list", f["name"])
		}
	}
}

// TestATaskIsFoundByIdAloneAcrossRepositories. A model reading a row off
// orbit_list_tasks has the id and nothing else; making it also supply a path
// it was never given is how a tool call becomes a guess.
func TestATaskIsFoundByIdAloneAcrossRepositories(t *testing.T) {
	s, work := newRoot(t)
	gitRepo(t, work, "payments")
	ledger := gitRepo(t, work, "ledger")
	sn := Session{Root: work, Version: "test"}

	addTask(t, s, ledger, "LED-1", record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"})

	got := call(t, sn, "orbit_inspect_task", map[string]any{"task_id": "LED-1"})
	if got["repo"] != "ledger" {
		t.Errorf("LED-1 was found in %v, want ledger", got["repo"])
	}
}

// TestAnAmbiguousIdIsRefusedWithTheRepositoriesThatHoldIt: acting on
// whichever one came first would be acting on a task the caller did not name.
func TestAnAmbiguousIdIsRefusedWithTheRepositoriesThatHoldIt(t *testing.T) {
	s, work := newRoot(t)
	payments := gitRepo(t, work, "payments")
	ledger := gitRepo(t, work, "ledger")
	sn := Session{Root: work, Version: "test"}

	addTask(t, s, payments, "SHARED-1", record.Event{At: at(1), Kind: record.TaskCreated, Text: "one"})
	addTask(t, s, ledger, "SHARED-1", record.Event{At: at(1), Kind: record.TaskCreated, Text: "two"})

	said := refused(t, sn, "orbit_add_note", map[string]any{"task_id": "SHARED-1", "text": "which one?"})
	for _, name := range []string{"payments", "ledger"} {
		if !strings.Contains(said, name) {
			t.Errorf("the refusal does not name %q: %s", name, said)
		}
	}

	got := call(t, sn, "orbit_inspect_task", map[string]any{"task_id": "SHARED-1", "repo": "ledger"})
	if got["repo"] != "ledger" {
		t.Errorf("the disambiguated call answered %v, want ledger", got["repo"])
	}
}

func TestAnUnknownTaskIsRefusedByName(t *testing.T) {
	_, sn, _ := oneRepo(t)
	for _, tool := range []string{"orbit_inspect_task", "orbit_pause_task", "orbit_cancel_task", "orbit_retry_task"} {
		if said := refused(t, sn, tool, map[string]any{"task_id": "NOPE-1"}); !strings.Contains(said, "NOPE-1") {
			t.Errorf("%s refused without naming the task: %s", tool, said)
		}
	}

	if said := refused(t, sn, "orbit_add_note", map[string]any{"task_id": "NOPE-1", "text": "hello"}); !strings.Contains(said, "NOPE-1") {
		t.Errorf("orbit_add_note refused without naming the task: %s", said)
	}
}

func TestAToolNobodySpelledRightIsRefusedWithTheOnesThatExist(t *testing.T) {
	_, sn, _ := oneRepo(t)

	res := sn.Call("orbit_do_the_thing", nil)
	if !res.IsError {
		t.Fatal("an unknown tool did not refuse")
	}

	if !strings.Contains(text(t, res), "orbit_list_tasks") {
		t.Errorf("the refusal does not name the tools that would have worked: %s", text(t, res))
	}
}

// TestASessionWithNoRootSaysWhereItLooked is the other half of the test
// above, and the half that could answer nothing at all. Where it looked was
// worked out a second time, by reopening the state root after the board had
// already been folded, and a failure there came back as an empty list —
// which a client reads as a server that looked nowhere. The roots now come
// back from the fold itself, so what a tool reports is what it read.
func TestASessionWithNoRootSaysWhereItLooked(t *testing.T) {
	s, _, r := oneRepo(t)
	addTask(t, s, r, "PAY-1", record.Event{At: at(1), Kind: record.TaskCreated, Text: "written"})

	rootless := Session{Version: "test"}

	roots, ok := call(t, rootless, "orbit_get_board_summary", nil)["roots"].([]any)
	if !ok || len(roots) == 0 {
		t.Fatal("a session with no root of its own does not say which directories it looked in")
	}

	if !slices.Contains(roots, any(r.Path)) {
		t.Errorf("it says it looked in %v, and the task it found is in %q", roots, r.Path)
	}
}
