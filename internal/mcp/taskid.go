package mcp

// Minting the id of a task nobody named, out of the repository's name and
// the highest number the record already carries in that shape.

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// nextTaskID mints an id for a task nobody named.
//
// The shape is the repository's name upper-cased and a number, which is what
// a reader typing `orbit new -id` writes by hand, and the number is one past
// the highest anything in the record already carries in that shape. It is
// checked against store.ValidTaskID before it is returned, so a repository
// whose name is not a legal id fragment is refused here rather than at the
// write.
//
// The whole record and not this repository's tasks: an id names one task in
// the state root, so counting only one checkout's meant a second checkout of
// the same name minted PAYMENTS-1 for ever, and every call failed with
// "task PAYMENTS-1 already exists" until the caller named one by hand.
func nextTaskID(s *store.Store, r repo.Repo) (string, error) {
	prefix := idPrefix(r.Name)

	existing, err := everyTaskID(s)
	if err != nil {
		return "", fmt.Errorf("list the tasks already in %s: %w", r.Name, err)
	}

	highest := 0

	for _, id := range existing {
		n, ok := suffixNumber(id, prefix)
		if ok && n > highest {
			highest = n
		}
	}

	id := fmt.Sprintf("%s-%d", prefix, highest+1)
	if err := store.ValidTaskID(id); err != nil {
		return "", fmt.Errorf("an id built from repository %q is not usable: %w", r.Name, err)
	}

	return id, nil
}

// idPrefix turns a repository name into the leading fragment of an id: upper
// case, and everything that is not a letter or a digit dropped. A name with
// nothing usable in it falls back to TASK, which is a poor prefix and a
// legal one.
func idPrefix(name string) string {
	var b strings.Builder

	for _, r := range strings.ToUpper(name) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}

	if b.Len() == 0 {
		return "TASK"
	}

	return b.String()
}

// suffixNumber reads the number off an id this package would have minted,
// and says so when the id is not one.
//
// The digits are counted by strconv rather than by multiplying through them,
// which is what this did and is where it went wrong: a directory named
// ORB-99999999999999999999 ran an int past its width and came back as some
// unrelated number, so nextTaskID took that for the highest id in the
// repository and minted its successor. Out of range is not a number this
// package minted, and the answer to that is no.
func suffixNumber(id, prefix string) (int, bool) {
	rest, ok := strings.CutPrefix(id, prefix+"-")
	if !ok {
		return 0, false
	}
	// strconv accepts a sign and this must not: ORB--1 and ORB-+1 are not
	// ids this package has ever written.
	for _, c := range rest {
		if c < '0' || c > '9' {
			return 0, false
		}
	}

	n, err := strconv.Atoi(rest)
	if err != nil {
		return 0, false
	}

	return n, true
}

// everyTaskID is the id of every task the record holds, however many
// repositories each one is worked in.
func everyTaskID(s *store.Store) ([]string, error) {
	d, err := s.Record()
	if err != nil {
		return nil, err
	}

	worked, err := d.TasksAndRepos()
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(worked))

	for _, w := range worked {
		if !slices.Contains(ids, w.Task) {
			ids = append(ids, w.Task)
		}
	}

	return ids, nil
}
