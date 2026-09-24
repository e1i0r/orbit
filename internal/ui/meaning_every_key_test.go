package ui

import (
	"reflect"
	"testing"

	"charm.land/bubbles/v2/key"
)

// TestEveryBoundKeyHasASentence. ? on M, and a hover over it in the bar,
// answered "nothing in this window answers M", while M opened the board's
// menu. Every key the keymap binds is walked, so the next one added
// without a sentence is caught here and not by a reader.
func TestEveryBoundKeyHasASentence(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	keys := reflect.ValueOf(m.keys)
	for i := range keys.NumField() {
		b, ok := keys.Field(i).Interface().(key.Binding)
		if !ok || len(b.Keys()) == 0 {
			continue
		}

		nothing := m.opts.Words.T("tip.nothing", "nothing in this window answers {key}", about("key", b.Keys()[0]))
		if says := m.meaning(keystroke(b.Keys()[0])); says == nothing {
			t.Errorf("? on %s (%s) says %q", b.Help().Key, keys.Type().Field(i).Name, says)
		}
	}
}
