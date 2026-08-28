package store

// UpdateSettings, and the lock that makes the read and the write one step.
//
// The window and `orbit set` are meant to be used together — a switch flipped
// on screen while a terminal sets the engine is not a rare arrangement — and
// every setter over this file writes the whole of it back. Without the lock,
// the second writer's copy was read before the first writer's change existed,
// so one of the two settings was gone with both sides reporting success.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestTwoChangesAtOnceBothSurvive is the whole point of the lock: five
// changers, five different fields, and afterwards all five are there.
//
// The sleep inside each change is what makes the test about the code rather
// than about scheduling luck. It holds every changer between its read and its
// write for long enough that, without the lock, they overlap every time —
// what is left is whichever wrote last, and the other four fields are empty.
func TestTwoChangesAtOnceBothSurvive(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	changers := map[string]func(*Settings){
		"language": func(cfg *Settings) { cfg.Language = "es" },
		"engine":   func(cfg *Settings) { cfg.Engine = "codex" },
		"model":    func(cfg *Settings) { cfg.Model = "opus" },
		"flow":     func(cfg *Settings) { cfg.Flow = "careful" },
		"theme":    func(cfg *Settings) { cfg.Theme = "nord" },
	}

	var wg sync.WaitGroup

	failures := make(chan error, len(changers))

	for _, change := range changers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if err := s.UpdateSettings(func(cfg *Settings) error {
				change(cfg)
				time.Sleep(20 * time.Millisecond)

				return nil
			}); err != nil {
				failures <- err
			}
		}()
	}

	wg.Wait()
	close(failures)

	for err := range failures {
		t.Errorf("a change refused: %v", err)
	}

	got, err := s.Settings()
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}

	// UnreadCap is nobody's field here, and it is in the answer because the
	// defaults a never-written file yields have to survive five changes as
	// well as one.
	want := Settings{Language: "es", UnreadCap: defaultUnreadCap, Engine: "codex", Model: "opus", Flow: "careful", Theme: "nord"}
	if got != want {
		t.Errorf("settings are %+v, want %+v: a change made at the same time as another was lost", got, want)
	}
}

// TestAChangeThatFailsStillGivesTheLockBack. The change is the caller's code
// and it is allowed to refuse — an unknown key, a number that will not parse.
// A refusal that kept the lock would make one bad `orbit set` the last one
// this state root ever accepts.
func TestAChangeThatFailsStillGivesTheLockBack(t *testing.T) {
	root := t.TempDir()

	s, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	refusal := os.ErrInvalid
	if err := s.UpdateSettings(func(*Settings) error { return refusal }); !errors.Is(err, refusal) {
		t.Fatalf("UpdateSettings = %v, want the change's own refusal %v", err, refusal)
	}

	if _, err := os.Stat(filepath.Join(root, "settings.json"+lockSuffix)); !os.IsNotExist(err) {
		t.Errorf("the lock is still there after a change that failed: %v", err)
	}

	if err := s.UpdateSettings(func(cfg *Settings) error { cfg.Engine = "codex"; return nil }); err != nil {
		t.Errorf("the next change was refused: %v", err)
	}
}

// TestALockNobodyIsHoldingIsBroken. A process killed between taking the lock
// and giving it back leaves the file behind, and nothing else ever removes
// it. Without the break, the settings would stop accepting changes for good
// and the person it happened to would have no way to know what to delete.
func TestALockNobodyIsHoldingIsBroken(t *testing.T) {
	root := t.TempDir()

	s, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	lock := filepath.Join(root, "settings.json"+lockSuffix)
	if err := os.WriteFile(lock, nil, 0o600); err != nil {
		t.Fatalf("write the leftover lock: %v", err)
	}

	old := time.Now().Add(-2 * lockStale)
	if err := os.Chtimes(lock, old, old); err != nil {
		t.Fatalf("age the leftover lock: %v", err)
	}

	if err := s.UpdateSettings(func(cfg *Settings) error { cfg.Engine = "codex"; return nil }); err != nil {
		t.Fatalf("a lock nobody is holding stopped a change: %v", err)
	}

	got, err := s.Settings()
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}

	if got.Engine != "codex" {
		t.Errorf("Engine = %q, want codex", got.Engine)
	}
}

// TestALockSomebodyIsHoldingIsWaitedForAndThenNamed. The other half of the
// bargain: a lock that is not old enough to be leftover is respected, and
// when the wait runs out the reader is told which file to delete rather than
// being left with "could not save".
func TestALockSomebodyIsHoldingIsWaitedForAndThenNamed(t *testing.T) {
	root := t.TempDir()

	s, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	lock := filepath.Join(root, "settings.json"+lockSuffix)
	if err := os.WriteFile(lock, nil, 0o600); err != nil {
		t.Fatalf("write the held lock: %v", err)
	}

	started := time.Now()

	err = s.UpdateSettings(func(cfg *Settings) error { cfg.Engine = "codex"; return nil })
	if err == nil {
		t.Fatal("a change went through while another process held the lock")
	}

	if waited := time.Since(started); waited < lockPatience {
		t.Errorf("waited %v before refusing, want at least %v", waited, lockPatience)
	}

	if !strings.Contains(err.Error(), lock) {
		t.Errorf("the refusal does not say which file to delete: %v", err)
	}
}
