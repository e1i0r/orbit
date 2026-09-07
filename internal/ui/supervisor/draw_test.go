package supervisor

// How a turn is drawn depends on who wrote it.

import "testing"

// A thread line's author was matched against a list of five names, so a
// sixth engine's answers were drawn in a person's colour and its markdown
// was re-wrapped as plain text. The roster is asked first now.
func TestAThreadLineIsRecognisedAsAnEnginesByTheRoster(t *testing.T) {
	_, e := open(t)

	if !isEngineName("zeta", e) {
		t.Error("a line written by an engine this build has is taken for a person's")
	}

	if !isEngineName("supervisor", e) {
		t.Error("the supervisor's own line is taken for a person's")
	}

	// A thread is a record and can hold an engine this build no longer
	// has, which a person did not type either.
	if !isEngineName("gemini", e) {
		t.Error("a line recorded from an engine this build dropped is taken for a person's")
	}

	if isEngineName("elio", e) {
		t.Error("a person's line is taken for an engine's")
	}
}
