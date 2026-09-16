package cli

// The whole settings table, as the window draws it and writes it back.
//
// Apart from settings.go for the reason internal/verb keeps its own table
// and its own writer apart: that file is this adapter's typed answers — the
// handful the header and the board ask for by name — and this is the pair
// that speaks every setting there is. The two were one file until it went
// over the ceiling.

import (
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/verb"
	"github.com/e1i0r/orbit/internal/words"
)

// Kept is every setting the vocabulary declares, as the window draws one.
//
// Straight through to internal/verb for the reason Fresh is: the table, what
// each row means and what each will accept are declared once, beside each
// other. The window used to keep its own list of rows, and it fell six
// behind.
func (a *settingsAdapter) Kept(p *words.Printer) []verb.Setting {
	return verb.Kept(p, a.read())
}

// Choose writes one setting by the name the vocabulary gives it, through the
// validator declared beside it, and answers the form worth showing.
//
// It goes through this adapter's own write, so a row changed on screen takes
// the same lock and refreshes the same cache a typed setter does — the
// window must not argue with the reader about what it just set, whichever
// door the value came through.
func (a *settingsAdapter) Choose(p *words.Printer, key, value string) (string, error) {
	// A refusal travels out here rather than out of write. The closure
	// write takes cannot fail, so a value the vocabulary will not have
	// leaves the field alone and is raised once the file is closed — and
	// the file is written either way, with nothing changed in it, which is
	// what UpdateSettings does with a closure that touched nothing.
	var (
		shown   string
		refused error
	)

	err := a.write(func(cfg *store.Settings) {
		out, bad := verb.Choose(p, cfg, key, value)
		if bad != nil {
			refused = bad

			return
		}

		shown = out
	})

	switch {
	case refused != nil:
		return "", refused
	case err != nil:
		return "", err
	}

	return shown, nil
}
