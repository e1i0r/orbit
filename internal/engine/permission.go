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
// edits need to prompt — which in a headless run is a refusal the model
// cannot see past. The program this replaces ran exactly that combination
// and spent $5.04 in eight minutes on twenty-one identical refusals. None of
// the three is reachable from any input here, and permission_test.go asserts
// it for every posture rather than for a chosen example.
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
// refuses to edit whatever the allowed list says — and lands on default,
// where the allowed list is the whole of the grant and everything absent
// from it is refused without a prompt. Naming default on the command line
// rather than omitting the flag is the point: the argv says which mode, so a
// later change of the binary's own default cannot silently move the posture.
//
// What this cannot express is the second half of repo's definition. "Run the
// repository's own commands" has no equivalent in claude's grammar, which
// grants Bash whole or not at all, so repo grants Bash whole. The
// containment for that is not the tool list — it is the working directory,
// which is the task's worktree, and the rule in root_test.go that no engine
// is ever handed a directory grant at or above the state root. The same
// bluntness makes repo and network overlap, since a shell can reach the
// network: network adds the web tools and takes nothing away from Bash. Both
// are stated here rather than papered over, because a posture whose limits
// are undocumented is the posture nobody can review.
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
	if len(tools) > 0 {
		args = append(args, "--allowedTools", strings.Join(tools, ","))
	}
	return args, nil
}

// allow marks a group of tools as granted.
func allow(set map[string]bool, tools []string) {
	for _, t := range tools {
		set[t] = true
	}
}
