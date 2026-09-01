package ui

// Every keystroke the window answers, and the verbs they raise.
//
// It is a file of its own because the 300-line ceiling would not have both
// this and the model in one, and because the split falls where a reader
// would put it anyway: ui.go is what the window is, this is what it can be
// asked to do. What a verb is allowed to do is not decided here — that is
// affordance.go, and this file only ever asks it.

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// confirmYes is the one keystroke that answers a question with yes.
//
// It is not translated, and that is deliberate: a keystroke is a key on a
// keyboard rather than a word in a sentence, and a catalogue that moved it
// to "s" for "sí" would move it off the key the prompt names. The prompt
// says which key it is, and that sentence is translated.
const confirmYes = "y"

// key routes one keystroke to whichever of the window's modes has it.
//
// The order is the order things are on top of each other: a palette line,
// a menu or a filter being typed swallows every letter, a question waiting
// for an answer takes the next key whatever it is, and the two screens
// below the board have their own small maps. Only what is left reaches the
// list.
//
// The ':' that opens the palette is matched here rather than in any one
// screen's map because the palette is not the board's tool alone; so is m,
// which opens the menu on whatever the context offers. A run's output being
// up sits under both and above everything else, keeping only its own way
// out.
func (m Model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.note.open:
		return m.noteKey(msg)
	case m.palette.open:
		return m.paletteKey(msg)
	case m.menu.open:
		return m.menuKey(msg)
	case m.filtering:
		return m.filterKey(msg)
	case m.confirm != confirmNone:
		return m.confirmKey(msg)
	case m.watchUp:
		return m.watchKey(msg)
	case m.screen == screenStart:
		return m.startKey(msg)
	case m.screen == screenDetail:
		return m.detailKey(msg)
	case m.screen == screenCompose:
		return m.composeKey(msg)
	case m.screen == screenSettings:
		return m.settingsKey(msg)
	case m.screen == screenFlows:
		return m.flowsKey(msg)
	case m.screen == screenRepos:
		return m.repolistKey(msg)
	case m.screen == screenEngines:
		return m.enginesKey(msg)
	case m.screen == screenQuota:
		return m.quotaKey(msg)
	case m.screen == screenHelp:
		return m.helpKey(msg)
	case m.screen == screenSupervisor:
		return m.supervisorKey(msg)
	case key.Matches(msg, m.keys.Commands):
		return m.openPalette(), nil
	case key.Matches(msg, m.keys.Menu):
		return m.openMenuForContext(), nil
	}

	return m.listKey(msg)
}

// listKey is the board's own map.
//
// Every verb here goes through affordance first, so a key that the task
// under the cursor cannot take says why rather than doing nothing. Doing
// nothing is what a reader reads as a broken keyboard.
//
// It takes a fmt.Stringer and not a tea.KeyPressMsg, which is what
// key.Matches has always wanted, and the reason is the mouse: a hint clicked
// in the key bar arrives here as the keystroke that hint names, through this
// same map, so a verb cannot be reachable by the keyboard and not by the
// pointer. The alternative was a second switch over the same bindings, and
// two copies of a dispatch table disagree the first time one of them gains a
// verb.
func (m Model) listKey(k fmt.Stringer) (tea.Model, tea.Cmd) {
	if targetTab, ok := keyToPane(k.String()); ok {
		m = m.showTab(targetTab)
		return m, nil
	}

	switch {
	case key.Matches(k, m.keys.Back):
		if m.filter != "" || m.repoFilter != "" || m.queueFilter != nil {
			m.filter, m.repoFilter, m.queueFilter = "", "", nil
			return m.say(m.opts.Words.T("repos.filter_cleared", "showing all repositories")), nil
		}

		return m, nil
	case key.Matches(k, m.keys.Up):
		return m.move(-1), nil
	case key.Matches(k, m.keys.Down):
		return m.move(1), nil
	case key.Matches(k, m.keys.First):
		return m.moveTo(0), nil
	case key.Matches(k, m.keys.Last):
		return m.moveTo(len(m.rows()) - 1), nil
	case key.Matches(k, m.keys.PageUp):
		return m.move(-m.frame.Body.H), nil
	case key.Matches(k, m.keys.PageDown):
		return m.move(m.frame.Body.H), nil
	case key.Matches(k, m.keys.Open):
		return m.open()
	case key.Matches(k, m.keys.Filter):
		m.filtering = true
		return m, nil
	case key.Matches(k, m.keys.Repos):
		return m.openRepos(), nil
	case key.Matches(k, m.keys.EngineKnobs):
		return m.openEngines(), nil
	case key.Matches(k, m.keys.Quota):
		return m.openQuota(), nil
	case key.Matches(k, m.keys.Supervisor):
		return m.openSupervisor(), nil
	case key.Matches(k, m.keys.Autopilot):
		return m.autopilot()
	case key.Matches(k, m.keys.Pause):
		return m.verb(m.keys.Pause, "pause")
	case key.Matches(k, m.keys.Resume):
		return m.verb(m.keys.Resume, "resume")
	case key.Matches(k, m.keys.Hand):
		return m.handBack()
	case key.Matches(k, m.keys.Cancel):
		return m.ask()
	case key.Matches(k, m.keys.Requeue):
		return m.askRequeue()
	case key.Matches(k, m.keys.Take):
		return m.takeKey()
	case key.Matches(k, m.keys.MarkRead):
		return m.markReadKey()
	case key.Matches(k, m.keys.Delete):
		return m.askDeleteTask()
	case key.Matches(k, m.keys.Ask):
		return m.openNote(), nil
	case key.Matches(k, m.keys.Start):
		return m.openStart()
	case key.Matches(k, m.keys.Compose):
		return m.openCompose(), nil
	case key.Matches(k, m.keys.CLI):
		return m.launchInteractiveCLI()
	case key.Matches(k, m.keys.Help):
		return m.openHelp(), nil
	case key.Matches(k, m.keys.Quit):
		return m, tea.Quit
	}

	return m, nil
}

