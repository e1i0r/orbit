package ui

import (
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

func TestDetailKeyInteractionsAndTabSwitching(t *testing.T) {
	m := openOn(t, "ACME-2662")

	// 1. Direct tab number keys: 1-9, 0, w
	for _, k := range []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0", "w"} {
		keyMsg := keystroke(k)
		newM, _ := m.detailKey(keyMsg)
		m = asModel(t, newM)
	}

	// 2. Next tab and Prev tab
	nextKey := keystroke("]")
	newM, _ := m.detailKey(nextKey)
	m = asModel(t, newM)

	prevKey := keystroke("[")
	newM, _ = m.detailKey(prevKey)
	m = asModel(t, newM)

	// 3. Diff sideways toggle on diff tab
	m.tab = tabDiff
	sideKey := keystroke("s")
	newM, _ = m.detailKey(sideKey)
	m = asModel(t, newM)

	// 4. ? asks which key, and ? again is the whole cheat sheet
	helpKey := keystroke("?")
	newM, _ = m.detailKey(helpKey)

	m = asModel(t, newM)
	if !m.tip.armed {
		t.Error("'?' on the task view did not ask which key")
	}

	newM, _ = m.tipKey(helpKey)

	m = asModel(t, newM)
	if m.screen != screenHelp {
		t.Errorf("expected screenHelp after '? ?', got %v", m.screen)
	}

	// 5. Back returns to list
	m.screen = screenDetail
	escKey := keystroke("esc")
	newM, _ = m.detailKey(escKey)

	m = asModel(t, newM)
	if m.screen != screenList {
		t.Errorf("expected screenList after Esc from detail, got %v", m.screen)
	}
}

func TestDetailTaskViewRenderingEmptyAndFull(t *testing.T) {
	m, _ := testModel(t, 100, 30)

	// Empty task view
	m.screen = screenDetail
	m.detail = ""

	renderedEmpty := m.detailRows(20, 100)
	if len(renderedEmpty) == 0 {
		t.Error("expected detailRows on empty task to render something")
	}

	// Task with live run
	m = openOn(t, "ACME-2705")

	renderedLive := m.detailRows(20, 100)
	if len(renderedLive) == 0 {
		t.Error("expected detailRows on live task to render")
	}

	// SyncPanes on live task
	m = m.syncPanes()
	if len(m.panes) != int(tabCount) {
		t.Errorf("expected %d panes, got %d", tabCount, len(m.panes))
	}
}

func TestDetailWhyRefusalFormatting(t *testing.T) {
	m := openOn(t, "ACME-2662")
	tk, _ := m.task(m.detail)

	// Overview formatting
	lines := m.overviewLines()
	if len(lines) == 0 {
		t.Error("expected overviewLines to render")
	}

	// Reason with arguments
	r := view.Reason{Key: view.ReasonFailed, Args: []view.Arg{{Name: "phase", Value: "implement"}}}
	tk.Reason = r
	_ = m.drawRow(row{task: tk}, 100, true)
}

func TestDetailMidTaskDialChanges(t *testing.T) {
	m := openOn(t, "ACME-2662")

	// 1. Press 't' toggles thinking
	newM, _ := m.detailKey(keystroke("t"))

	m = asModel(t, newM)
	if m.knobs.Thinking == "" {
		t.Error("expected thinking to be updated after 't'")
	}

	// 2. Press 'E' cycles effort
	newM, _ = m.detailKey(keystroke("E"))

	m = asModel(t, newM)
	if m.knobs.Effort == "" {
		t.Error("expected effort to be updated after 'E'")
	}

	// 3. Press 'k' opens engines screen
	newM, _ = m.detailKey(keystroke("k"))

	mEngines := asModel(t, newM)
	if mEngines.screen != screenEngines {
		t.Errorf("expected screenEngines after 'k', got %v", mEngines.screen)
	}

	// 4. Press 'F' opens flows screen
	newM, _ = m.detailKey(keystroke("F"))

	mFlows := asModel(t, newM)
	if mFlows.screen != screenFlows {
		t.Errorf("expected screenFlows after 'F', got %v", mFlows.screen)
	}
}
