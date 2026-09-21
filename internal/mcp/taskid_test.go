package mcp

// The id a task gets when the caller did not name one.
//
// A model calling this has no idea what ids are already in the record, so
// the whole of the id is decided here — and an id that collides is a call
// that fails with "already exists" about something the caller never saw.

import "testing"

// TestARepositoryNameBecomesAPrefix.
//
// Upper case, and everything that is not a letter or a digit dropped. The
// two runs are read at both ends, because a reading one character short of
// them would drop the Z out of `xyz` and the 9 out of `app9` — a prefix
// nobody would notice was wrong until two repositories minted the same ids.
func TestARepositoryNameBecomesAPrefix(t *testing.T) {
	for _, one := range []struct {
		why  string
		name string
		want string
	}{
		{"an ordinary name", "payments", "PAYMENTS"},
		{"the ends of the alphabet survive", "az", "AZ"},
		{"and the ends of the digits", "app09", "APP09"},
		{"a dash is not a letter", "orbit-web", "ORBITWEB"},
		{"nor is anything else around them", "a@z", "AZ"},
		{"nor a bracket or a colon", "a[z", "AZ"},
		{"a name with nothing usable in it falls back", "---", "TASK"},
		{"and so does an empty one", "", "TASK"},
	} {
		if got := idPrefix(one.name); got != one.want {
			t.Errorf("idPrefix(%q) = %q, want %q — %s", one.name, got, one.want, one.why)
		}
	}
}

// TestWhatReadsAsAnIdThisPackageMinted.
//
// strconv accepts a sign and this must not: `ACME--1` and `ACME-+1` are not
// ids anything here has ever written, and reading one as a number would let
// whatever is in the record decide what the next id is. The digits are read
// at both ends for the same reason the prefix is.
func TestWhatReadsAsAnIdThisPackageMinted(t *testing.T) {
	for _, one := range []struct {
		id   string
		want int
		is   bool
	}{
		{"ACME-7", 7, true},
		{"ACME-0", 0, true},
		{"ACME-1234567890", 1234567890, true},
		{"ACME--1", 0, false},
		{"ACME-+1", 0, false},
		{"ACME-1a", 0, false},
		{"ACME-a1", 0, false},
		{"ACME-", 0, false},
		{"ACME-1.0", 0, false},
		{"ACME-1/2", 0, false},
		{"ACME-1:2", 0, false},
		{"OTHER-7", 0, false},
		{"ACME7", 0, false},
		// Past what an int holds, which is what a directory somebody named
		// by hand can be. Out of range is not a number this package minted.
		{"ACME-99999999999999999999", 0, false},
	} {
		got, is := suffixNumber(one.id, "ACME")
		if is != one.is || got != one.want {
			t.Errorf("suffixNumber(%q) = (%d, %v), want (%d, %v)", one.id, got, is, one.want, one.is)
		}
	}
}
