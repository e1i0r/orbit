package ui

import "testing"

// errandOf is the tag the supervisor's reply carries for the verb that is
// out, as askSupervisorCmd hands it back.
func errandOf(m Model) Errand {
	return Errand{Task: m.delivering.task, Verb: m.delivering.verb}
}

// TestAReplyToAnotherLineClosesNoVerb. A line the operator typed, or a turn
// of autopilot, came back down the same wire and closed whichever verb was
// out, with its own text written as that verb's answer. Only the reply to
// the verb's own ask closes it.
func TestAReplyToAnotherLineClosesNoVerb(t *testing.T) {
	m, written := deliverWindow(t)

	next, _ := m.fixChecks()
	out := asModel(t, next)

	stray, _ := out.Update(supervisorReplyMsg{Text: "hello to you too"})
	if v := asModel(t, stray).delivering.verb; v != "FIX CHECKS" || len(*written) != 1 {
		t.Fatalf("a reply to another line left verb %q and wrote %d events; "+
			"want FIX CHECKS still out, 1 event", v, len(*written))
	}

	answered, _ := asModel(t, stray).Update(supervisorReplyMsg{Text: "green", About: errandOf(out)})
	if v := asModel(t, answered).delivering.verb; v != "" || len(*written) != 2 {
		t.Errorf("the verb's own reply left %q out and wrote %d events, want it closed", v, len(*written))
	}
}
