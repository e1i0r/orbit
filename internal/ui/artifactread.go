package ui

// What an opened file turned out to hold.
//
// The artifacts pane draws the answer and the window keeps it: a file is
// read when the reader opens it and not before, because reading every file
// of the directory on every tick to draw a list of names would be a read
// nobody asked for.

import (
	"maps"

	"github.com/e1i0r/orbit/internal/view"
)

// fileRead is what one file turned out to hold, or why it could not be read.
// The two are one type because a row shows one or the other and never both.
type fileRead struct {
	text view.FileText
	err  error
}

// readFile writes down what a file turned out to hold.
//
// The map is cloned rather than written in place, for the reason every other
// map on the model is: a Model is copied by every method that returns one
// and a map is not.
func (m Model) readFile(msg fileTextMsg) Model {
	held := maps.Clone(m.read)
	if held == nil {
		held = map[string]fileRead{}
	}

	held[msg.Name] = fileRead{text: msg.Text, err: msg.Err}
	m.read = held

	return m.syncPanes()
}
