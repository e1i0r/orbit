package mcp

// Writing and removing flows through the server. handlers_test.go covers the
// listing; this covers the three tools that change what is in the flow
// directory, and the two answers that make changing it safe: a save says
// whether it has shadowed something Orbit ships, and a delete says what would
// be left resolving the name.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// onePhase is a phase list as it arrives from a tool call: decoded JSON, not
// a flow.Phase, because that is what the handler has to make sense of.
func onePhase(name, engine string) []any {
	return []any{map[string]any{"name": name, "engine": engine}}
}

// taskOnFlow writes a task the board will report as walking one flow.
func taskOnFlow(t *testing.T, s *store.Store, r repo.Repo, id, name string) {
	t.Helper()
	addTask(t, s, r, id, record.Event{
		At:   at(1),
		Kind: record.TaskCreated,
		Text: "written",
		Data: map[string]string{"flow": name},
	})
}

func TestGetFlowIsTheDocumentSaveTakes(t *testing.T) {
	_, sn, _ := oneRepo(t)

	got := call(t, sn, "orbit_get_flow", map[string]any{"name": "quick"})
	if got["origin"] != "builtin" {
		t.Errorf("origin = %v, want builtin", got["origin"])
	}

	doc := obj(t, got["flow"])
	if doc["name"] != "quick" {
		t.Errorf("the flow calls itself %v, want quick", doc["name"])
	}

	phases := list(t, doc["phases"])
	if len(phases) == 0 {
		t.Fatal("the flow came back with no phases, and phases are the flow")
	}

	if engine := obj(t, phases[0])["engine"]; engine == nil || engine == "" {
		t.Errorf("phase 1 names no engine: %#v", phases[0])
	}

	// The round trip the tool exists for: what get answers is what save takes.
	doc["name"] = "quick-mine"

	saved := call(t, sn, "orbit_save_flow", map[string]any{"name": "quick-mine", "phases": doc["phases"]})
	if saved["saved"] != true {
		t.Errorf("saving the document orbit_get_flow answered with was refused: %v", saved)
	}
}

func TestSaveFlowWritesOneTheListingThenNames(t *testing.T) {
	s, sn, _ := oneRepo(t)

	got := call(t, sn, "orbit_save_flow", map[string]any{
		"name":        "review",
		"description": "one pass over somebody else's work",
		"phases":      onePhase("read", "claude"),
	})
	if got["saved"] != true || got["shadows"] != false {
		t.Errorf("orbit_save_flow answered %v, want a save that shadows nothing", got)
	}

	if want := s.FlowDir() + "/review.json"; got["path"] != want {
		t.Errorf("saved at %v, want %q", got["path"], want)
	}

	for _, entry := range list(t, call(t, sn, "orbit_list_flows", nil)["flows"]) {
		f := obj(t, entry)
		if f["name"] != "review" {
			continue
		}

		if f["origin"] != "user" {
			t.Errorf("review is listed as %v, want user", f["origin"])
		}

		if f["description"] != "one pass over somebody else's work" {
			t.Errorf("description = %v, want the one that was saved", f["description"])
		}

		return
	}

	t.Error("orbit_list_flows does not name the flow that was just saved")
}

func TestSaveFlowCopiesTheFlowItWasToldToStartFrom(t *testing.T) {
	_, sn, _ := oneRepo(t)

	careful, err := flow.Resolve(nil, "careful")
	if err != nil {
		t.Fatalf("resolve the built-in careful: %v", err)
	}

	got := call(t, sn, "orbit_save_flow", map[string]any{"name": "careful-mine", "from": "careful"})

	phases := list(t, got["phases"])
	if len(phases) != len(careful.Phases) {
		t.Fatalf("the copy walks %d phases, want the %d careful walks", len(phases), len(careful.Phases))
	}

	if name := obj(t, phases[0])["name"]; name != careful.Phases[0].Name {
		t.Errorf("phase 1 is %v, want %q", name, careful.Phases[0].Name)
	}

	if doc := obj(t, call(t, sn, "orbit_get_flow", map[string]any{"name": "careful-mine"})["flow"]); doc["description"] != careful.Description {
		t.Errorf("description = %v, want the one careful carries", doc["description"])
	}
}

