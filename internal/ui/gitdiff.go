package ui

// gitdiff.go is how the diff tab asks git for an answer: running the
// subprocess, waiting for it, and deciding how long is too long to wait.
//
// It is a file of its own, split out of diff.go, along the seam between two
// different questions. diff.go answers "what does this line of the diff
// mean" once an answer exists; this file answers "how do we get one, and
// what do we do if git never comes back". Splitting there — rather than
// growing diff.go past the 300-line ceiling once fileAt's multi-file fix and
// diffLines' third state landed in it — keeps each file answerable by
// reading it alone.

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/view"
)

// gitDiffTimeout bounds one git diff. Five seconds is longer than any
// ordinary repository takes and short enough that a hang surfaces within the
// sitting a reader opened the pane for, rather than at the end of it. It is
// also the reason a slower cadence than the log's is defensible for the
// retry this file's diffOf is put on in ui.go: a poll that can cost up to
// five seconds has no business running on the log's half-second clock.
//
// It is a var and not a const because gitrepo_test.go shrinks it to put a
// fake git that never answers under test without the suite actually
// waiting five real seconds to see the bound hit. Nothing in production
// writes it; the two tests that do save and restore it under t.Cleanup, and
// that is safe for exactly one reason — internal/ui has no t.Parallel
// anywhere in it, so no other test can be reading this while one of them
// holds it. Whoever adds the first parallel test to this package is the
// person that sentence is addressed to: it would be a real data race, since
// the value is read on a Cmd goroutine.
var gitDiffTimeout = 5 * time.Second

// baseTimeout bounds boundedBaseOf's own wait. What it is waiting on is two
// rev-parses and a remote listing rather than a walk of the working tree, so
// it is given less patience than the diff itself: a base that has not
// answered in two seconds is one gitDiff should stop waiting on and fall
// back from, well before the diff's own five have run out.
//
// Also a var, and for the same reason as gitDiffTimeout above.
var baseTimeout = 2 * time.Second

// errGitTimedOut is what runGitDiff reports when gitDiffTimeout is hit, so
// the pane can say a bound was hit rather than repeat exec's own words —
// "signal: killed" — for a process that is, as far as this program can
// prove, still running somewhere.
var errGitTimedOut = errors.New("git did not answer in time")

// diffOf runs git diff in the task's worktree, against the branch the
// repository's work is measured from.
//
// The worktree is asked for through the port rather than worked out here.
// Where a task's checkout lives is internal/store's answer — it is a hash of
// the repository's path under the state root — and internal/ui may not name
// that package. Running the diff in the repository instead, which is what
// this did before the tab that draws it existed, shows the reader whatever
// they happen to have uncommitted in their own checkout under the heading of
// an agent's task.
func diffOf(r Reader, t view.Task, base baseRef) tea.Cmd {
	return func() tea.Msg {
		if r == nil {
			return diffMsg{ID: t.ID, Err: errors.New("this window was opened without a way to find the worktree")}
		}
		if t.RepoPath == "" {
			return diffMsg{ID: t.ID, Err: errors.New("this task does not say where its repository is")}
		}
		dir, err := r.Worktree(t.RepoPath, t.ID)
		if err != nil {
			return diffMsg{ID: t.ID, Err: err}
		}
		if !base.known {
			base = boundedBaseOf(t.RepoPath)
		}
		out, noBase, err := gitDiff(dir, base.name)
		if err != nil {
			return diffMsg{ID: t.ID, Tree: dir, Err: err, Base: base}
		}
		// A base that never answered is not a repository without a base,
		// and the strip may only say the second. gitDiff fell back either
		// way — the diff on screen is the plain working tree and is true —
		// but labelling it would be asserting something about the
		// repository that this program timed out before observing, which is
		// the rule the pending state upstairs exists to keep.
		return diffMsg{ID: t.ID, Tree: dir, Text: out, NoBase: noBase && !base.timedOut, Base: base}
	}
}

