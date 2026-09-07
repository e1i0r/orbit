package ui

// The knowledge screen as the window reaches it. What the screen itself does
// is tested in internal/ui/known.

import "testing"

// TestKOpensTheKnowledgeScreen. The one screen where what Orbit knows can be
// read whole, and the place the supervisor's side panel points at.
func TestKOpensTheKnowledgeScreen(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	if m = step(t, m, "K"); m.screen != screenKnowledge {
		t.Fatalf("K left the window on %v", m.screen)
	}

	if back := step(t, m, "esc"); back.screen != screenList {
		t.Errorf("esc left the window on %v, want the board", back.screen)
	}
}
