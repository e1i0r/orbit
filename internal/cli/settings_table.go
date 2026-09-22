package cli

// The whole settings table, as the window draws it and writes it back.
//
// Apart from settings.go for the reason internal/verb keeps its own table
// and its own writer apart: that file is this adapter's typed answers — the
// handful the header and the board ask for by name — and this is the pair
// that speaks every setting there is. The two were one file until it went
// over the ceiling. The window's own wrapper around the adapter, which
// remembers what language it is speaking, is at the end of this one.

import (
	"sync"

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

// spokenSettings is the window's settings port, and it remembers the
// language the window is speaking right now.
//
// The window is drawn in the language the four sources weigh when it opens
// — -lang, $ORBIT_LANG, the saved setting, $LANG — and in whatever the
// reader picks after that. The saved setting alone is neither: over an empty
// settings file, `ORBIT_LANG=es orbit top` is a Spanish window whose saved
// language is nothing. A command typed at that window's command line was
// handed the saved setting, and refused in English under a Spanish screen.
//
// Language is left as the settings file answers it, because the settings
// screen shows what is saved and not what is spoken. Speaking is the other
// question, and it is the one doPort asks.
type spokenSettings struct {
	*settingsAdapter

	mu   sync.Mutex
	code string
}

// SetLanguage writes the reader's pick down and speaks it from now on — the
// window has already switched its own printer to it by the time this
// returns, and its commands switch with it.
func (s *spokenSettings) SetLanguage(lang string) error {
	if err := s.settingsAdapter.SetLanguage(lang); err != nil {
		return err
	}

	s.mu.Lock()
	s.code = lang
	s.mu.Unlock()

	return nil
}

// speaking is the language the window is in. The window's commands run off
// the event loop, so the pick is read under the same lock it is written in.
func (s *spokenSettings) speaking() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.code
}

// spokenPort is speaking in the shape doPort asks for.
type spokenPort func() string

// Language is the code the window is speaking.
func (f spokenPort) Language() string { return f() }