func TestSaveFlowNeedsPhasesOrAFlowToCopy(t *testing.T) {
	_, sn, _ := oneRepo(t)

	said := refused(t, sn, "orbit_save_flow", map[string]any{"name": "review"})
	if !strings.Contains(said, "from") {
		t.Errorf("the refusal does not say how to give it phases: %s", said)
	}

	if said := refused(t, sn, "orbit_save_flow", map[string]any{"name": "review", "from": "nothing-ships-this"}); !strings.Contains(said, "nothing-ships-this") {
		t.Errorf("the refusal does not name the flow it could not copy: %s", said)
	}
}

// A field nobody declared is an error the caller reads. Saving the phase
// without it would leave a flow that fails at the moment it is walked, which
// is after a worktree and a process.
func TestSaveFlowRefusesAPhaseFieldNobodyDeclared(t *testing.T) {
	_, sn, _ := oneRepo(t)

	said := refused(t, sn, "orbit_save_flow", map[string]any{
		"name":   "review",
		"phases": []any{map[string]any{"name": "read", "engines": "claude"}},
	})
	if !strings.Contains(said, "engines") {
		t.Errorf("the refusal does not name the field that was wrong: %s", said)
	}

	if refused(t, sn, "orbit_get_flow", map[string]any{"name": "review"}) == "" {
		t.Error("the refused flow was saved anyway")
	}
}

func TestFlowToolsNeedAName(t *testing.T) {
	_, sn, _ := oneRepo(t)
	for _, name := range []string{"orbit_get_flow", "orbit_save_flow", "orbit_delete_flow"} {
		if said := refused(t, sn, name, nil); !strings.Contains(said, "name") {
			t.Errorf("%s with no name said %q, want it to ask for one", name, said)
		}
	}
}

// A flow saved under the name of one Orbit ships is the extension mechanism
// working as designed — and the one case where a save changes what an
// already-written task will do, so the answer says so.
func TestSaveFlowUnderABuiltinNameSaysItShadowsIt(t *testing.T) {
	_, sn, _ := oneRepo(t)

	got := call(t, sn, "orbit_save_flow", map[string]any{"name": "quick", "phases": onePhase("mine", "codex")})
	if got["shadows"] != true {
		t.Errorf("orbit_save_flow answered %v, want it to say the save shadows a flow orbit ships", got)
	}

	if origin := call(t, sn, "orbit_get_flow", map[string]any{"name": "quick"})["origin"]; origin != "shadow" {
		t.Errorf("quick now resolves as %v, want shadow", origin)
	}
}

func TestDeleteFlowPutsTheBuiltinBack(t *testing.T) {
	_, sn, _ := oneRepo(t)
	call(t, sn, "orbit_save_flow", map[string]any{"name": "quick", "phases": onePhase("mine", "codex")})

	got := call(t, sn, "orbit_delete_flow", map[string]any{"name": "quick"})
	if got["deleted"] != true || got["restored_builtin"] != true {
		t.Errorf("orbit_delete_flow answered %v, want a delete that put the shipped flow back", got)
	}

	after := call(t, sn, "orbit_get_flow", map[string]any{"name": "quick"})
	if after["origin"] != "builtin" {
		t.Errorf("quick resolves as %v after its shadow was deleted, want builtin", after["origin"])
	}

	if name := obj(t, list(t, obj(t, after["flow"])["phases"])[0])["name"]; name == "mine" {
		t.Error("quick still walks the phase of the deleted shadow")
	}
}

func TestDeleteFlowRefusesAFlowOrbitShips(t *testing.T) {
	_, sn, _ := oneRepo(t)
	if said := refused(t, sn, "orbit_delete_flow", map[string]any{"name": "quick"}); !strings.Contains(said, "built into orbit") {
		t.Errorf("deleting a built-in said %q, want it to say why it cannot", said)
	}
}

// Nothing would resolve the name afterwards, so the tasks written against it
// would have nothing to walk. The refusal names them, and a caller that means
// it says force.
func TestDeleteFlowRefusesWhileTasksAreWrittenAgainstIt(t *testing.T) {
	s, sn, r := oneRepo(t)
	call(t, sn, "orbit_save_flow", map[string]any{"name": "review", "phases": onePhase("read", "claude")})
	taskOnFlow(t, s, r, "PAY-1", "review")

	said := refused(t, sn, "orbit_delete_flow", map[string]any{"name": "review"})
	for _, want := range []string{"PAY-1", "force"} {
		if !strings.Contains(said, want) {
			t.Errorf("the refusal does not mention %q: %s", want, said)
		}
	}

	if call(t, sn, "orbit_get_flow", map[string]any{"name": "review"})["origin"] != "user" {
		t.Error("the flow is gone after a refused delete")
	}

	got := call(t, sn, "orbit_delete_flow", map[string]any{"name": "review", "force": true})
	if got["deleted"] != true {
		t.Fatalf("a forced delete answered %v, want it done", got)
	}

	if users := list(t, got["used_by"]); len(users) != 1 || str(t, users[0]) != r.Name+"/PAY-1" {
		t.Errorf("used_by = %v, want the task that is now stranded", got["used_by"])
	}

	if refused(t, sn, "orbit_get_flow", map[string]any{"name": "review"}) == "" {
		t.Error("the name still resolves after a forced delete")
	}
}