// gitDiff asks git twice at most: once against the base branch, and once for
// the working tree alone if that failed.
//
// The fallback is not a way of hiding an error. A worktree cut before its
// base existed, or one whose base has been deleted since, still has changes
// worth showing, and refusing to show them because the comparison is
// unavailable would be refusing the reader the only view they have. If the
// plain diff fails too, that failure is the one reported — it is the one
// that says the worktree itself is not there.
//
// The bool says whether the text is that plain fallback rather than the
// comparison against base — true whenever base was empty, or the merge-base
// diff itself failed. A timeout is not one of those: it returns above,
// without a fallback and without a text, so the bool never speaks for it.
// A reader cannot tell "there is no committed work" from "there was nothing
// to compare the commits against" by reading the diff's text alone; the
// tabStrip is what says which, and this is where that answer starts.
func gitDiff(dir, base string) (string, bool, error) {
	if base != "" {
		out, err := runGitDiff(dir, "diff", "--merge-base", base)
		if err == nil {
			return out, false, nil
		}
		if errors.Is(err, errGitTimedOut) {
			// A second git diff, on the same worktree, is not a fresh
			// chance — it is the same hang again. Reporting the timeout
			// directly is more honest than a retry that is likely to sit
			// for another five seconds before it, too, gives up.
			return "", false, err
		}
	}
	out, err := runGitDiff(dir, "diff")
	if err != nil {
		return "", false, err
	}
	return out, true, nil
}

// runGitDiff runs one git diff, bounded by gitDiffTimeout. CombinedOutput is
// what git said; err alone is "exit status 128" or "signal: killed", neither
// of which is an answer a reader can act on, so the bytes travel with it.
func runGitDiff(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitDiffTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil && ctx.Err() != nil {
		return "", fmt.Errorf("%w: git %s in %s", errGitTimedOut, strings.Join(args, " "), dir)
	}
	if err != nil {
		return "", fmt.Errorf("git %s in %s: %w: %s", strings.Join(args, " "), dir, err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// baseOf is the branch a repository's work is measured against, or nothing
// when it cannot be read. Nothing is a usable answer — gitDiff falls back to
// the working tree — so a repository that is detached, or gone, costs the
// comparison and not the pane.
func baseOf(repoPath string) string {
	r, err := repo.Open(repoPath)
	if err != nil {
		return ""
	}
	return r.Base
}

// baseRef is a base branch and whether anybody has been asked for it yet.
//
// It exists because the lookup is the expensive, uncancellable end of this
// file — three git subprocesses through a helper that takes no context — and
// the answer does not change between two ticks of a clock. So it is looked
// up once, when the task view opens, and rides back on the diffMsg to the
// Model, which hands it to every rescan that follows. The zero value is
// "nobody has asked yet", which is the state openDetail starts a view in and
// the only state that spends anything.
type baseRef struct {
	// name is the branch to measure against, or "" when there is none: a
	// detached checkout, or a repository that could not be read at all.
	name string
	// known is whether the lookup has happened. False means ask.
	known bool
	// timedOut is whether the lookup gave up rather than answered. It is
	// not the same fact as "there is no base branch", and diffOf keeps the
	// two apart: the tab strip may say a repository has no base only when
	// git actually said so.
	timedOut bool
}

// boundedBaseOf is baseOf with a deadline of its own, and a soft one rather
// than a killing one.
//
// repo.Open shells out three times — rev-parse --show-toplevel, the remote
// pick, rev-parse --abbrev-ref HEAD — through internal/repo's own git()
// helper, which takes no context.Context and has no way to be told to stop.
// Threading one through it would mean widening Open, pickRemote,
// AddWorktree, RemoveWorktree, hasBranch and Discover — every call site in a
// package this task did not otherwise touch — to bound a read this function
// only ever uses to decide gitDiff's second argument. What is bounded here
// instead is the wait, not the call: the real read keeps running on its own
// goroutine regardless of the deadline, and this function answers the moment
// baseTimeout passes.
//
// The disclosed cost is that a repository whose git never returns leaves the
// goroutine, and the git process under it, running until it does. What
// bounds that is the caller, and only the caller: diffOf asks exactly once
// per opened task view, because the answer it gets is carried on the diffMsg
// and handed back to every rescan afterwards. One goroutine per time a
// reader opens a diff on a hung repository is a leak that stops growing when
// they stop opening it. Once per two-second tick — which is what this cost
// before the base was carried — was not.
func boundedBaseOf(repoPath string) baseRef {
	got := make(chan string, 1)
	go func() { got <- baseOf(repoPath) }()
	select {
	case name := <-got:
		return baseRef{name: name, known: true}
	case <-time.After(baseTimeout):
		return baseRef{known: true, timedOut: true}
	}
}
