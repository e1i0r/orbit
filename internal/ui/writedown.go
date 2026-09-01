package ui

// Where a failure the reader saw goes after it has been said on screen.

import (
	"errors"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/logger"
)

// notedErrs is the last failure written down by each source that arrives on
// a clock. There are four of them and they are named here rather than kept
// in a map, because the fact worth being able to read is that only four
// messages in this window repeat themselves without a reader asking.
type notedErrs struct{ board, diff, record, files string }

// writeDown writes down whatever error arrived with a message.
//
// It is called once, at the top of Update, and not from the cases below it
// that each carry one. Two places were candidates and only this one
// is right: an error is news when it arrives, and drawing is not when it
// arrives — six panes render m.logErr through errSaid on every frame, so a
// line written there would be one failure written down as often as the
// terminal is repainted.
//
// Nothing here changes what is drawn. A window whose log is going nowhere
// draws exactly the frame it drew before.
func (m Model) writeDown(msg tea.Msg) Model {
	switch msg := msg.(type) {
	// The three that arrive on a clock are written down when they change,
	// and not on every beat: a board that cannot be read is read again a
	// second later, and the same sentence once a second is a file with
	// nothing findable in it.
	case boardMsg:
		m.noted.board = writeOnce("ui/board", "", oneLine(msg.Board.Errs...), m.noted.board)
	case diffMsg:
		m.noted.diff = writeOnce("ui/diff", msg.ID, oneLine(msg.Err), m.noted.diff)
	case logMsg:
		m.noted.record = writeOnce("ui/record", msg.ID, oneLine(msg.Err), m.noted.record)
	case filesMsg:
		m.noted.files = writeOnce("ui/files", msg.ID, oneLine(msg.Err), m.noted.files)

	// The rest are gestures. A reader pressed a key, so a failure that
	// reads the same as the one before it is a second attempt that also
	// failed, and both of them belong in the file.
	case controlMsg:
		writeLine("ui/control", msg.Word+" "+msg.ID, oneLine(msg.Err))
	case startedMsg:
		writeLine("ui/start", msg.ID, oneLine(msg.Err))
	case readMsg:
		writeLine("ui/read", msg.ID, oneLine(msg.Err))
	case requeuedMsg:
		writeLine("ui/requeue", msg.ID, oneLine(msg.Err))
	case fileTextMsg:
		writeLine("ui/files", msg.ID+" "+msg.Name, oneLine(msg.Err))
	case sessionMsg:
		writeLine("ui/session", msg.ID, oneLine(msg.Err))
	case sessionEndedMsg:
		writeLine("ui/session", msg.ID, oneLine(msg.Err))
	case cliEndedMsg:
		writeLine("ui/session", msg.Engine+" in "+msg.Repo, oneLine(msg.Err))
	case commandMsg:
		writeLine("ui/command", msg.Name, oneLine(msg.Err))
	case editorMsg:
		writeLine("ui/editor", "", oneLine(msg.Err))
	case supervisorReplyMsg:
		writeLine("ui/supervisor", "", oneLine(msg.Err))
	}

	return m
}

// oneLine is what a set of failures says, as one line.
//
// The errors are said in English, untranslated, and on purpose: errSaid
// draws them in the reader's language and this writes down what the machine
// underneath actually returned, which is the half a reader pastes into an
// issue. errors.Join separates with newlines, and every entry in the log is
// one line with one timestamp on it, so the newlines become semicolons
// rather than three entries the format does not fit.
func oneLine(errs ...error) string {
	err := errors.Join(errs...)
	if err == nil {
		return ""
	}

	return strings.ReplaceAll(err.Error(), "\n", "; ")
}

// writeLine writes one failure down, with whatever it was about, and writes
// nothing at all when there was no failure.
func writeLine(module, about, text string) {
	switch {
	case text == "":
	case about == "":
		logger.Error(module, "%s", text)
	default:
		logger.Error(module, "%s: %s", about, text)
	}
}

// writeOnce is writeLine for the clocked sources: it writes the failure down
// unless it is the failure already written, and answers with the line the
// next beat is compared against. A failure that clears and comes back is
// news again, which is why what is kept is the line and not a flag.
func writeOnce(module, about, text, last string) string {
	if text != last {
		writeLine(module, about, text)
	}

	return text
}
