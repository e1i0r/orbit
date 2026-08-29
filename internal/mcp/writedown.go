package mcp

// What a model did through this door.
//
// This is the front door nobody is standing at. A reader driving the command
// line sees every refusal on their own terminal and a run's own account goes
// into the record; a supervising model calls this server for hours, from
// another process, and the only trace of any of it was the four kinds of
// journal entry a mutation leaves on the task it touched. Everything else —
// which tools were called, in what order, what they were given, and which of
// them said no — happened in a process nobody was watching.
//
// One place, because there is one place a tool call arrives: Session.Call.

import (
	"fmt"
	"maps"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/e1i0r/orbit/internal/logger"
)

// What one line of this log is allowed to cost.
const (
	// argChars is how much of one argument is written out before it is
	// counted instead. It is well past the longest fact this server is
	// handed — a repository path, a task id, a flow name, a permission
	// list — and well short of the prose a model writes into a prompt, a
	// note or a correction, all of which the record already holds in full.
	argChars = 200
	// refusalChars is how much of a refusal is kept. The longest of them
	// names every task in a repository somebody asked to forget.
	refusalChars = 300
)

// guard runs one tool with the two things every tool call needs and no tool
// should have to remember: it is written down, and it cannot take the
// session with it.
//
// The recover is not for a bug anyone has seen. It is for the shape of this
// process: a model drives it for hours over one pipe, and a panic in any of
// nineteen handlers would end the session for every call after it — the
// client sees the pipe close mid-conversation and has no idea which of its
// requests was the last one answered. A refusal it can read, and go on from,
// is worth more than a stack trace on a stream nobody is reading. The stack
// goes to the errors file, where the reader will look.
func guard(name string, args map[string]any, run func() CallToolResult) (res CallToolResult) {
	start := noteCall(name, args)

	defer func() {
		if p := recover(); p != nil {
			logger.Error("mcp/"+name, "panicked: %v\n%s", p, debug.Stack())
			res = refuse(fmt.Errorf("orbit_%s: this tool crashed and the session went on; the stack is in the errors log", strings.TrimPrefix(name, "orbit_")))
		}

		noteAnswer(name, start, res)
	}()

	return run()
}

// noteCall writes down a tool call as it arrives, and answers with the
// moment it did — which is what noteAnswer is handed, so the clock for a
// call lives here rather than in the middle of guard.
func noteCall(name string, args map[string]any) time.Time {
	logger.Info("mcp/"+name, "called with %s", argsLine(args))

	return time.Now()
}

// noteAnswer writes down what the tool said.
//
// A refusal is an error in the file, and it is the reason this exists: "no
// task ORB-9 on the board" is answered to the model and to nobody else, so a
// supervisor that spent an afternoon calling tools against ids it invented
// left an Orbit that looked idle and a log that said nothing.
func noteAnswer(name string, start time.Time, res CallToolResult) {
	took := time.Since(start).Round(time.Millisecond)

	if res.IsError {
		logger.Error("mcp/"+name, "refused after %s: %s", took, said(res))

		return
	}

	logger.Info("mcp/"+name, "answered after %s", took)
}

// noteFault writes down a request this server answered with a JSON-RPC error
// object rather than with a result.
//
// It is a warning and not a failure: the session goes on, and what went
// wrong is something the client sent rather than something Orbit did. A file
// that says ERROR for a thing already handled is a file with less in it.
func noteFault(what string, err error) {
	logger.Warn("mcp/transport", "%s: %v", what, err)
}

// argsLine is the arguments of one call, in a fixed order so that two calls
// of the same tool read the same way.
func argsLine(args map[string]any) string {
	if len(args) == 0 {
		return "no arguments"
	}

	parts := make([]string, 0, len(args))
	for _, k := range slices.Sorted(maps.Keys(args)) {
		parts = append(parts, k+"="+argValue(args[k]))
	}

	return strings.Join(parts, " ")
}

// argValue is one argument as the log shows it: what it says when it fits on
// a line, and how long it is when it does not.
//
// The size is still written down for the long ones, because "the prompt was
// four characters" and "the prompt was nine thousand" are different bugs and
// telling them apart is most of what this line is for.
func argValue(v any) string {
	s, isText := v.(string)
	if !isText {
		s = fmt.Sprintf("%v", v)
	}

	if n := utf8.RuneCountInString(s); n > argChars {
		return fmt.Sprintf("(%d characters)", n)
	}

	if isText {
		return strconv.Quote(s)
	}

	return s
}

// said is what a result says, for the log. Every refusal this package builds
// is one text block holding a sentence.
func said(res CallToolResult) string {
	if len(res.Content) == 0 {
		return "(nothing)"
	}

	return clip(res.Content[0].Text, refusalChars)
}
