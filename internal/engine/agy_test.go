package engine

import (
	"slices"
	"strings"
	"testing"
	"time"
)

func TestAgyInterface(t *testing.T) {
	a := NewAgy()
	if a.Name() != "agy" {
		t.Errorf("Name() = %q, want agy", a.Name())
	}

	if !a.CanResume() {
		t.Error("CanResume() = false, want true: --conversation takes the id every stream opens with")
	}

	if a.CanThink() {
		t.Error("CanThink() = true, want false: agy has no thinking flag, only --effort")
	}

	if len(a.Models()) == 0 {
		t.Error("Models() is empty")
	}

	if len(a.Efforts()) == 0 {
		t.Error("Efforts() is empty")
	}
}

// TestAgyArgs is the argv Antigravity CLI 1.1.25 actually parses.
func TestAgyArgs(t *testing.T) {
	req := Request{
		Prompt:      "refactor the handler",
		Model:       "gemini-3.1-pro-high",
		Effort:      "high",
		Resume:      "4eee6f79-1d3e-4537-8951-95a196ac5152",
		Permissions: []string{PermissionRead, PermissionRepo},
	}

	got, err := agyArgs(req)
	if err != nil {
		t.Fatalf("agyArgs: %v", err)
	}

	want := []string{
		"--print", "refactor the handler",
		"--output-format", "stream-json",
		"--model", "gemini-3.1-pro-high",
		"--effort", "high",
		"--conversation", "4eee6f79-1d3e-4537-8951-95a196ac5152",
		"--dangerously-skip-permissions",
	}

	if !slices.Equal(got, want) {
		t.Errorf("agyArgs = %v, want %v", got, want)
	}
}

// TestAgyRefusesAPostureItCannotEnforce.
//
// agy asks a person at the prompt, and a headless run has nobody there: it
// auto-denies the tool and says so on its own stream. A read phase would
// spend its tokens being told no and the record would carry a posture the
// run never had, so it does not start.
func TestAgyRefusesAPostureItCannotEnforce(t *testing.T) {
	for _, names := range [][]string{nil, {PermissionRead}, {PermissionNetwork}} {
		if _, err := agyArgs(Request{Prompt: "look", Permissions: names}); err == nil {
			t.Errorf("agyArgs with %v built a command line, want it refused", names)
		}
	}
}

// TestAgyNeverSkipsPermissionsUnasked: the flag that turns the question off
// is the one repo buys, and no narrower posture reaches it.
func TestAgyNeverSkipsPermissionsUnasked(t *testing.T) {
	got, err := agyArgs(Request{Prompt: "write it", Permissions: []string{PermissionRepo}})
	if err != nil {
		t.Fatalf("agyArgs: %v", err)
	}

	if !slices.Contains(got, "--dangerously-skip-permissions") {
		t.Errorf("agyArgs = %v, want the flag that lets a headless run act at all", got)
	}

	if _, err := agyArgs(Request{Prompt: "look", Permissions: []string{"telepathy"}}); err == nil {
		t.Error("agyArgs accepted a permission no engine knows")
	}
}

// TestAgyModelsAreWhatTheBinaryTakes: ids as `agy models` prints them, and a
// default that names nothing.
func TestAgyModelsAreWhatTheBinaryTakes(t *testing.T) {
	models := NewAgy().Models()
	if models[0].ID != "" {
		t.Errorf("the first model is %q, want the default that names nothing", models[0].ID)
	}

	for _, m := range models[1:] {
		if strings.TrimSpace(m.ID) != m.ID || m.ID == "" || m.Label == "" {
			t.Errorf("model %+v is not a pair of an id and a label", m)
		}
	}
}

// TestAgySaysNothingAboutASessionItCannotRead. Its conversations are
// protobuf blobs with no schema, and the interface's answer for a transcript
// nobody has mapped is nothing rather than a guess.
func TestAgySaysNothingAboutASessionItCannotRead(t *testing.T) {
	turns, err := NewAgy().Transcript(t.TempDir(), time.Time{})
	if err != nil {
		t.Fatalf("Transcript: %v", err)
	}

	if len(turns) != 0 {
		t.Errorf("Transcript answered %d turns, want none", len(turns))
	}
}
