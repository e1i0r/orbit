package ui

// The watched run: what a palette command prints while it runs, and the
// body that shows it.
//
// It is a file of its own because it is a different thing from the palette
// that raised it: the line answers "which command did you mean", this
// answers "what is that command doing right now". The buffer sits between
// the runner goroutine and the draw loop for the reason Model's own comment
// gives — neither may write the other — and everything here reads or writes
// only that buffer and the two model fields watching it.

import (
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// outputCap is how much of a command's output the window keeps. A run can
// print megabytes, and the reader is watching the end of it: everything
// past the cap is dropped off the front, which is the same honesty the
// evidence cut argues for — what was kept is all there is, and nothing
// pretends otherwise.
const outputCap = 8192

// outputTick is how often the watch's buffer is read onto the screen while
// its command runs. Faster than that is faster than a reader reads; slower
// starts to feel like the window is not listening.
const outputTick = 150 * time.Millisecond

// commandWatch is one palette command's output while it runs.
//
// It exists because the run happens in a tea.Cmd goroutine and the model is
// a value: neither may write the other. The buffer sits between them behind
// a mutex — the runner writes, the poller reads a snapshot, and no field of
// Model is ever touched from another goroutine. done is closed when the
// command has returned, so a poll arriving after the verdict can be told
// from one racing it.
type commandWatch struct {
	name string
	said string // the sentence to leave behind, when the name is not one

	mu   sync.Mutex
	buf  []byte
	done bool
}

// Write keeps the last outputCap bytes of whatever the command prints,
// dropping the front once the cap is passed.
func (w *commandWatch) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf = append(w.buf, p...)
	if len(w.buf) > outputCap {
		w.buf = w.buf[len(w.buf)-outputCap:]
	}

	return len(p), nil
}

// finish marks the run over; the final snapshot is taken by whoever got the
// verdict, after this returns.
func (w *commandWatch) finish() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.done = true
}

// snapshot is the output as it stands right now.
func (w *commandWatch) snapshot() (string, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	return string(w.buf), w.done
}

// closeWatch takes the run's output off the screen without pretending the
// run itself can be stopped: Orbit has no handle on the process to send it
// anything, and a close button that claimed otherwise would be lying. The
// band still says how it ended.
func (m Model) closeWatch() Model {
	m.watchUp = false
	return m
}

// reopenWatch brings a still-running command's output back up. It is the
// answer to the reader who closed it to look at the board and now wants
// the run again: choosing the same command in the palette reopens rather
// than restarts.
func (m Model) reopenWatch() Model {
	m.watchUp = true
	return m
}

// runSelected raises commandMsg for the selection, through the same port
// the keyboard and the pointer both reach. A refusal comes back verbatim
// and is said in the band, because the window is a keyboard in front of
// the commands and not a second copy of their rules.
//
// The run is watched: the body turns over to its output until esc takes it
// down or the command finishes, so the reader sees the work as it happens
// and the result when it lands — one command at a time, since a second
// interleaved into the first's output would be two answers to neither
// question.
//
// The line is split on spaces and quoting is not understood: a path with a
// space in it cannot be passed this way yet, and the compose screen is the
// answer for the one command that most wants one.
func (m Model) runSelected() (tea.Model, tea.Cmd) {
	c, ok := m.palette.selected(m.opts.Commands)
	if !ok {
		// Nothing under the selection: an empty list, usually. Staying
		// open says "not yet" more honestly than closing would.
		return m, nil
	}

	args := strings.Fields(m.palette.typed)
	if len(args) > 0 {
		args = args[1:]
	}

	if c.NeedsArgs && len(args) == 0 {
		// The line stays up, with the name still on it, because what is
		// missing goes on the end of what is already typed. Closing it to
		// print the same sentence into the watch is where this used to end:
		// an answer in a pane with nowhere to type what it asks for.
		//
		// The usage is the table's own, so the sentence is right for every
		// command that reaches it without one being written per command.
		return m.say(m.opts.Words.T("msg.needs_args", "{name} takes {args}; type them here",
			about("name", c.Name), about("args", c.Args))), nil
	}

	return m.closePalette().launch(c, args)
}

