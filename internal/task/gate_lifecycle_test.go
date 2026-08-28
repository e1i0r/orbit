package task

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

func TestFileGateComprehensive(t *testing.T) {
	root := t.TempDir()

	s, err := store.New(root)
	if err != nil {
		t.Fatal(err)
	}

	r := repo.Repo{Path: filepath.Join(t.TempDir(), "repo"), Name: "repo"}

	tk, err := Create(s, r, "TASK-GATE", "Gate testing", "quick")
	if err != nil {
		t.Fatal(err)
	}

	gate := FileGate(s, 10*time.Millisecond).(fileGate) //nolint:errcheck
	p := flow.Phase{Name: "plan", Wait: false}

	// 1. Context already cancelled before gate
	cancCtx, cancel := context.WithCancel(context.Background())
	cancel()

	action, err := gate.Before(cancCtx, tk, p, 0)
	if err != nil || action != Stop {
		t.Errorf("Before on cancelled context = (%v, %v), want Stop", action, err)
	}

	// 2. Word cancel in control file
	if err := Control(s, tk, "cancel"); err != nil {
		t.Fatal(err)
	}

	action, err = gate.Before(context.Background(), tk, p, 0)
	if err != nil || action != Stop {
		t.Errorf("Before on cancel control = (%v, %v), want Stop", action, err)
	}

	// 3. Word skip in control file
	if err := Control(s, tk, "skip"); err != nil {
		t.Fatal(err)
	}

	action, err = gate.Before(context.Background(), tk, p, 0)
	if err != nil || action != Skip {
		t.Errorf("Before on skip control = (%v, %v), want Skip", action, err)
	}

	// 4. Word resume in control file (consumed and continues)
	if err := Control(s, tk, "resume"); err != nil {
		t.Fatal(err)
	}

	action, err = gate.Before(context.Background(), tk, p, 0)
	if err != nil || action != Continue {
		t.Errorf("Before on resume control = (%v, %v), want Continue", action, err)
	}

	// 5. Wait with wordPause released by resume
	if err := Control(s, tk, "pause"); err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(20 * time.Millisecond)

		_ = Control(s, tk, "resume") //nolint:errcheck
	}()

	action, err = gate.Before(context.Background(), tk, p, 0)
	if err != nil || action != Continue {
		t.Errorf("Before on pause released by resume = (%v, %v), want Continue", action, err)
	}

	// 6. Wait with p.Wait=true released by skip
	pWait := flow.Phase{Name: "review", Wait: true}
	// disable autopilot
	if err := s.SaveSettings(store.Settings{Autopilot: false}); err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(20 * time.Millisecond)

		_ = Control(s, tk, "skip") //nolint:errcheck
	}()

	action, err = gate.Before(context.Background(), tk, pWait, 0)
	if err != nil || action != Skip {
		t.Errorf("Before on p.Wait released by skip = (%v, %v), want Skip", action, err)
	}

	// 7. Wait with p.Wait=true released by autopilot setting flipped
	go func() {
		time.Sleep(20 * time.Millisecond)

		_ = s.SaveSettings(store.Settings{Autopilot: true}) //nolint:errcheck
	}()

	action, err = gate.Before(context.Background(), tk, pWait, 0)
	if err != nil || action != Continue {
		t.Errorf("Before on p.Wait released by autopilot = (%v, %v), want Continue", action, err)
	}
}

// TestBeforeErrorPaths covers Before's two error returns: take failing, and
// autopilot failing once take answered with nothing to act on.
func TestBeforeErrorPaths(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "GATE-BEFORE-ERR", "gate before error test", "quick")
	if err != nil {
		t.Fatal(err)
	}

	gate := FileGate(s, 10*time.Millisecond).(fileGate) //nolint:errcheck
	p := flow.Phase{Name: "plan", Wait: false}

	// 1. take fails: a directory sitting where the control file belongs.
	controlPath, err := s.ControlPath(r.Path, tk.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Mkdir(controlPath, 0o700); err != nil {
		t.Fatal(err)
	}

	if _, err := gate.Before(context.Background(), tk, p, 0); err == nil {
		t.Error("Before should have failed when take cannot read the control word")
	}

	if err := os.Remove(controlPath); err != nil {
		t.Fatal(err)
	}

	// 2. autopilot fails: settings.json is a directory, and no control word
	// is in the way to short-circuit the switch check.
	settingsPath := filepath.Join(s.Root(), "settings.json")
	if err := os.Mkdir(settingsPath, 0o700); err != nil {
		t.Fatal(err)
	}

	if _, err := gate.Before(context.Background(), tk, p, 0); err == nil {
		t.Error("Before should have failed when the autopilot switch cannot be read")
	}
}

