package engine

import (
	"context"
	"fmt"
	"strconv"
)

// Claude runs the claude command line in headless mode.
//
// It asks for streaming JSON rather than plain text, because the session id
// and the cost of a run are reported in that stream and nowhere else. Those
// two fields have been on Result since the first plan and had never once
// been non-empty from a real run: the record said every phase cost nothing
// and none of them could be resumed. Without a session id there is no taking
// the keyboard, and taking the keyboard is the gesture the window is built
// around. The price is that stdout is no longer prose — ParseStream turns it
// back into the one sentence a reader wants.
type Claude struct{}

var _ Engine = Claude{}

// NewClaude returns the adapter.
func NewClaude() Claude { return Claude{} }

// Name identifies the engine in the record.
func (Claude) Name() string { return "claude" }

// CanResume is true because claudeArgs builds --resume, and because the
// stream this adapter parses is where a session id comes from at all: a run
// records one, so a person can carry it on.
func (Claude) CanResume() bool { return true }

// Models returns the models claude supports, from claudemodels.go.
func (Claude) Models() []Choice { return claudeModels }

// Efforts returns the effort levels claude supports.
func (Claude) Efforts() []Choice {
	return []Choice{
		{ID: "", Label: "default"},
		{ID: "low", Label: "low"},
		{ID: "medium", Label: "medium"},
		{ID: "high", Label: "high"},
		{ID: "xhigh", Label: "xhigh"},
		{ID: "max", Label: "max"},
	}
}

// CanThink is true because claude supports extended thinking mode.
func (Claude) CanThink() bool { return true }

// spec is how claude is driven: where it lives, what its argv looks like,
// what thinking budget it is given, and the stream shape it prints.
func (Claude) spec() spec {
	return spec{
		name:  "claude",
		dirs:  []string{".local/bin", ".claude/local", "bin"},
		args:  claudeArgs,
		env:   claudeEnv,
		parse: ParseStreamWithCallback,
	}
}

// Locate is where this machine keeps claude.
func (c Claude) Locate() (string, error) { return c.spec().locate() }

// Run invokes claude in the worktree and returns what it reported.
//
// The engine's working directory is the task's worktree, which lives inside
// the Orbit state root by design. The record that is the product's whole
// trust model, and the credentials file the design puts in the same root,
// are therefore reachable from here by relative path. The control is not the
// layout, which buys nothing against a process running as the same user: it
// is that no engine is ever handed a directory permission at or above the
// state root. No --add-dir, and no equivalent on any engine added later, at
// store.Root() or above it. root_test.go is where that stopped being a
// comment and became a rule. That rule contains the tools that name a path;
// it does not contain a shell, and what repo really grants is written out in
// claudePermissionArgs.
func (c Claude) Run(ctx context.Context, req Request) (Result, error) {
	return c.spec().run(ctx, req)
}

// claudeEnv is the thinking budget, as the one environment variable claude
// reads it from.
//
// A budget is a number of tokens, and anything else was passed through to
// the binary as if it were one: MAX_THINKING_TOKENS=lots is not a budget,
// and claude's own answer to it is not something this adapter should be
// choosing on a reader's behalf. A word this switch does not know is left
// alone, which lands on claude's adaptive default — the same place an empty
// thinking dial lands, and the only other honest option.
func claudeEnv(req Request) []string {
	switch req.Thinking {
	case "":
		return nil
	case "off", "none", "0":
		return []string{"MAX_THINKING_TOKENS=0"}
	case "adaptive", "on":
		// Unset leaves the adaptive default.
		return nil
	}

	if n, err := strconv.Atoi(req.Thinking); err == nil && n > 0 {
		return []string{"MAX_THINKING_TOKENS=" + req.Thinking}
	}

	return nil
}

// maxStream is how much of an engine's stdout this adapter will hold.
//
// The stream is JSON lines now rather than prose: every assistant turn and
// every tool result arrives on it, and a tool result is content the engine
// did not write. maxStreamLine bounds one line at four mebibytes with an
// argument of its own, and until this bound the total was held by nothing at
// all, so a long run — or a hostile one — was a way to grow the Orbit
// process until the machine complained. Sixteen mebibytes is four whole
// lines at the line limit: room for the terminal result object and the
// messages around it, and a ceiling a real run has to work at to reach.
const maxStream = 16 << 20

// maxStderr is the same bound on the other pipe, which was unbounded for the
// same reason. A mebibyte, because stderr carries a message that names a
// failure rather than a stream, and it is bounded by the same type so that
// "the buffers are capped" is true of both and not of one.
const maxStderr = 1 << 20

// boundedBuffer keeps the last max bytes written to it and counts what it
// dropped.
//
// internal/task's captured() is the precedent — truncation that announces
// itself, because silent loss writes something untrue into the record — and
// this keeps its principle while reversing its end. captured keeps the head,
// since a phase's answer starts at the top. The answer here is the terminal
// result object, which is the last line of the stream, so keeping the head
// would throw away the only line ParseStream reads and turn a long run into
// "the stream ended with no result object". Keeping the tail costs a partial
// first line, which ParseStream skips along with everything else that is not
// a result object.
type boundedBuffer struct {
	max     int
	buf     []byte
	dropped int
}

// Write never fails and never grows the buffer past max, so a run cannot be
// lost to a write error on its way to being reported.
func (b *boundedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if n >= b.max {
		b.dropped += len(b.buf) + n - b.max
		b.buf = append(b.buf[:0], p[n-b.max:]...)

		return n, nil
	}

	if over := len(b.buf) + n - b.max; over > 0 {
		b.dropped += over
		// Slide the tail down in place rather than reslicing: a reslice
		// walks the start forward through an array that only ever grows,
		// which is the leak this type exists to stop.
		b.buf = b.buf[:copy(b.buf, b.buf[over:])]
	}

	b.buf = append(b.buf, p...)

	return n, nil
}

// Bytes is what survived the cap, oldest dropped first.
func (b *boundedBuffer) Bytes() []byte { return b.buf }

// String is what survived the cap, as text.
func (b *boundedBuffer) String() string { return string(b.buf) }

// noteDropped says on the phase's own text that the stream it was read from
// was cut. A run that hits the cap has to admit it: an answer that lost its
// stream and reads exactly like one that did not is the record stating
// something it cannot know.
func noteDropped(text string, dropped int) string {
	if dropped <= 0 {
		return text
	}

	note := fmt.Sprintf("…[the engine's stream ran past %d bytes; the %d earliest were dropped]", maxStream, dropped)
	if text == "" {
		return note
	}

	return text + "\n" + note
}

// claudeArgs is separate from Run so the command line can be tested without
// a claude binary present and without spending anything.
//
// It returns an error because a posture nobody defined must not be able to
// become a command line at all. That is a stronger guarantee than checking
// in Run: there is no path through this package that builds argv for a
// permission the vocabulary does not hold, so the tests that assert what
// never appears on a command line have something total to assert against.
func claudeArgs(req Request) ([]string, error) {
	perms, err := claudePermissionArgs(req.Permissions)
	if err != nil {
		return nil, err
	}
	// --verbose is not a preference. claude refuses --output-format
	// stream-json under -p without it, so it is part of the same decision
	// as the format itself.
	args := []string{"-p", req.Prompt, "--output-format", "stream-json", "--verbose"}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}

	if req.Effort != "" {
		args = append(args, "--effort", req.Effort)
	}

	if req.Resume != "" {
		args = append(args, "--resume", req.Resume)
	}

	return append(args, perms...), nil
}