// launch starts what a named command does from inside the window, and it
// is the one door both entry points go through — the palette's ⏎ and the
// board menu's entries.
//
// `new` is the exception and is answered with a screen: writing a task is
// a form and not a flag line, and N, :new and the board menu all land on
// the same one. The command itself still runs — the form submits through
// it — so the table keeps its rule that no command exists in one entry
// point and not the other.
func (m Model) launch(c Command, args []string) (tea.Model, tea.Cmd) {
	// The screens first. A command the window answers with one of its own
	// is answered with it however little was typed: `new` refuses on the
	// command line without -id, and the compose form is where that id is
	// filled in.
	switch c.Name {
	case "new":
		return m.openCompose(), nil
	case "set", "settings":
		return m.openSettings(), nil
	case "flows":
		return m.openFlows(), nil
	case "repos":
		return m.openRepos(), nil
	case "supervisor":
		return m.openSupervisor(), nil
	}

	// Then what is left to run, which cannot be run with nothing. The menu
	// chooses with no arguments and has none to give — no task, no
	// directory — so running one of these bare printed "requeue needs the
	// id of a task" into the watch: an answer to a question nobody asked,
	// in a pane with nowhere to type what it asks for. The line is where
	// they can be typed, and it shows the usage under it.
	if c.NeedsArgs && len(args) == 0 {
		return m.openPaletteWith(c.Name + " "), nil
	}

	return m.runWatched(c, args)
}

// runWatched runs one command under the watch: one at a time, its output
// on screen while it works and its result left there when it lands. The
// caller closes whatever list it was showing; the watch opens here.
func (m Model) runWatched(c Command, args []string) (tea.Model, tea.Cmd) {
	return m.runWatchedSaying(c, args, "")
}

// runWatchedSaying is runWatched with the sentence to leave on the band when
// the command lands.
//
// The verb a reader pressed is not always the command that carries it out:
// fix checks and more tests are both `orbit note`, and a reader who pressed
// neither of those was answered "note finished". What is said here is what
// was asked for, in the words the keystroke was offered in.
func (m Model) runWatchedSaying(c Command, args []string, said string) (tea.Model, tea.Cmd) {
	if m.watching != nil {
		if c.Name == m.watching.name {
			return m.reopenWatch(), nil
		}

		return m.say(m.opts.Words.T("watch.busy", "{name} is still running",
			about("name", m.watching.name))), nil
	}

	w := &commandWatch{name: c.Name, said: said}
	next := m
	next.watching, next.watchUp, next.output = w, true, ""

	return next, tea.Batch(runCommand(m.opts.Do, w, args), outputPump(w))
}

// watchRows is the body while a run's output is up: the tail of what the
// command has printed so far, newest at the bottom, with one line under it
// saying which of the two states it is in. The tail follows on its own —
// there is no scroll here to release, because nothing has been offered
// worth scrolling back for yet.
func (m Model) watchRows(h, w int) []string {
	if h <= 0 {
		return nil
	}

	p := m.opts.Words

	var status string
	if m.watching != nil {
		status = Paint(Dim).Render(p.T("watch.running", "{name}: still running…", about("name", m.watching.name)))
	} else {
		status = Paint(Dim).Render(p.T("watch.finished_line", "finished — {back} closes",
			about("back", m.keys.Back.Help().Key)))
	}

	lines := strings.Split(strings.TrimRight(strings.ReplaceAll(m.output, "\r\n", "\n"), "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = []string{p.T("watch.quiet", "no output yet…")}
	}

	if len(lines) > h-1 {
		lines = lines[len(lines)-(h-1):]
	}

	out := make([]string, 0, h)
	for _, l := range lines {
		out = append(out, fit("  "+l, w))
	}

	return fill(append(out, "  "+status), h)
}

// watchKey answers the keyboard while a run's output is up. It keeps only
// the way out; every other key does nothing rather than acting on a board
// the reader cannot currently see.
func (m Model) watchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Open):
		return m.closeWatch(), nil
	}

	return m, nil
}
