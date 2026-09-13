package task

// Where the work was when somebody said a rule in the middle of it.

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// workedIn is the folder every file the task has changed sits under, and
// empty when there is no one folder that holds them all.
//
// It is not a reading of what the sentence meant. Nothing here looks at the
// words: this is where the person was working at the moment they said it,
// which is the one thing about the place that is known rather than guessed
// at. A rule said while three files under internal/db were open is about
// internal/db often enough that making somebody type it again is the small
// tax that ends with nobody placing rules at all — and whoever keeps the
// sentence can say somewhere else, or say the whole checkout with a dot.
//
// The task's own checkout and not every repository it has joined. A path is
// only a place once you know which repository it is inside, and a task
// spanning three of them has three answers to that — so the one the sentence
// already carries is the only one that can be written down.
func workedIn(s *store.Store, t Task) string {
	if t.Repo.Path == "" {
		return ""
	}

	r, err := repo.Open(t.Repo.Path)
	if err != nil {
		return ""
	}

	wt, err := s.WorktreeDir(r.Path, t.ID)
	if err != nil {
		return ""
	}

	changes, err := r.WorktreeChanges(wt)
	if err != nil {
		return ""
	}

	return thereToo(r.Path, commonFolder(taskWrote(changes)))
}

// commonFolder is the deepest folder that holds every file in the list.
//
// Folders and never one of the files, even when the list is one file long. A
// rule about exactly one file is a precision a person means and says; what
// can be read off a diff is the folder the work was in, and answering with
// the file would file half the rules there are somewhere they reach nothing
// else.
func commonFolder(changes []repo.Change) string {
	if len(changes) == 0 {
		return ""
	}

	shared := strings.Split(path.Dir(changes[0].Path), "/")
	for _, c := range changes[1:] {
		shared = sharedHead(shared, strings.Split(path.Dir(c.Path), "/"))
	}

	// The root is the whole checkout, which is what an empty answer already
	// means — and a rule filed on "." would be a second spelling of it.
	if where := path.Join(shared...); where != "." {
		return where
	}

	return ""
}

// sharedHead is how much of two paths is the same reading from the left.
func sharedHead(a, b []string) []string {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}

	return a[:n]
}

// thereToo drops a folder that is only in the worktree.
//
// The work happens on a branch and the rule is filed against the checkout
// that branch came from, so a folder the task has just created exists where
// the work is and nowhere the rule could point at. Offering it would put a
// refusal in front of somebody who typed nothing — and the whole checkout,
// which is what they get instead, is true.
func thereToo(repoPath, where string) string {
	if where == "" {
		return ""
	}

	at, err := os.Stat(filepath.Join(repoPath, filepath.FromSlash(where)))
	if err != nil || !at.IsDir() {
		return ""
	}

	return where
}