// askRequeue opens the confirm in front of b.
//
// It asks for the same reason cancel does and not for cancel's reason: the
// task is not lost — it is going back to to do, where it can be started
// again — but whatever was running when the key was pressed does not come
// back, and neither does what that phase spent.
func (m Model) askRequeue() (tea.Model, tea.Cmd) {
	t, next, ok := m.gesture(m.keys.Requeue)
	if !ok {
		return next, nil
	}

	next.confirm, next.confirmID = confirmRequeue, t.ID

	return next, nil
}

func (m Model) askDeleteTask() (tea.Model, tea.Cmd) {
	r, ok := m.selected()
	if !ok || r.head || r.blank {
		return m, nil
	}

	m.confirm = confirmDeleteTask
	m.confirmID = r.task.ID

	return m.say(m.opts.Words.T("msg.confirm_delete_task", "delete task {id}? press y or ⏎ to confirm deletion, any other key to cancel", about("id", r.task.ID))), nil
}

// confirmKey answers the one question the window ever asks.
//
// Anything that is not the confirming key is a no. A question that only a
// specific "no" closes is a question that traps a reader who has already
// looked away, and the safe answer to "shall I cancel this run" is no.
func (m Model) confirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	c := m.confirm
	id := m.confirmID

	m.confirm, m.confirmID = confirmNone, ""
	if c == confirmPostCliTask {
		if msg.String() == confirmYes || msg.String() == "s" || msg.String() == "S" || key.Matches(msg, m.keys.Open) {
			m = m.openCompose()
			if id != "" {
				m.compose.repo = id
				m.compose.field = composeText
			}

			return m.say(m.opts.Words.T("msg.compose_prompt", "write the task to run")), nil
		}

		return m.say(m.opts.Words.T("msg.cli_ended", "interactive session ended")), nil
	}

	if c == confirmDeleteTask {
		if msg.String() == confirmYes || msg.String() == "s" || msg.String() == "S" || key.Matches(msg, m.keys.Open) {
			t, ok := m.task(id)
			if ok && m.opts.DeleteTask != nil {
				if err := m.opts.DeleteTask(t); err != nil {
					return m.say(Paint(Bad).Render(err.Error())), nil
				}
			}

			if m.opts.Reader != nil {
				if err := m.opts.Reader.Rescan(); err != nil {
					return m.say(Paint(Bad).Render(err.Error())), nil
				}
			}

			return m.say(m.opts.Words.T("msg.task_deleted", "task {id} deleted", about("id", id))), nil
		}

		return m.say(m.opts.Words.T("msg.delete_cancelled", "deletion cancelled")), nil
	}

	if msg.String() != confirmYes {
		return m, nil
	}

	t, ok := m.task(id)
	if !ok {
		return m, nil
	}

	if c == confirmRequeue {
		return m, requeue(m.opts.Requeue, t)
	}

	return m, control(m.opts.Control, t, "cancel")
}

// open is one key doing two things, and it is not an overload: on a band
// header it opens the band in place, on a row it opens the task, and the
// cursor is on exactly one of the two.
func (m Model) open() (tea.Model, tea.Cmd) {
	r, ok := m.selected()
	if !ok {
		return m, nil
	}

	if r.head {
		return m.expand(r.band).clampCursor(), nil
	}

	return m.openDetail(r.task)
}

// verb asks the command behind one key to write one word. Whether the key is
// allowed at all is gesture's answer, in gesture.go.
func (m Model) verb(b key.Binding, word string) (Model, tea.Cmd) {
	t, next, ok := m.gesture(b)
	if !ok {
		return next, nil
	}

	return next, control(next.opts.Control, t, word)
}

// ask opens the confirm in front of a cancel.
//
// Cancelling is the one gesture here that cannot be undone by pressing
// something else — a run that was ended did not keep going — so it is the
// one that asks first.
func (m Model) ask() (tea.Model, tea.Cmd) {
	t, next, ok := m.gesture(m.keys.Cancel)
	if !ok {
		return next, nil
	}

	next.confirm, next.confirmID = confirmCancel, t.ID

	return next, nil
}

// autopilot flips the standing switch and says which way it went.
//
// It says what it just did rather than what it undid. The program this
// replaces printed "autopilot was off" after turning it on, and the sentence
// is ambiguous in exactly the moment a reader needs it not to be.
func (m Model) autopilot() (tea.Model, tea.Cmd) {
	if m.opts.Settings == nil {
		return m, nil
	}

	on := !m.opts.Settings.Autopilot()
	if err := m.opts.Settings.SetAutopilot(on); err != nil {
		return m.say(err.Error()), nil
	}

	if on {
		m = m.say(m.opts.Words.T("msg.autopilot_on", "autopilot is on: every phase runs without asking"))
		nextM, cmd := m.autoStartNext()

		return nextM, cmd
	}

	return m.say(m.opts.Words.T("msg.autopilot_off", "autopilot is off: every phase stops for you")), nil
}
