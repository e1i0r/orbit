package ui

// One frame, as text, for something that is not a terminal.
//
// It is the same board through the same view functions — the same header,
// the same rows, the same words in the same language — with the styling
// stripped and the whole list open. Nothing here runs an event loop, asks
// the terminal anything, or starts a goroutine: Plain is a function from
// Options to a string, which is what makes the window's layout testable
// without a pseudo-terminal and what makes `orbit top -once` usable in a
// pipe, in CI, and under TERM=dumb.
//
// The program this replaces could only be checked by looking at it. That is
// the entire reason this file exists.

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/e1i0r/orbit/internal/view"
)

// The frame one plain render is drawn at when the caller names no size.
//
// There is no terminal to ask, so a number has to be chosen: a hundred cells
// is wide enough for every column of a row to survive the drop order, and
// thirty rows is a screen. A caller that knows better — a test pinning a
// width, a script that has measured its own output — sets Width and Height
// and is given exactly that.
const (
	plainWidth  = 100
	plainHeight = 30
)

// Plain draws one frame of the board and hands it back as text.
//
// It reads the board once, synchronously, through the same port the window
// polls, and then renders. The Cmd the update would have returned is
// discarded rather than run, which is what keeps the bell out of a pipe: a
// board that crosses into NEEDS YOU rings the terminal, and a byte of that
// kind in a file is corruption rather than a notification. The first board a
// window ever sees rings nothing anyway — this is the belt as well as the
// braces.
//
// A nil Reader is not a failure. It is a window opened without a state root,
// and what it draws is the empty state, which is a sentence rather than a
// blank screen.
func Plain(o Options) (string, error) {
	if o.Width <= 0 {
		o.Width = plainWidth
	}
	if o.Height <= 0 {
		o.Height = plainHeight
	}
	m := New(o)
	// Every band open. The window opens on NEEDS YOU and RUNNING and leaves
	// the other two shut because the reader can press o — and there is no o
	// to press in a pipe, so a shut band is a band whose rows nobody can
	// ever see. A frame of four headings over nothing would be the wiring
	// looking correct while showing none of it.
	m.expanded = everyBand()
	if o.Reader != nil {
		b, changed, err := o.Reader.Refresh()
		if err != nil {
			return "", fmt.Errorf("%s: %w", m.opts.Words.T("plain.read_board", "read the board"), err)
		}
		next, _ := m.Update(boardMsg{Board: b, Changed: changed})
		loaded, ok := next.(Model)
		if !ok {
			return "", fmt.Errorf("%s: %T", m.opts.Words.T("plain.not_model", "the window answered with an unexpected type"), next)
		}
		m = loaded
	}
	return plainText(m.grown().View().Content), nil
}

// everyBand is the four bands, every one of them open.
//
// It is built from view.Bands rather than written out, so a fifth band is
// open here the day it exists rather than the day somebody notices this map
// has four keys in it.
func everyBand() map[view.Band]bool {
	open := map[view.Band]bool{}
	for _, b := range view.Bands() {
		open[b] = true
	}
	return open
}

// grown gives the body enough rows to hold the whole list.
//
// On a terminal a list that runs past the fold spends its last row saying
// how many more there are, and the reader scrolls. Neither half of that
// works here: nothing scrolls a pipe, so "… and nine more" is nine tasks
// this frame simply does not contain. Growing the frame is the only answer
// that keeps -once honest about what the board holds.
//
// It asks the frame how tall its body is rather than subtracting a number of
// its own, because how many rows the furniture takes is layout's arithmetic
// and a second copy of it here would be the class of bug that package exists
// to end. Every row a resize adds goes to the body, so the body ends up
// exactly as tall as the list and the "and more" row is never spent.
func (m Model) grown() Model {
	if m.tooNarrow {
		return m
	}
	extra := len(m.rows()) - m.frame.Body.H
	if extra <= 0 {
		return m
	}
	return m.resize(m.width, m.height+extra)
}

// plainText takes the styling off a rendered frame and the trailing spaces
// off its lines.
//
// The padding is what makes a frame exactly as wide as the terminal it was
// drawn for, and in a file it is a run of spaces at the end of every line —
// which a golden, a reviewer and every tool between them read as noise. The
// frame keeps its full height, blank rows included: what -once renders is
// the window, not a summary of it.
func plainText(frame string) string {
	lines := strings.Split(ansi.Strip(frame), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}
