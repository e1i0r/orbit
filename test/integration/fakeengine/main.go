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
	turns := s.Phases[phase]

	if len(turns) == 0 {
		return fmt.Errorf("the script says nothing about the phase %q", phase)
	}

	n, err := nth(phase)
	if err != nil {
		return err
	}

	t := turns[min(n, len(turns)-1)]

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

// promptOf is the prompt off the command line, which every engine Orbit
// drives is given as one argument after -p.
func promptOf(args []string) string {
	for i, a := range args {
		if (a == "-p" || a == "--print") && i+1 < len(args) {
			return args[i+1]
		}
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
