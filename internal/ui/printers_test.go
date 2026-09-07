package ui

// The two printers a translated sentence is checked in.
//
// It sits in its own file because three suites reach for it and the one it
// used to live beside — the key map's — is a package of its own now.

import (
	"testing"

	"github.com/e1i0r/orbit/internal/ui/keymap"
	"github.com/e1i0r/orbit/internal/words"
)

func printers(t *testing.T) (english, spanish *words.Printer) {
	t.Helper()
	t.Setenv("ORBIT_HOME", t.TempDir())

	return words.For("en"), words.For("es")
}

// find is the affordance for one verb, or an empty one when the verb is not
// offered at all — which is a difference the tests that call it check for
// themselves.
func find(as []keymap.Affordance, verb string) keymap.Affordance {
	for _, a := range as {
		if a.Key.Keys()[0] == verb {
			return a
		}
	}

	return keymap.Affordance{}
}
