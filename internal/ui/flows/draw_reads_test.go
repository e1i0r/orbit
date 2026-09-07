package flows

import (
	"testing"
)

// countingFlowDir is a reader's own flow directory that counts how many
// times it was asked where it is.
//
// Every read of the flow directory goes through FlowDir — flow.List asks
// once, flow.Resolve asks once per flow — so the count is a faithful stand-in
// for "this touched the disk", without needing to watch syscalls.
type countingFlowDir struct {
	dir  string
	asks *int
}

func (c countingFlowDir) FlowDir() string {
	*c.asks++

	return c.dir
}

// TestDrawingTheFlowsScreenDoesNotReadTheDisk.
//
// flowsRows is called from View, so it runs on every frame. It called
// flow.List and then flow.Resolve once per flow, which is one os.ReadDir
// plus one os.ReadFile per flow — per frame, on the thread that draws, for
// as long as the screen is open. hitFlows did the same walk again on every
// mouse event, to work out where rows it had not drawn would land.
//
// Both now read what the screen read when it opened. The count below is
// taken after that reading and must not move, however many times the screen
// is drawn or pointed at.
func TestDrawingTheFlowsScreenDoesNotReadTheDisk(t *testing.T) {
	asks := 0

	_, e := designer(t)
	e.Flows = countingFlowDir{dir: t.TempDir(), asks: &asks}
	s := Open(FromBoard, e)

	if asks == 0 {
		t.Fatal("opening the screen has to read the flow directory at least once")
	}

	afterOpen := asks

	for range 5 {
		if rows := s.View(30, 100, e); len(rows) == 0 {
			t.Fatal("the flows screen drew nothing")
		}

		s.Hit(10, 8, e)
	}

	if asks != afterOpen {
		t.Errorf("the flow directory was read %d more times by drawing and pointing; a screen draws what it has already read", asks-afterOpen)
	}
}

// TestSavingAFlowShowsItWithoutReopeningTheScreen is the other half of
// reading once: a screen that holds its own copy has to refresh it when it
// is the one that changed the directory.
func TestSavingAFlowShowsItWithoutReopeningTheScreen(t *testing.T) {
	_, e := designer(t)
	e.Flows = flowsTestDir(t.TempDir())
	s := Open(FromBoard, e)
	s = s.startCreateFlow(e).OnFields()
	s.flowName = "recien-guardado"

	m2, _ := s.saveCustomFlow(e)

	found := false

	for _, d := range m2.listed {
		if d.Name == "recien-guardado" {
			found = true
		}
	}

	if !found {
		t.Error("the flow just saved is not in what the screen is showing")
	}
}
