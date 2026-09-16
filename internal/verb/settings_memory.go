package verb

// The settings table in memory, for a surface that needs one to stand in for
// the file.

import (
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// Table is every setting, held in memory, answering the same two doors the
// settings file answers through.
//
// It exists because of what went wrong without it. Every surface that draws
// the settings had a fixture with its own table written out by hand, and a
// fixture with its own table keeps passing while the screen it stands for
// falls behind — which is exactly how six settings came to be declared and
// never drawn. A fixture over this one is a fixture that knows about a
// setting the afternoon it is declared, and refuses what the real table
// refuses.
//
// It is not concurrent and does not pretend to be: the file has a lock
// because several orbits write it, and nothing that holds one of these is
// shared with anything.
type Table struct{ cfg store.Settings }

// NewTable is the table as Orbit ships it, before anybody has chosen
// anything.
func NewTable() *Table { return &Table{cfg: store.Shipped()} }

// Kept is every setting and what this table holds for it.
func (t *Table) Kept(p *words.Printer) []Setting { return Kept(p, t.cfg) }

// Choose writes one setting by name, through the validator declared beside
// it, and answers the form worth showing.
func (t *Table) Choose(p *words.Printer, key, value string) (string, error) {
	return Choose(p, &t.cfg, key, value)
}

// Fresh is what one setting reads as when nobody has chosen anything.
func (t *Table) Fresh(key string) string { return Fresh(key) }

// Value is what this table holds for one setting, for a caller that wants
// one answer rather than the list.
func (t *Table) Value(key string) string {
	for _, one := range settingTable() {
		if one.Name == key {
			return one.Value(t.cfg)
		}
	}

	return ""
}
