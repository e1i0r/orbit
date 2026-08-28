package engine

import (
	"context"
	"fmt"
	"slices"
)

// OpenCode runs the opencode command line in headless mode.
type OpenCode struct{}

var _ Engine = OpenCode{}

// NewOpenCode returns the OpenCode CLI adapter.
func NewOpenCode() OpenCode { return OpenCode{} }

// Name identifies the engine in the record.
func (OpenCode) Name() string { return "opencode" }

// CanResume reports whether opencode can carry on a previous session.
func (OpenCode) CanResume() bool { return true }

// Models returns the models opencode supports.
//
// The IDs are what `opencode models` prints and what --model takes:
// provider-qualified, and rejected by opencode without the qualifier. The
// list this replaced carried five bare names — deepseek-r1, qwen-2.5-coder,
// llama-3.3-70b, claude-3-5-sonnet, gemini-2.5-flash — none of which
// opencode has ever accepted, and internal/task checks a phase's model
// against this list before running anything, so the only models a phase
// could name on opencode were five that could not work.
//
// Half of them are free, and they say so in their own names rather than in
// a mark this program invents: the label is the ID without the provider, so
// what a reader picks from and what opencode is told are the same string
// with nothing in between deciding.
//
// It is a written-down copy of a catalogue that moves, which is a cost taken
// deliberately: reading it at run time means shelling out to opencode before
// a dial can be drawn, on a machine where opencode may not be installed at
// all. Refresh it with `opencode models`.
func (OpenCode) Models() []Choice {
	return []Choice{
		{ID: "", Label: "default"},
		// Paid.
		{ID: "opencode/claude-opus-5", Label: "claude-opus-5"},
		{ID: "opencode/claude-sonnet-5", Label: "claude-sonnet-5"},
		{ID: "opencode/gpt-5.3-codex", Label: "gpt-5.3-codex"},
		{ID: "opencode/gemini-3.1-pro", Label: "gemini-3.1-pro"},
		{ID: "opencode/grok-4.6", Label: "grok-4.6"},
		// Free.
		{ID: "opencode/nemotron-3-ultra-free", Label: "nemotron-3-ultra-free"},
		{ID: "opencode/nemotron-3.5-lightning-free", Label: "nemotron-3.5-lightning-free"},
		{ID: "opencode/mimo-v2.5-free", Label: "mimo-v2.5-free"},
		{ID: "opencode/ling-3.0-flash-fin-free", Label: "ling-3.0-flash-fin-free"},
		{ID: "opencode/hy3-free", Label: "hy3-free"},
		{ID: "opencode/muse-spark-1.2-contributor-free", Label: "muse-spark-1.2-contributor-free"},
	}
}

// Efforts returns the effort choices opencode supports.
//
// opencode calls this a model variant and its own help names three — high,
// max and minimal — but the word after --variant is handed to whichever
// provider is behind the chosen model, so what is accepted moves with the
// model rather than with opencode. These four are the ones every reasoning
// provider behind opencode shares.
func (OpenCode) Efforts() []Choice {
	return []Choice{
		{ID: "", Label: "default"},
		{ID: "minimal", Label: "minimal"},
		{ID: "medium", Label: "medium"},
		{ID: "high", Label: "high"},
	}
}

// CanThink reports whether opencode supports thinking mode.
func (OpenCode) CanThink() bool { return true }

// spec is how opencode is driven.
//
// dirs carries .opencode/bin because that is where opencode's own installer
// puts the binary, and it adds that directory to a shell profile — so a PATH
// is only as current as the session that exported it. Orbit started from a
// terminal older than the install drew "opencode [setup required]" on the
// engine screen to a reader with opencode running in the next pane.
func (OpenCode) spec() spec {
	return spec{
		name:  "opencode",
		dirs:  []string{".opencode/bin", ".local/bin", "bin"},
		args:  openCodeArgs,
		parse: ParseOpenCodeStream,
	}
}

// Locate is where this machine keeps opencode.
func (o OpenCode) Locate() (string, error) { return o.spec().locate() }

// Run invokes opencode in the worktree and returns what it reported.
func (o OpenCode) Run(ctx context.Context, req Request) (Result, error) {
	return o.spec().run(ctx, req)
}

// openCodeArgs builds the command line for opencode run.
//
// Three of these flags were wrong. Effort is --variant, not --effort, and
// opencode answers --effort by printing its help and exiting one, so every
// opencode phase that named an effort failed before a model saw it. The
// output was left at opencode's default, which is formatted prose, and this
// adapter piped that prose into claude's stream parser — so every opencode
// run recorded no session and no cost. --format json is what makes those two
// fields real, and ParseOpenCodeStream is what reads them.
func openCodeArgs(req Request) ([]string, error) {
	perm, err := openCodePermissionArgs(req.Permissions)
	if err != nil {
		return nil, err
	}

	args := append([]string{"run", "--format", "json"}, perm...)

	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}

	if req.Effort != "" {
		args = append(args, "--variant", req.Effort)
	}

	if req.Resume != "" {
		args = append(args, "--session", req.Resume)
	}

	return append(args, req.Prompt), nil
}

// openCodePermissionArgs turns a posture into what opencode can state, and
// refuses the postures it cannot.
//
// This adapter used to validate the permission names and then emit nothing
// about them, which is the silent widening the head of permission.go exists
// to prevent. What makes it worse here than a missing flag is what opencode
// does without one: asked to write a file with no --auto and no terminal to
// prompt at, `opencode run` wrote the file. That was run against the binary,
// not reasoned about. Headless opencode approves everything.
//
// So opencode has exactly one posture it can honestly carry, and it is the
// wide one. A phase holding repo gets --auto, which changes nothing about
// what opencode would have done and makes the argv say so, so that `ps` and
// the record agree with the behaviour.
//
// Anything narrower is refused. A read phase on opencode is a phase that can
// write, and running it would put a sentence in the record — "this phase was
// allowed to read" — that the engine had already contradicted. There is no
// flag to fix that with, so the run does not start. Every builtin flow ships
// repo on every phase, so what this refuses is a posture nothing builds yet
// and a reader would have had lied to them about.
func openCodePermissionArgs(names []string) ([]string, error) {
	if err := Permitted(names); err != nil {
		return nil, err
	}

	if !slices.Contains(names, PermissionRepo) {
		return nil, fmt.Errorf(
			"opencode cannot run a phase narrower than %s: headless `opencode run` approves every tool it is asked for, so a posture of %v would be recorded and not enforced",
			PermissionRepo, names)
	}

	return []string{"--auto"}, nil
}
