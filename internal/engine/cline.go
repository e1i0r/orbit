package engine

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// Cline runs the cline command line in headless mode.
type Cline struct{}

var _ Engine = Cline{}

// NewCline returns the Cline CLI adapter.
func NewCline() Cline { return Cline{} }

// Name identifies the engine in the record.
func (Cline) Name() string { return "cline" }

// CanResume is false, and it is cline's answer rather than this adapter's.
//
// cline resumes a session with --id, and --id opens its terminal interface
// and drops the prompt it was given: with --json the pair is refused
// outright. A headless run cannot carry on a session, so a phase cannot be
// continued and the keyboard cannot be taken; saying true here would offer
// both and fail at the key.
func (Cline) CanResume() bool { return false }

// Models returns the models cline supports: see cline_models.go.
func (Cline) Models() []Choice { return clineModels }

// Efforts returns the effort choices cline supports.
//
// cline has no effort of its own. What it has is --thinking, with five
// levels, which is how hard the model is asked to reason, and that is what
// an effort is on every other engine here. none is left out: an effort
// left at default already leaves the flag off.
func (Cline) Efforts() []Choice {
	return []Choice{
		{ID: "", Label: "default"},
		{ID: "low", Label: "low"},
		{ID: "medium", Label: "medium"},
		{ID: "high", Label: "high"},
		{ID: "xhigh", Label: "xhigh"},
	}
}

// CanThink is false: the thinking level is the effort, above, and a second
// dial for the same flag would be two answers to one question.
func (Cline) CanThink() bool { return false }

// spec is how cline is driven.
//
// npm installs cline where npm's global prefix is, which PATH usually has;
// .local/bin and bin are the prefixes people set by hand.
func (Cline) spec() spec {
	return spec{
		name:  "cline",
		dirs:  []string{".local/bin", "bin", ".npm-global/bin"},
		args:  clineArgs,
		env:   clineEnv,
		parse: ParseClineStream,
	}
}

// Locate is where this machine keeps cline.
func (c Cline) Locate() (string, error) { return c.spec().locate() }

// Run invokes cline in the worktree and returns what it reported.
func (c Cline) Run(ctx context.Context, req Request) (Result, error) {
	return c.spec().run(ctx, req)
}

// clineArgs builds the command line for a headless cline run.
//
// --json is its event stream, and -v is not decoration: the session id is
// in the run_start line and nowhere else, and cline writes that line only
// when it is asked to be verbose.
//
// The model is provider-qualified the way opencode's are, cline-pass/glm-5.2,
// and the provider in front is handed to -P as well: -m alone goes to
// whichever provider cline last used, and a cline-pass model sent to another
// provider is a model it has never heard of.
func clineArgs(req Request) ([]string, error) {
	perm, err := clinePermissionArgs(req.Permissions)
	if err != nil {
		return nil, err
	}

	args := append([]string{"--json", "-v"}, perm...)

	if req.Dir != "" {
		args = append(args, "--cwd", req.Dir)
	}

	if req.Model != "" {
		if provider, _, ok := strings.Cut(req.Model, "/"); ok && provider == clinePass {
			args = append(args, "-P", provider)
		}

		args = append(args, "-m", req.Model)
	}

	if req.Effort != "" {
		args = append(args, "--thinking", req.Effort)
	}

	return append(args, req.Prompt), nil
}

// clinePass is the provider cline's own subscription answers under.
const clinePass = "cline-pass"

// clineEnv keeps the run in this process.
//
// Left to itself, cline hands a run to a hub daemon it keeps in the
// background when one is up, and the run then happens in that daemon's
// environment rather than this one — not in the worktree's process group,
// not stopped when the phase is, not under the priority orbit set. local is
// cline's own word for "here".
func clineEnv(Request) []string {
	return []string{"CLINE_SESSION_BACKEND_MODE=local"}
}

// clinePermissionArgs turns a posture into cline's flags, and refuses the
// ones it cannot state.
//
// --yolo is the posture a headless run can hold: every tool approved, and
// the run ending when the turn does. Anything narrower is a prompt, and
// cline answers a prompt it cannot show by denying the tool and carrying
// on — "Tool requires approval in a TTY session" — so a read posture would
// be recorded while the run spent its budget being told no. plan mode is the
// narrower posture cline plausibly has, and it keeps its shell, so it is not
// written down here as read. network is not stated separately: under --yolo
// cline turns its web fetch off, and a shell can still reach out, so there
// is no boundary here to name.
func clinePermissionArgs(names []string) ([]string, error) {
	if err := Permitted(names); err != nil {
		return nil, err
	}

	if !slices.Contains(names, PermissionRepo) {
		return nil, fmt.Errorf(
			"cline cannot run a phase narrower than %s: a headless run denies every tool it would have to ask about, so a posture of %v would be recorded and not enforced",
			PermissionRepo, names)
	}

	return []string{"--yolo"}, nil
}

// RanOut is whether this run stopped because the allowance is gone.
//
// cline is a command line over whichever provider it was pointed at, so a
// provider's words are read the way every engine's are; and cline's own
// subscription says it in words of its own, which are read as well.
func (Cline) RanOut(out Result, err error) bool {
	if ranOut(out, err) {
		return true
	}

	said := strings.ToLower(out.Output)
	if err != nil {
		said += "\n" + strings.ToLower(err.Error())
	}

	for _, mark := range clineSpentMarks {
		if strings.Contains(said, mark) {
			return true
		}
	}

	return false
}

// clineSpentMarks are the sentences cline writes when its own allowance is
// gone, lowered.
var clineSpentMarks = []string{
	"clinepass limit reached",
	"daily free model limit reached",
	"free limit reached on model",
}
