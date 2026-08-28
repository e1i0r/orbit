package engine

import (
	"fmt"
	"slices"
	"strings"
)

// The closed vocabulary a phase may use to say what it is allowed to touch.
//
// Three names, and no fourth without a change here and a reviewer. They are
// spelled out again in internal/flow, which is where a flow file declares
// one, and the duplication is deliberate: neither package may import the
// other — internal/engine and internal/flow both have an empty row in the
// layer table — and three string constants are a cheaper price than a
// shared package that would let the window reach an engine. The two copies
// answer different questions. flow.Validate asks whether a file wrote down
// a name that exists; Permitted asks whether the engine about to be started
// can honour it. A name could pass the first and fail the second, which is
// exactly the day the two checks earn the duplication.
const (
	// PermissionRead may read the worktree and may run no command that
	// writes.
	PermissionRead = "read"
	// PermissionRepo may read and write inside the worktree and run the
	// repository's own commands.
	PermissionRepo = "repo"
	// PermissionNetwork may reach the network.
	PermissionNetwork = "network"
)

// The tools each name buys on claude's command line.
//
// These are lists rather than a mode because a mode is a switch and a list
// is a statement. claude offers four permission modes and two of them are
// grants rather than postures: bypassPermissions and its command-line twin
// --dangerously-skip-permissions turn the question off instead of answering
// it, and acceptEdits auto-approves edits while leaving the commands those
// edits need to prompt. None of the three is reachable from any input here,
// and permission_test.go asserts it for every posture rather than for a
// chosen example.
//
// acceptEdits is refused for what it did in the program this replaces, and
// that lesson is worth stating precisely, because the loose version of it
// flatters this file. The mode is not what burned the money. What burned it
// was a headless run asked to do what its posture refused: every attempt
// denied, the same edit tried again for eight minutes, because a denial in a
// headless run is not a prompt anybody can answer. acceptEdits is one way to
// arrive there and not the only one — a phase told to implement under read,
// or under the posture that asks for nothing, is denied on every edit just
// the same, and nothing below stops it trying again. This argv bounds
// neither turns nor cost. Half the lesson is acted on here, in the mode; the
// other half is a ceiling on a run, which belongs to the task that gives
// Orbit a budget rather than to this one, and until that task lands the
// bound is a human watching the window.
var (
	readTools    = []string{"Read", "Glob", "Grep"}
	writeTools   = []string{"Edit", "Write", "Bash"}
	networkTools = []string{"WebFetch", "WebSearch"}
)

// toolOrder is the order tools are written in, so that the same posture
// always produces the same command line whatever order the flow file listed
// its names in. A command line that varies with the spelling of a file is a
// command line nobody can compare between two runs.
var toolOrder = slices.Concat(readTools, writeTools, networkTools)

// noTools is what the posture that asks for nothing writes in its
// --allowedTools. It is a name no tool has; the argument for stating an
// empty grant rather than omitting the flag is in claudePermissionArgs.
const noTools = "none"

// Permitted reports whether the engines in this package can honour a
// posture.
//
// It is not the same check as flow.Validate, and neither replaces the other.
// flow.Validate reads a file and refuses a name that does not exist, which
// catches a typo at the moment the flow is loaded — before a worktree, a
// process or a bill. Permitted is asked later, by the adapter that is about
// to build a command line, and its answer is about this engine: an engine
// that cannot state a posture must refuse the run rather than start one
// under a posture it silently widened.
//
// An empty list is permitted and means something. It is the phase that asks
// for nothing, and it comes out as the most restrictive command line the
// engine can state — not as an absence of flags, which would leave the
// binary's own default in charge of a process whose working directory sits
// inside the state root.
func Permitted(names []string) error {
	for _, n := range names {
		switch n {
		case PermissionRead, PermissionRepo, PermissionNetwork:
		default:
			return fmt.Errorf("no engine here knows the permission %q; the vocabulary is %s, %s and %s", n, PermissionRead, PermissionRepo, PermissionNetwork)
		}
	}

	return nil
}

