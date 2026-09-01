package store

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestTheRecordIsOpenedOnceAndHandedOutAgain(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer closed(t, s)

	first, err := s.Record()
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	second, err := s.Record()
	if err != nil {
		t.Fatalf("Record again: %v", err)
	}

	if first != second {
		t.Error("the second call opened a second handle, so this process is two writers")
	}
}

func TestAskingForTheRecordFromTwoGoroutinesOpensItOnce(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer closed(t, s)

	// The window refreshes on one goroutine and rescans on another, so this
	// is the real first call and not a contrived one. Under -race a handle
	// guarded by nothing fails here.
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		seen = map[any]bool{}
	)

	for range 8 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			d, err := s.Record()

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				t.Errorf("Record: %v", err)

				return
			}

			seen[d] = true
		}()
	}

	wg.Wait()

	if len(seen) != 1 {
		t.Errorf("%d handles were opened, want 1", len(seen))
	}
}

func TestTheRecordIsNotOpenedUntilItIsAskedFor(t *testing.T) {
	root := t.TempDir()

	s, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer closed(t, s)

	// Opening a store is what every command does, including the ones that
	// only want a path. A file minted there would make `orbit repos` write
	// to the state root, which is the rule the package doc is about.
	if _, err := os.Stat(filepath.Join(root, "orbit.db")); !os.IsNotExist(err) {
		t.Fatalf("stat orbit.db before Record: %v, want it not to be there", err)
	}

	if _, err := s.Record(); err != nil {
		t.Fatalf("Record: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "orbit.db")); err != nil {
		t.Errorf("stat orbit.db after Record: %v", err)
	}
}

func TestClosingAStoreThatNeverOpenedTheRecordClosesNothing(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := s.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestClosingTwiceIsNotAFailure(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := s.Record(); err != nil {
		t.Fatalf("Record: %v", err)
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Commands defer this at the top and there is more than one way out of
	// one of those.
	if err := s.Close(); err != nil {
		t.Errorf("Close again: %v", err)
	}
}

func TestAStoreClosedAndAskedAgainOpensTheRecordAgain(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer closed(t, s)

	first, err := s.Record()
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	second, err := s.Record()
	if err != nil {
		t.Fatalf("Record after Close: %v", err)
	}

	if first == second {
		t.Error("the handle survived Close, so it is being used after it was closed")
	}
}

// closed shuts the record down at the end of a test and says so if it could
// not, which on this path means a file left open under a temporary root.
func closed(t *testing.T, s *Store) {
	t.Helper()

	if err := s.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}
