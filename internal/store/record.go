package store

// The record's handle. The state root holds one SQLite file and this is
// where it is opened, once, for as long as the process lives.

import "github.com/e1i0r/orbit/internal/db"

// Record is the open record, opened on the first call and the same handle
// on every one after it.
//
// One process gets one handle, and that is the point rather than a saving.
// db.Open pins each handle to a single connection so that one process is
// one writer; two handles in one process would be two writers contending
// for the one write lock SQLite has, which is a queue behind a queue and
// buys nothing. Hanging it off the store is what makes "one" true without
// a package-level variable or a handle threaded through every signature in
// internal/task.
//
// Opening it here does mean this package now knows what the record is made
// of, where before it only knew where things were. That is the trade: the
// alternative is every caller opening its own, which is exactly the shape
// the paragraph above rules out.
func (s *Store) Record() (*db.DB, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.record != nil {
		return s.record, nil
	}

	d, err := db.Open(s.DBPath())
	if err != nil {
		return nil, err
	}

	s.record = d

	return s.record, nil
}

// Close lets go of the record, and a store that never opened it closes
// nothing.
//
// It is safe to call twice, because the callers that have to call it are
// deferred at the top of a command and there is more than one way out of
// one of those.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.record == nil {
		return nil
	}

	d := s.record
	s.record = nil

	return d.Close()
}
