package cli

// The engines this build can run: the one table of them, and everything the
// window asks of it.
//
// The table used to be written out three times — twice as a map of
// constructors, in `orbit run` and in the window, and once more as a list of
// bare names for the settings screen — so an engine added to one was an
// engine the others did not have: startable from the command line, and
// neither offered on the settings screen nor recognised when the window went
// looking for it. It is one table here, and the questions asked of it live
// beside it.
//
// The ports are here rather than in ports.go for the same reason they are in
// this package at all: they take a *store.Store, which internal/ui cannot
// name. What they have in common with each other is the table above.

import (
	"context"
	"maps"
	"os/exec"
	"slices"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/task"
	"github.com/e1i0r/orbit/internal/ui"
	"github.com/e1i0r/orbit/internal/view"
)

// newEngines is every engine this build can run, by the name a record
// carries.
//
// A map and not a default, because the record already names its engine: a
// task run by something this build does not have has to be answered by name
// rather than by assumption.
func newEngines() map[string]engine.Engine {
	return map[string]engine.Engine{
		"claude":   engine.NewClaude(),
		"codex":    engine.NewCodex(),
		"opencode": engine.NewOpenCode(),
	}
}

// engineNames is the same table in the order a list of them is shown.
//
// Sorted, because a map has no order and a settings screen whose rows move
// between two openings is a screen nobody can learn the shape of.
func engineNames(engines map[string]engine.Engine) []string {
	return slices.Sorted(maps.Keys(engines))
}

// takePort builds the interactive session the window suspends itself for,
// and runs nothing.
//
// A task that has never run has no engine and no session, and that is not a
// failure: it is answered with no command and no error, which is what makes
// the window say so in the reader's own language rather than in this
// package's English. An engine this build does not have is a different
// thing, and it is named — that one is a fact about the binary, and a reader
// who is told which engine is missing can do something about it.
func takePort(r *board.Reader, engines map[string]engine.Engine) func(view.Task) (*exec.Cmd, error) {
	return func(t view.Task) (*exec.Cmd, error) {
		if t.Engine == "" {
			return nil, nil
		}

		eng, ok := engines[t.Engine]
		if !ok {
			return nil, &unknownEngineError{Name: t.Engine, ID: t.ID}
		}

		session, err := lastSession(r, t)
		if err != nil {
			return nil, err
		}

		dir, err := r.Worktree(t.RepoPath, t.ID)
		if err != nil {
			return nil, err
		}
		// A session of "" is takeCommand's rule and not this function's:
		// it answers with no command, and the window has one sentence for
		// a task there is nothing to carry on for.
		return takeCommand(eng, session, dir)
	}
}

// unknownEngineError is a record naming an engine this build cannot run.
//
// It is a type rather than fmt.Errorf because the window puts it in the
// activity band verbatim, and a sentence a reader is expected to act on is
// worth being able to test by name.
type unknownEngineError struct {
	Name string // the engine the record names
	ID   string // the task whose record names it
}

func (e *unknownEngineError) Error() string {
	// Without a task there is no "ran on": the supervisor is asked for by
	// the engine the window's dial names, and nothing has run yet.
	if e.ID == "" {
		return e.Name + " is not an engine this build of orbit can run"
	}

	return "task " + e.ID + " ran on " + e.Name + ", which this build of orbit cannot run"
}

// engineNamed is the engine a name asks for, or a refusal naming it.
//
// A name nothing answers to is refused rather than quietly replaced with
// claude, which is what the supervisor's two doors did. The window's engine
// dial is a choice a reader makes about which model reads their tasks and
// what it costs them; answering a different one and labelling the reply with
// the name they picked is the one outcome they cannot detect.
func engineNamed(engines map[string]engine.Engine, name string) (engine.Engine, error) {
	eng, ok := engines[name]
	if !ok || eng == nil {
		return nil, &unknownEngineError{Name: name}
	}

	return eng, nil
}

// askSupervisorPort is the supervisor thread: what a reader types in it goes
// to the engine their dial names, and the answer comes back into the thread.
func askSupervisorPort(s *store.Store, engines map[string]engine.Engine) func(string, string) (string, error) {
	return func(name, prompt string) (string, error) {
		eng, err := engineNamed(engines, name)
		if err != nil {
			return "", err
		}

		return task.Supervise(context.Background(), s, eng, prompt)
	}
}

// autoSupervisePort is the same engine, asked on autopilot about the tasks
// that are waiting for somebody.
func autoSupervisePort(s *store.Store, engines map[string]engine.Engine) func(string, []string) (string, error) {
	return func(name string, taskIDs []string) (string, error) {
		eng, err := engineNamed(engines, name)
		if err != nil {
			return "", err
		}

		return task.AutoSupervise(context.Background(), s, eng, taskIDs)
	}
}

// enginesPort adapts the engine map and declared engines to ui.Options.Engines.
func enginesPort(engines map[string]engine.Engine) func() []ui.EngineInfo {
	return func() []ui.EngineInfo {
		var list []ui.EngineInfo

		setupGuides := map[string][]string{
			"claude": {
				"1. Install Claude Code: npm install -g @anthropic-ai/claude-code",
				"2. Run 'claude' in a terminal to authenticate",
				"3. Ensure 'claude' is in your PATH",
			},
			"codex": {
				"1. Install Codex CLI: npm install -g @openai/codex",
				"2. Export OPENAI_API_KEY in your environment",
				"3. Ensure 'codex' is in your PATH",
			},
			"opencode": {
				"1. Install OpenCode CLI binary",
				"2. Configure local model endpoint or API keys",
				"3. Ensure 'opencode' is in your PATH",
			},
		}

		for _, name := range engineNames(engines) {
			eng, hasEng := engines[name]
			_, pathErr := exec.LookPath(name)

			available := hasEng && pathErr == nil
			if available {
				var models []ui.ChoiceInfo
				for _, m := range eng.Models() {
					models = append(models, ui.ChoiceInfo{ID: m.ID, Label: m.Label})
				}

				var efforts []ui.ChoiceInfo
				for _, e := range eng.Efforts() {
					efforts = append(efforts, ui.ChoiceInfo{ID: e.ID, Label: e.Label})
				}

				list = append(list, ui.EngineInfo{
					Name:      name,
					Available: true,
					Models:    models,
					Efforts:   efforts,
					CanThink:  eng.CanThink(),
				})
			} else {
				list = append(list, ui.EngineInfo{
					Name:      name,
					Available: false,
					Setup:     setupGuides[name],
				})
			}
		}

		return list
	}
}
