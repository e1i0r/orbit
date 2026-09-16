package main

// A stand-in for a coding agent, for the flow tests.
//
// It is a real binary, put on PATH under the name a flow asks for, so the
// thing under test is the whole of Orbit: the command line resolves an
// engine, spawns it, reads its stream, and writes down what came back. What
// is faked is the model and only the model.
//
// It does real work. A turn writes real files into the worktree it was
// started in, so the diff budget, the dependency gate, a loop's checks and
// the story all read something that is actually there. A stand-in that only
// printed would leave every one of those reading an empty change.
//
// What it says on stdout is the stream-json claude speaks, because that is
// what internal/engine parses. Nothing here invents a field: the shapes are
// the ones in internal/engine/testdata.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// script is what this engine does, phase by phase.
//
// A phase holds one turn per time it is called, and the last one repeats: a
// loop that goes round more times than the script describes keeps doing what
// it did last rather than failing on a missing entry.
type script struct {
	Phases map[string][]turn `json:"phases"`
	// Engines is what one engine does, whatever phase it is called for, and
	// it outranks Phases.
	//
	// The relay needs it and a script keyed only by phase cannot say it:
	// the scenario is one engine running out of allowance and another
	// finishing the same phase, which is two behaviours for one phase name.
	// The stand-in knows which it is because it is installed under each
	// engine's name and reads its own.
	Engines map[string][]turn `json:"engines"`
}

// turn is one call: what it writes, what it says, and what it cost.
type turn struct {
	Write    map[string]string `json:"write"`
	Remove   []string          `json:"remove"`
	Say      string            `json:"say"`
	Thoughts []string          `json:"thoughts"`
	Cost     float64           `json:"cost"`
	// Exit is what to leave with, for the scenario where an engine fails.
	Exit int `json:"exit"`
	// RanOut is what a provider prints when the allowance is gone, and
	// saying it is what makes this turn a run that ran out rather than one
	// that broke.
	//
	// It goes to stderr and leaves non-zero, which is how a real one arrives:
	// exec gives a program one way to say it stopped, and the provider's own
	// words are above it. internal/engine reads both halves, so a scenario
	// written here is read the same way a provider's refusal is.
	RanOut string `json:"ranOut"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fakeengine:", err)
		os.Exit(1)
	}
}

func run() error {
	prompt := promptOf(os.Args[1:])
	if prompt == "" {
		return fmt.Errorf("no prompt on the command line: %v", os.Args[1:])
	}

	s, err := load(os.Getenv(scriptEnv))
	if err != nil {
		return err
	}

	phase := phaseOf(prompt)

	// The engine's own script first. Which engine this is, is the name it
	// was installed under and called by.
	me := filepath.Base(os.Args[0])
	counted := phase

	turns := s.Engines[me]
	if len(turns) > 0 {
		counted = me
	} else {
		turns = s.Phases[phase]
	}

	if len(turns) == 0 {
		return fmt.Errorf("the script says nothing about the phase %q or the engine %q", phase, me)
	}

	n, err := nth(counted)
	if err != nil {
		return err
	}

	t := turns[min(n, len(turns)-1)]

	// Before anything is written or said. An engine the provider refused
	// did no work and printed no stream: it was turned away on the request,
	// and a stand-in that wrote files first would be describing something
	// that never happens.
	if t.RanOut != "" {
		fmt.Fprintln(os.Stderr, t.RanOut)
		os.Exit(1)
	}

	if err := apply(t); err != nil {
		return err
	}

	if err := say(t, phase); err != nil {
		return err
	}

	if t.Exit != 0 {
		os.Exit(t.Exit)
	}

	return nil
}

// scriptEnv names the file describing what to do, and stateEnv the directory
// where the count of turns per phase is kept — one file per phase, because
// each turn is a process of its own and nothing else survives between them.
const (
	scriptEnv = "ORBIT_FAKE_SCRIPT"
	stateEnv  = "ORBIT_FAKE_STATE"
)

func load(path string) (script, error) {
	var s script

	if path == "" {
		return s, fmt.Errorf("%s is not set", scriptEnv)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		return s, fmt.Errorf("read the script: %w", err)
	}

	if err := json.Unmarshal(body, &s); err != nil {
		return s, fmt.Errorf("read the script %q: %w", path, err)
	}

	return s, nil
}

// promptOf is the prompt off the command line.
//
// The four engines take it two ways and this stands in for all four, so it
// reads both: claude and agy name it with a flag, codex and opencode put it
// last with no flag at all. Reading only the flag was enough while nothing
// ever ran a second engine — and the first thing that did, the relay, got
// "no prompt on the command line" from the engine that took the task on.
func promptOf(args []string) string {
	for i, a := range args {
		if (a == "-p" || a == "--print") && i+1 < len(args) {
			return args[i+1]
		}
	}

	// The last argument, which is where the ones with no flag put it. A
	// flag's own value is never last for these two — every argument they
	// pass before the prompt is a switch or a switch with a value — so a
	// command line ending in something is a command line ending in the
	// prompt.
	if len(args) > 0 {
		return args[len(args)-1]
	}

	return ""
}

// named is the phase's own name, which the prompt carries in backticks under
// its Phase heading — see internal/task's where().
var named = regexp.MustCompile("## Phase\n\n`([^`]+)`")

func phaseOf(prompt string) string {
	if m := named.FindStringSubmatch(prompt); len(m) == 2 {
		return m[1]
	}

	return ""
}

// nth is how many times this phase has been run before, and counts this one.
func nth(phase string) (int, error) {
	dir := os.Getenv(stateEnv)
	if dir == "" {
		return 0, nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, fmt.Errorf("make the state directory: %w", err)
	}

	path := filepath.Join(dir, strings.ReplaceAll(phase, "/", "_"))

	seen := 0

	if body, err := os.ReadFile(path); err == nil {
		// A count that will not parse is a count nobody wrote: the first
		// turn is the honest answer, and failing here would fail a run over
		// this stand-in's own bookkeeping.
		if n, convErr := strconv.Atoi(strings.TrimSpace(string(body))); convErr == nil {
			seen = n
		}
	}

	if err := os.WriteFile(path, []byte(strconv.Itoa(seen+1)), 0o600); err != nil {
		return 0, fmt.Errorf("write the state: %w", err)
	}

	return seen, nil
}

// apply does the turn's real work in the directory the engine was started
// in, which is the task's worktree.
func apply(t turn) error {
	for path, body := range t.Write {
		if dir := filepath.Dir(path); dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("make %q: %w", dir, err)
			}
		}

		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			return fmt.Errorf("write %q: %w", path, err)
		}
	}

	for _, path := range t.Remove {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %q: %w", path, err)
		}
	}

	return nil
}