// claudePermissionArgs turns a posture into claude's own flags.
//
// The shape is always the same — a stated mode, and an explicit list of what
// may be used — because the posture has to be readable off the command line
// of a running process. `ps` is the last resort when a run is doing
// something surprising, and an engine started with no permission flags at
// all tells that reader nothing.
//
// The mode is plan unless the posture may write, because plan is claude's
// own name for look-but-do-not-touch and it is the strictest mode the binary
// names. A posture with repo in it has to leave plan behind — plan mode
// refuses to edit whatever the allowed list says — and lands on default.
// Naming default on the command line rather than omitting the flag is the
// point: the argv says which mode, so a later change of the binary's own
// default cannot silently move the posture. What turns an un-allowed tool
// into a silent denial rather than a prompt is not default mode, it is -p:
// a headless run has nobody to ask, so it refuses instead of waiting.
// claudeArgs passes -p unconditionally, which is why no posture built here
// can hang on a question.
//
// The allowed list is a floor, not a ceiling. --allowedTools is additive to
// the settings claude reads on its own — the user's ~/.claude/settings.json,
// the project's, and the worktree's own .claude/settings.json — so this argv
// states the least a phase may do and has no way to state the most. On a
// machine whose settings grant Bash, a read phase has Bash, and nothing here
// or in the record says so. Pinning the settings claude reads, instead of
// adding to them, is the fix; it changes how every engine is started and
// what a worktree is allowed to carry, so it is a task of its own and not
// something to smuggle in behind this comment.
//
// Additivity has a sharper edge across phases. A repo phase may write
// .claude/settings.local.json inside the very worktree it is allowed to
// write, and the next phase's fresh claude process reads that file as an
// allow-list. No argv can prevent it, because the file is inside the grant
// the posture gave. It is a known limit of this design, named here rather
// than left for somebody to find.
//
// What this cannot express is the second half of repo's definition. "Run the
// repository's own commands" has no equivalent in claude's grammar, which
// grants Bash whole or not at all, so repo grants Bash whole. What the
// working directory and the rule in root_test.go contain is the tools that
// name a path: Read, Edit and Write reach the task's worktree and whatever
// --add-dir names, no engine is ever handed a directory grant at or above
// the state root, and nothing here names a directory at all. They do not
// contain Bash. A granted shell is scoped by neither cwd nor --add-dir nor
// that rule — repo can read the developer's home directory and the keys in
// it, write anywhere the user can write, and reach the network whether or
// not the posture named network. Every builtin flow ships repo on every
// phase, so a shell with the user's own reach is the posture of every Orbit
// run today, and that is the sentence a reviewer is signing. It is stated
// here rather than papered over, because a posture whose limits are
// undocumented is the posture nobody can review.
//
// The posture that asks for nothing still states its tool half. Leaving
// --allowedTools off would hand the tool question to the settings files and
// the binary's own default, which is exactly the failure the mode half of
// this file refuses; an empty flag value risks being read as no flag at all.
// So the flag is written with noTools, a name nothing matches: non-empty to
// the parser, empty in effect, and a reader of `ps` sees a phase that asked
// for nothing instead of an absence they have to interpret. Given the floor
// above, it narrows nothing the settings can widen — what it buys is that
// the argv says what the phase asked for. permission_test.go holds the
// sentinel to naming no tool this package knows.
func claudePermissionArgs(names []string) ([]string, error) {
	if err := Permitted(names); err != nil {
		return nil, err
	}

	mode := "plan"
	allowed := map[string]bool{}

	for _, n := range names {
		switch n {
		case PermissionRead:
			allow(allowed, readTools)
		case PermissionRepo:
			// repo implies read: a posture that may rewrite a file it
			// cannot open is not a posture, it is a bug report.
			mode = "default"

			allow(allowed, readTools)
			allow(allowed, writeTools)
		case PermissionNetwork:
			allow(allowed, networkTools)
		}
	}

	args := []string{"--permission-mode", mode}

	var tools []string

	for _, t := range toolOrder {
		if allowed[t] {
			tools = append(tools, t)
		}
	}

	if len(tools) == 0 {
		tools = []string{noTools}
	}

	args = append(args, "--allowedTools", strings.Join(tools, ","))

	return args, nil
}

// allow marks a group of tools as granted.
func allow(set map[string]bool, tools []string) {
	for _, t := range tools {
		set[t] = true
	}
}
