package flows

// The editor's per-phase dials: they answer with what the engine the phase
// runs on actually offers, and never with a list written out here.

import (
	"slices"
	"testing"
)

// The flow editor's per-phase dials. They were three lists written out in
// flowsdelta.go, and the model one offered every engine sonnet — claude's
// alone — so a codex phase could be built holding a model codex refuses.
func TestTheFlowEditorDialsAreTheEnginesOwn(t *testing.T) {
	s, e := designer(t)
	e.Engines = zeta
	s = s.startCreateFlow(e).OnFields()

	for _, tc := range []struct {
		field int
		got   func(s State) string
		want  []string
	}{
		{flowFieldEngine, func(s State) string { return s.cur().Engine }, []string{"zeta"}},
		{flowFieldModel, func(s State) string { return s.cur().Model }, []string{"zeta/one", "zeta/two"}},
		{flowFieldEffort, func(s State) string { return s.cur().Effort }, []string{"brisk"}},
	} {
		s.field = tc.field
		// Around the dial twice, so a list longer than the engine's would
		// show up as a value that is not on it rather than as an order.
		for range 2 * len(tc.want) {
			next, _ := s.handleFlowFieldDelta(1, e)
			s = next

			if got := tc.got(s); !slices.Contains(tc.want, got) {
				t.Errorf("field %d cycled onto %q, which is not one of zeta's %v", tc.field, got, tc.want)
			}
		}
	}
}