// TestWaitErrorPaths calls fileGate.wait directly, which is how the emit and
// take failures inside its own loop are reached without racing a poll.
func TestWaitErrorPaths(t *testing.T) {
	s, r := fixture(t)
	gate := FileGate(s, 10*time.Millisecond).(fileGate) //nolint:errcheck
	p := flow.Phase{Name: "review"}

	// 1. The opening phase.waiting write fails: a bad id.
	bad := Task{ID: "has/slash", Repo: r}
	if _, err := gate.wait(context.Background(), bad, p, whyFlow, false); err == nil {
		t.Error("wait should have failed when phase.waiting cannot be recorded")
	}

	// 2. take fails inside the loop: phase.waiting is written fine, but the
	// control file is a directory.
	tk, err := Create(s, r, "GATE-WAIT-ERR-1", "gate wait error test", "quick")
	if err != nil {
		t.Fatal(err)
	}

	controlPath, err := s.ControlPath(r.Path, tk.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Mkdir(controlPath, 0o700); err != nil {
		t.Fatal(err)
	}

	if _, err := gate.wait(context.Background(), tk, p, whyFlow, false); err == nil {
		t.Error("wait should have failed when take cannot read the control word")
	}

	if err := os.Remove(controlPath); err != nil {
		t.Fatal(err)
	}

	// 3. autopilot fails inside the loop: no control word waiting, so the
	// switch is checked and the settings file cannot be read.
	tk2, err := Create(s, r, "GATE-WAIT-ERR-2", "gate wait error test 2", "quick")
	if err != nil {
		t.Fatal(err)
	}

	settingsPath := filepath.Join(s.Root(), "settings.json")
	if err := os.Mkdir(settingsPath, 0o700); err != nil {
		t.Fatal(err)
	}

	if _, err := gate.wait(context.Background(), tk2, p, whyFlow, false); err == nil {
		t.Error("wait should have failed when the autopilot switch cannot be read")
	}
}

// TestWaitResumedEmitFailure covers wait's remaining error return: the word
// that releases it was taken cleanly, but recording that release fails.
func TestWaitResumedEmitFailure(t *testing.T) {
	s, r := fixture(t)

	tk, err := Create(s, r, "GATE-WAIT-RESUME-ERR", "gate wait resumed error test", "quick")
	if err != nil {
		t.Fatal(err)
	}

	eventsPath, err := s.EventsPath(r.Path, tk.ID)
	if err != nil {
		t.Fatal(err)
	}

	gate := FileGate(s, 5*time.Millisecond).(fileGate) //nolint:errcheck
	p := flow.Phase{Name: "review"}

	synctest.Test(t, func(t *testing.T) {
		done := make(chan error, 1)

		go func() {
			_, err := gate.wait(context.Background(), tk, p, whyFlow, false)
			done <- err
		}()
		// wait returns from synctest.Wait once phase.waiting is written and
		// the goroutine is parked on its first poll — at which point the log
		// is made unwritable before the word that would release it arrives.
		synctest.Wait()

		if err := os.Chmod(eventsPath, 0o400); err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() { _ = os.Chmod(eventsPath, 0o600) }) //nolint:errcheck

		if err := Control(s, tk, "resume"); err != nil {
			t.Fatal(err)
		}

		if err := <-done; err == nil {
			t.Error("wait should have failed when phase.resumed could not be recorded")
		}
	})
}
