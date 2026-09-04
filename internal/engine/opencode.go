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
// The catalogue itself is in opencodemodels.go, where its length is
// somebody else's business rather than this file's.
func (OpenCode) Models() []Choice { return opencodeModels }

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
// Validating the permission names and then emitting nothing about them is
// the silent widening the head of permission.go exists to prevent. What
// makes it worse here than a missing flag is what opencode does without one: asked to write a file with no --auto and no terminal to
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