// Shadowing is the case where tasks written against the name are not
// stranded: the shipped flow is underneath, so the delete goes ahead without
// being forced.
func TestDeleteFlowDoesNotAskAboutTasksWhenAShippedFlowIsUnderneath(t *testing.T) {
	s, sn, r := oneRepo(t)
	call(t, sn, "orbit_save_flow", map[string]any{"name": "quick", "phases": onePhase("mine", "codex")})
	taskOnFlow(t, s, r, "PAY-1", "quick")

	if got := call(t, sn, "orbit_delete_flow", map[string]any{"name": "quick"}); got["deleted"] != true {
		t.Errorf("orbit_delete_flow answered %v, want it done: quick goes on resolving", got)
	}
}

func TestDeleteFlowSaysWhenThereIsNoSuchFlow(t *testing.T) {
	_, sn, _ := oneRepo(t)
	if said := refused(t, sn, "orbit_delete_flow", map[string]any{"name": "nothing-wrote-this"}); !strings.Contains(said, "nothing-wrote-this") {
		t.Errorf("the refusal does not name the flow: %s", said)
	}
}

// TestEveryPermissionTheSchemaOffersIsOneAFlowCanBeSavedWith is the promise
// tools.go states in words: a schema here is a promise the handler keeps.
//
// The enum is read out of the tool list rather than out of internal/flow,
// because what is being tested is what a model is shown. A list of literals
// that has drifted from the vocabulary would pass a test that asked
// internal/flow what the permissions are, and fail this one — the caller
// offered a word, passed it back, and had the save refused.
func TestEveryPermissionTheSchemaOffersIsOneAFlowCanBeSavedWith(t *testing.T) {
	_, sn, _ := oneRepo(t)

	offered := permissionEnum(t)
	if len(offered) == 0 {
		t.Fatal("orbit_save_flow offers no permissions at all, so a model composing a phase has to invent them")
	}

	for _, perm := range offered {
		phase := map[string]any{"name": "implement", "engine": "claude", "permissions": []any{perm}}

		got := call(t, sn, "orbit_save_flow", map[string]any{"name": "perm-" + perm, "phases": []any{phase}})
		if got["saved"] != true {
			t.Errorf("a phase asking for %q was not saved: %v", perm, got)
		}
	}
}

// permissionEnum digs the permissions a phase may ask for out of the schema
// of orbit_save_flow, which is three levels down: the phases array, one
// phase, its permissions array.
func permissionEnum(t *testing.T) []string {
	t.Helper()

	for _, tool := range Tools() {
		if tool.Name != "orbit_save_flow" {
			continue
		}

		phases, ok := tool.InputSchema.Properties["phases"]
		if !ok || phases.Items == nil {
			t.Fatal("orbit_save_flow declares no phases array")
		}

		perms, ok := phases.Items.Properties["permissions"]
		if !ok || perms.Items == nil {
			t.Fatal("a phase declares no permissions array")
		}

		return perms.Items.Enum
	}

	t.Fatal("orbit_save_flow is not in the tool list")

	return nil
}

// TestSaveFlowKeepsHowManyAttemptsItAllows is the cap of 3.1 arriving
// through the door a model uses: a flow that says one attempt has to be
// savable and readable, or the setting exists only in a file nobody here
// can write.
func TestSaveFlowKeepsHowManyAttemptsItAllows(t *testing.T) {
	_, sn, _ := oneRepo(t)

	call(t, sn, "orbit_save_flow", map[string]any{
		"name": "one-shot",
		// float64 and not 1, because that is what a client's JSON arrives
		// as and what intArg reads.
		"attempts": float64(1),
		"phases":   onePhase("implement", "claude"),
	})

	doc := obj(t, call(t, sn, "orbit_get_flow", map[string]any{"name": "one-shot"})["flow"])
	if got := fmt.Sprint(doc["attempts"]); got != "1" {
		t.Errorf("the saved flow allows %v attempts, want the 1 it was saved with", doc["attempts"])
	}
}
