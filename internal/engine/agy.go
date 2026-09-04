package engine

// The Antigravity command line, which calls itself agy.

import (
	"context"
	"fmt"
	"slices"
	"time"
)

// Agy runs Antigravity's command line in headless mode.
type Agy struct{}

var _ Engine = Agy{}

// NewAgy returns the Antigravity CLI adapter.
func NewAgy() Agy { return Agy{} }

// Name identifies the engine in the record.
//
// It is the program's name and not the product's. A record's engine name is
// also what the window runs when the reader asks for a session, so calling
// this antigravity would be a name nothing on the machine answers to.
func (Agy) Name() string { return "agy" }

// CanResume reports whether agy can carry on a previous conversation. It
// can: --conversation takes the id every one of its streams opens with.
func (Agy) CanResume() bool { return true }

// Models returns the models agy supports.
//
// It is what `agy models` prints, IDs on the left and its own labels on the
// right, and it is a written-down copy of a catalogue that moves for the
// reason opencode's is: reading it at run time means shelling out to agy
// before a dial can be drawn, on a machine where agy may not be installed.
// Refresh it with `agy models`.
//
// Gemini names its reasoning inside the model — a family appears three
// times, high, medium and low — while --effort says the same thing beside
// it. Both are offered as they are printed rather than folded together,
// because what a dial shows and what the binary is told have to be the same
// string.
func (Agy) Models() []Choice {
	return []Choice{
		{ID: "", Label: "default"},
		{ID: "gemini-3.8-flash-high", Label: "Gemini 3.8 Flash (High)"},
		{ID: "gemini-3.8-flash-medium", Label: "Gemini 3.8 Flash (Medium)"},
		{ID: "gemini-3.8-flash-low", Label: "Gemini 3.8 Flash (Low)"},
		{ID: "gemini-3.7-flash-high", Label: "Gemini 3.7 Flash (High)"},
		{ID: "gemini-3.7-flash-medium", Label: "Gemini 3.7 Flash (Medium)"},
		{ID: "gemini-3.7-flash-low", Label: "Gemini 3.7 Flash (Low)"},
		{ID: "gemini-3.6-flash-high", Label: "Gemini 3.6 Flash (High)"},
		{ID: "gemini-3.6-flash-medium", Label: "Gemini 3.6 Flash (Medium)"},
		{ID: "gemini-3.6-flash-low", Label: "Gemini 3.6 Flash (Low)"},
		{ID: "gemini-3.1-pro-high", Label: "Gemini 3.1 Pro (High)"},
		{ID: "gemini-3.1-pro-low", Label: "Gemini 3.1 Pro (Low)"},
		{ID: "claude-sonnet-4-6", Label: "Claude Sonnet 4.6 (Thinking)"},
		{ID: "claude-opus-4-6-thinking", Label: "Claude Opus 4.6 (Thinking)"},
		{ID: "gpt-oss-120b-medium", Label: "GPT-OSS 120B (Medium)"},
	}
}

// Efforts returns the effort choices agy supports: the three its own --effort
// names, and nothing invented beside them.
func (Agy) Efforts() []Choice {
	return []Choice{
		{ID: "", Label: "default"},
		{ID: "low", Label: "low"},
		{ID: "medium", Label: "medium"},
		{ID: "high", Label: "high"},
	}
}

// CanThink reports whether agy supports thinking mode. It has no flag for
// one: its models think — the stream counts thinking_tokens on every step —
// and how much is asked for by effort, so a thinking dial here would be a
// second name for the dial beside it.
func (Agy) CanThink() bool { return false }

// Transcript is nothing, and says so rather than guessing.
//
// agy keeps each conversation in a SQLite database of its own, under
// ~/.gemini/antigravity-cli/conversations, with the directory it ran in in
// conversation_summaries.db beside them — so which conversation belongs to a
// worktree is answerable. What is not is what was said: every step in those
// databases is a protobuf blob and no schema for it ships with the program.
// Walking that wire format field by field would be a guess written into a
// task's record as if it were an account of the session.
//
// The interface's answer for an engine nobody has mapped is nothing: the
// session is left unread, and the rest of what agy does is recorded as any
// other engine's is.
func (Agy) Transcript(string, time.Time) ([]Turn, error) { return nil, nil }

// spec is how agy is driven.
//
// The installer puts the binary in ~/.local/bin and adds it to a shell
// profile, so the dirs carry it for the reason opencode's carry .opencode/bin:
// a PATH exported before the install has no agy in it.
func (Agy) spec() spec {
	return spec{
		name:  "agy",
		dirs:  []string{".local/bin", "bin"},
		args:  agyArgs,
		parse: ParseAgyStream,
	}
}

// Locate is where this machine keeps agy.
func (a Agy) Locate() (string, error) { return a.spec().locate() }

// Run invokes agy in the worktree and returns what it reported.
func (a Agy) Run(ctx context.Context, req Request) (Result, error) {
	return a.spec().run(ctx, req)
}

// agyArgs builds the command line for a headless agy run.
//
// --print is the headless mode and takes the prompt as its value. The two
// spellings of it are worth keeping straight, because they are opposites:
// --prompt is an alias for --print and runs without a terminal, while
// --prompt-interactive opens the session and types the first line into it.
// The window's `c` wants the second one, and internal/cli names it there.
//
// stream-json is what makes the session id, the tool calls and the token
// counts real; agy's default prints the answer and nothing else.
func agyArgs(req Request) ([]string, error) {
	perms, err := agyPermissionArgs(req.Permissions)
	if err != nil {
		return nil, err
	}

	args := []string{"--print", req.Prompt, "--output-format", "stream-json"}

	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}

	if req.Effort != "" {
		args = append(args, "--effort", req.Effort)
	}

	if req.Resume != "" {
		args = append(args, "--conversation", req.Resume)
	}

	return append(args, perms...), nil
}

// agyPermissionArgs turns a posture into agy's own flags.
//
// agy states a posture as a mode a person answers at the prompt — plan,
// accept-edits, or the review it defaults to — and a headless run has nobody
// to ask. It says so itself, in prose, after auto-denying the tool: "a tool
// required the "command" permission that headless mode cannot prompt for, so
// it was auto-denied ... re-run with --dangerously-skip-permissions". That
// was watched happening, not read off a help page: a run asked to list a
// directory was denied twice and finished having done nothing, at the price
// of the tokens it spent trying.
//
// So repo is the only posture this engine can state, and anything narrower
// is refused for the reason opencode's is: it would be a posture written
// into the record that the run never had, and a phase that burns its budget
// being told no.
//
// plan is the narrower posture agy plausibly has, and it is not written down
// here until somebody has watched a headless plan run finish rather than
// stall. network is not stated separately: agy grants tools by name in its
// own settings file and has no flag for the network, so a run allowed to
// write is allowed to reach out, and this says as much rather than implying
// a boundary that is not there.
func agyPermissionArgs(names []string) ([]string, error) {
	if err := Permitted(names); err != nil {
		return nil, err
	}

	if !slices.Contains(names, PermissionRepo) {
		return nil, fmt.Errorf(
			"agy cannot run a phase narrower than %s: its postures are prompts a person answers, and a headless run auto-denies them, so a posture of %v would be recorded and not enforced",
			PermissionRepo, names)
	}

	return []string{"--dangerously-skip-permissions"}, nil
}
