package mcp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/view"
)

// Session is the Orbit state one MCP server acts on.
//
// It is a value the command line builds and hands over, rather than a set of
// package-level functions reading the environment where they stand, because
// where to look for the board is the one thing a client gets wrong. A desktop
// application spawns `orbit mcp` with a working directory of / or of its own
// bundle; a server that scanned the working directory would answer every
// question with an empty board, and be believed.
type Session struct {
	// Root is where repositories are looked for. Empty means the ones the
	// state root already has a record of — every repository Orbit has
	// written a task against — which is the answer that does not depend on
	// which directory a client happened to spawn this process in.
	Root string
	// Version is the orbit release, for the initialize handshake. It is
	// passed in because internal/cli owns it and this package must not
	// import internal/cli.
	Version string
}

// open resolves the state root. Every tool call opens it rather than the
// server holding one open, because a server started before `orbit` had ever
// run must not be the reason the first task cannot be found.
func (sn Session) open() (*store.Store, error) {
	s, err := store.Open()
	if err != nil {
		return nil, fmt.Errorf("open the state root: %w", err)
	}

	return s, nil
}

// board folds every record under every root this session looks in, and says
// which roots those were: a client that finds an empty board can then tell
// "nothing to do" from "looked in the wrong place", which is the failure a
// server spawned by a desktop application actually has. The list comes back
// from here rather than being asked for again afterwards, so that what a
// tool reports having looked in is what it did look in.
//
// One board.Reader is built per root and the results are merged, rather than
// a single reader over some common ancestor: the repositories Orbit knows
// are scattered wherever the reader keeps them, and the nearest directory
// containing all of them is routinely the home directory — a walk that costs
// minutes and turns up every checkout the reader has ever cloned.
func (sn Session) board(s *store.Store) (board.Board, []string, error) {
	roots, err := sn.roots(s)
	if err != nil {
		return board.Board{}, nil, err
	}

	var (
		merged board.Board
		errs   []error
	)

	for _, root := range roots {
		r := board.NewReader(s, root)
		if err := r.Rescan(); err != nil {
			errs = append(errs, err)
			continue
		}

		b, _, err := r.Refresh()
		if err != nil {
			errs = append(errs, err)
			continue
		}

		merged = mergeBoards(merged, b)
	}
	// A root that would not be read is a shorter board, not a failed one —
	// the same treatment store.Repos gives a damaged repository directory.
	// Every root failing is different: there is no board at all, and saying
	// "no tasks" would be a lie about a directory nobody could look in.
	if len(merged.RepoList) == 0 && len(errs) > 0 {
		return board.Board{}, roots, errors.Join(errs...)
	}

	slices.SortFunc(merged.Tasks, func(a, b view.Task) int {
		if c := strings.Compare(a.Repo, b.Repo); c != 0 {
			return c
		}

		return strings.Compare(a.ID, b.ID)
	})

	return merged, roots, nil
}

// roots is every directory this session walks for repositories.
func (sn Session) roots(s *store.Store) ([]string, error) {
	if sn.Root != "" {
		abs, err := filepath.Abs(sn.Root)
		if err != nil {
			return nil, fmt.Errorf("resolve %q: %w", sn.Root, err)
		}

		return []string{abs}, nil
	}
	// Repos reports damage alongside the list it did manage to read, so a
	// non-nil error with repositories in hand is a shorter list rather than
	// a failure, and only an empty list makes the error the whole answer.
	refs, err := s.Repos()
	if len(refs) == 0 {
		if err != nil {
			return nil, fmt.Errorf("list the repositories orbit knows: %w", err)
		}
		// Nothing has ever had a task written against it. The working
		// directory is the only place left to look, and for a server
		// spawned from a terminal it is the right one.
		wd, wdErr := os.Getwd()
		if wdErr != nil {
			return nil, fmt.Errorf("resolve the working directory: %w", wdErr)
		}

		return []string{wd}, nil
	}

	paths := make([]string, 0, len(refs))
	for _, ref := range refs {
		paths = append(paths, ref.Path)
	}

	return paths, nil
}

// mergeBoards folds one root's board into the running total.
//
// Health is deliberately not merged. It is one root's measurement of the
// state root, and four copies of it averaged into one number would be a
// figure no reader could trace back to anything — so no tool reports it
// rather than every tool reporting an invention.
func mergeBoards(into, b board.Board) board.Board {
	into.Tasks = append(into.Tasks, b.Tasks...)
	into.RepoList = append(into.RepoList, b.RepoList...)

	into.Repos += b.Repos
	for i, n := range b.Counts {
		into.Counts[i] += n
	}

	into.Errs = append(into.Errs, b.Errs...)
	if b.ReadAt.After(into.ReadAt) {
		into.ReadAt = b.ReadAt
	}

	return into
}
