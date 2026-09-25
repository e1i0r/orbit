package ui

import (
	"errors"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// TestAnAutopilotPassIsAVerbOutOnEachTask. The pass is written onto every
// task it is about as an AUTOPILOT verb, the way a delivery key is, so its
// steps are drawn on the task and a pass whose window died is closed as
// broken; and the answer, or the error, closes it on each.
func TestAnAutopilotPassIsAVerbOutOnEachTask(t *testing.T) {
	m, _ := testModel(t, 100, 30)
	m.opts.Settings = settingsWith(true, "en", 9)
	m.opts.AutoSupervise = func(string, []view.Task) (string, error) { return "", nil }

	wrote := map[string][]Delivery{}
	m.opts.RecordDeliver = func(t view.Task, d Delivery) error {
		wrote[t.ID] = append(wrote[t.ID], d)

		return nil
	}

	next, cmd := m.autoSuperviseNeedsYou()
	if cmd == nil || len(wrote) == 0 {
		t.Fatalf("autopilot asked %v and wrote %v", cmd != nil, wrote)
	}

	var pass []view.Task

	for id, ds := range wrote {
		if len(ds) != 1 || ds[0].Verb != autopilotVerb || ds[0].Done {
			t.Errorf("%s was written %+v, want one AUTOPILOT verb out", id, ds)
		}

		pass = append(pass, view.Task{ID: id})
	}

	broke := errors.New("the engine went away")
	next.Update(supervisorReplyMsg{Err: broke, Autopilot: pass})

	for id, ds := range wrote {
		last := ds[len(ds)-1]
		if !last.Done || last.Verb != autopilotVerb || !errors.Is(last.Failure, broke) {
			t.Errorf("%s ends %+v, want the AUTOPILOT verb closed with the error", id, last)
		}
	}
}
