package cli

// What a relay is told about an engine's allowance.

import (
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/quota"
)

// TestAWindowNobodyReadIsNotAWindowWithNothingLeft is the whole reason this
// port has three answers and not two. Three of the four engines Orbit can run
// have never reported a number, and a relay that read silence as empty would
// refuse to hand work to any of them.
func TestAWindowNobodyReadIsNotAWindowWithNothingLeft(t *testing.T) {
	if got := fromWindows(nil); !got.Free {
		t.Errorf("an engine that has not answered reads as spent: %+v", got)
	}
}

func TestWhatWindowsSayAboutHandingAnEngineWork(t *testing.T) {
	for _, c := range []struct {
		name    string
		windows []quota.Window
		free    bool
		back    time.Duration
	}{
		{
			name:    "there is room",
			windows: []quota.Window{{Pct: 40, ResetsIn: time.Hour}},
			free:    true,
		},
		{
			name: "one window of two is full",
			// A provider that reports an hour and a week refuses the
			// request when either of them is full, so either is enough.
			windows: []quota.Window{
				{Pct: 100, ResetsIn: 20 * time.Minute},
				{Pct: 3, ResetsIn: 100 * time.Hour},
			},
			back: 20 * time.Minute,
		},
		{
			name: "both are full, and the nearer one is the answer",
			windows: []quota.Window{
				{Pct: 100, ResetsIn: 5 * time.Hour},
				{Pct: 140, ResetsIn: 30 * time.Minute},
			},
			back: 30 * time.Minute,
		},
		{
			name:    "full, and nobody said when it comes back",
			windows: []quota.Window{{Pct: 100}},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := fromWindows(c.windows)
			if got.Free != c.free || got.Back != c.back {
				t.Errorf("fromWindows = %+v, want free=%v back=%v", got, c.free, c.back)
			}
		})
	}
}

// TestNoMeterIsNoPort. internal/task reads a nil Allowance as every engine
// being free, which is what every run had before relays existed.
func TestNoMeterIsNoPort(t *testing.T) {
	if allowancePort(nil) != nil {
		t.Error("a machine with no meter handed the run a port that answers nothing")
	}
}
