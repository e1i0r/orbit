package cli

// The engines this build can run: the one table of them, and everything the
// window asks of it.
//
// It is one table, and the questions asked of it live beside it. Written out
// three times — twice as a map of constructors, in `orbit run` and in the
// window, and once more as a list of bare names for the settings screen — an
// engine added to one is an engine the others do not have: startable from the
// command line, and neither offered on the settings screen nor recognised
// when the window goes looking for it.
//
// The ports are here rather than in ports.go for the same reason they are in
// this package at all: they take a *store.Store, which internal/ui cannot
// name. What they have in common with each other is the table above.

import (
	"context"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"slices"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/supervisor"
	"github.com/e1i0r/orbit/internal/ui"
	"github.com/e1i0r/orbit/internal/view"
	"github.com/e1i0r/orbit/internal/words"
)

// newEngines is every engine this build can run, by the name a record
// carries.
//
// A map and not a default, because the record already names its engine: a
// task run by something this build does not have has to be answered by name
// rather than by assumption.
func newEngines() map[string]engine.Engine {
	return map[string]engine.Engine{
		"agy":      engine.NewAgy(),
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

		return supervisor.Supervise(context.Background(), s, eng, prompt)
	}
}

// draftPort is one engine asked one question, with no thread and no
// contract around it: the flow designer's third tab, which turns a sentence
// into a draft flow.
//
// It is read-only and it runs in the state root rather than in a repository.
// Writing a flow is reading a request and answering with JSON; an engine
// that can also change the checkout would be one that can act on a sentence
// nobody has approved yet.
func draftPort(engines map[string]engine.Engine) func(string, string, string) (string, error) {
	return func(name, model, prompt string) (string, error) {
		eng, err := engineNamed(engines, name)
		if err != nil {
			return "", err
		}

		// An empty directory of its own, thrown away afterwards.
		//
		// Two engines — agy and opencode — refuse to run a phase narrower
		// than repo, because their headless runs cannot enforce anything
		// narrower and they say so rather than recording a posture that is
		// not real. So a draft cannot be asked for read-only everywhere,
		// and asking for repo in the state root would put a model that was
		// asked for JSON in reach of the record. Here it can write all it
		// likes into a directory with nothing in it.
		dir, err := os.MkdirTemp("", "orbit-draft-")
		if err != nil {
			return "", fmt.Errorf("make a scratch directory for the draft: %w", err)
		}

		defer func() { _ = os.RemoveAll(dir) }() //nolint:errcheck // a temporary directory the OS will take back

		out, err := eng.Run(context.Background(), engine.Request{
			Prompt:      prompt,
			Model:       model,
			Dir:         dir,
			Permissions: []string{engine.PermissionRead, engine.PermissionRepo},
		})
		if err != nil {
			return "", err
		}

		return out.Output, nil
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

		return supervisor.AutoSupervise(context.Background(), s, eng, taskIDs)
	}
}

// enginesPort adapts the engine map and declared engines to ui.Options.Engines.
//
// What an engine offers and whether this machine can run it are two separate
// facts, and both are answered for every engine. The models and the efforts
// come from the adapter in internal/engine, which knows them whether or not
// the command line is installed; only Available is a fact about $PATH.
//
// Filling them in only for engines that are installed leaves the window with
// nothing to draw a dial from unless the reader already has the engine —
// which is how the window comes to keep its own copy of the catalogue, and
// that copy is what drifts.
func enginesPort(engines map[string]engine.Engine) func() []ui.EngineInfo {
	return func() []ui.EngineInfo {
		var list []ui.EngineInfo

		for _, name := range engineNames(engines) {
			eng, hasEng := engines[name]
			if !hasEng || eng == nil {
				continue
			}

			// The engine answers where its program is, rather than this
			// package guessing that the name is the binary and PATH is
			// the only place to look. opencode installs into
			// ~/.opencode/bin and puts that on a shell profile, so a PATH
			// exported before the install has no opencode in it — and
			// this screen said "[setup required]" to readers who had
			// opencode running in the next window.
			_, pathErr := eng.Locate()

			info := ui.EngineInfo{
				Name:      name,
				Available: pathErr == nil,
				Models:    choices(eng.Models()),
				Efforts:   choices(eng.Efforts()),
				CanThink:  eng.CanThink(),
			}
			// The steps are only worth carrying for an engine that cannot
			// run: the screen shows them in place of the dials.
			if !info.Available {
				info.Setup = setupGuide(name)
			}

			list = append(list, info)
		}

		return list
	}
}

// choices carries one engine's dial across the port, which is a copy because
// internal/ui may not name internal/engine.
func choices(from []engine.Choice) []ui.ChoiceInfo {
	var out []ui.ChoiceInfo
	for _, c := range from {
		out = append(out, ui.ChoiceInfo{ID: c.ID, Label: c.Label})
	}

	return out
}

// setupGuide is what to do about an engine this machine cannot run yet.
//
// It is a function of a printer and not three strings because these are
// sentences a reader reads, and every other line of that screen — its title,
// its notice, the way back out of it — is already in their language. Three
// English steps in the middle of it were the only thing on the screen that
// was not, and they are the one part a reader has to follow.
//
// What is translated is the instruction and not the command: `npm install -g
// @anthropic-ai/claude-code` is typed, not read, and a translated command is
// a command that does not run.
//
// An engine with no guide answers nil, and the screen shows its heading with
// no steps under it. That is the honest answer for an engine nobody has
// written the steps for yet — better than three lines of somebody else's.
func setupGuide(name string) func(*words.Printer) []string {
	switch name {
	case "agy":
		return func(p *words.Printer) []string {
			return []string{
				p.T("engine.setup.agy.install", "1. Install the Antigravity CLI: agy is its binary"),
				p.T("engine.setup.agy.login", "2. Run 'agy' in a terminal and sign in with your Google account"),
				p.T("engine.setup.agy.path", "3. Ensure 'agy' is in your PATH"),
			}
		}
	case "claude":
		return func(p *words.Printer) []string {
			return []string{
				p.T("engine.setup.claude.install", "1. Install Claude Code: npm install -g @anthropic-ai/claude-code"),
				p.T("engine.setup.claude.login", "2. Run 'claude' in a terminal to authenticate"),
				p.T("engine.setup.claude.path", "3. Ensure 'claude' is in your PATH"),
			}
		}
	case "codex":
		return func(p *words.Printer) []string {
			return []string{
				p.T("engine.setup.codex.install", "1. Install Codex CLI: npm install -g @openai/codex"),
				p.T("engine.setup.codex.key", "2. Export OPENAI_API_KEY in your environment"),
				p.T("engine.setup.codex.path", "3. Ensure 'codex' is in your PATH"),
			}
		}
	case "opencode":
		return func(p *words.Printer) []string {
			return []string{
				p.T("engine.setup.opencode.install", "1. Install OpenCode CLI binary"),
				p.T("engine.setup.opencode.configure", "2. Configure local model endpoint or API keys"),
				p.T("engine.setup.opencode.path", "3. Ensure 'opencode' is in your PATH"),
			}
		}
	}

	return nil
}
