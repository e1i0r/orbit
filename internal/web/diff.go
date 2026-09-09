package web

// What a task changed, and the file it changed it in.
//
// The diff is handed over as git wrote it: turning a stream into files,
// hunks and lines is the page's job, and a browser has a whole language for
// that where this has none. What this does own is the two things a page
// cannot work out for itself — which checkout to read, and the file whole,
// for opening a hunk out past the three lines of context git leaves.

import (
	"net/http"
	"os"
	"strings"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/view"
)

// serveDiff is what the task changed, as git wrote it.
//
// The text is handed over as it came. Turning it into hunks and files is the
// page's job: a browser has a whole language for that and this has none.
func (s *Server) serveDiff(w http.ResponseWriter, r *http.Request) {
	c, ok := s.checkoutOf(w, r)
	if !ok {
		return
	}

	if c.missing {
		answer(w, diffAnswer{ID: c.task.ID, Missing: true})

		return
	}

	how := repo.DiffOptions{IgnoreWhitespace: r.URL.Query().Get("space") == "ignore"}

	text, err := c.repo.WorktreeDiff(c.dir, how)
	if err != nil {
		answer(w, diffAnswer{ID: c.task.ID, Failed: err.Error()})

		return
	}

	answer(w, diffAnswer{ID: c.task.ID, Text: text, Empty: strings.TrimSpace(text) == ""})
}

// mostOfAFile is the largest file this will hand over, in bytes.
const mostOfAFile = 4 << 20

// serveFile is one file of the worktree, whole.
//
// It is what opens a diff out: git writes three lines around each hunk, and
// the reader who wants to see what is above them is asking about the file
// and not about the change. Answered as text with the lines left as they
// are, because which lines the page wants is the page's arithmetic.
func (s *Server) serveFile(w http.ResponseWriter, r *http.Request) {
	c, ok := s.checkoutOf(w, r)
	if !ok {
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		fail(w, http.StatusBadRequest, "say which file with ?path=", nil)

		return
	}

	if c.missing {
		answer(w, fileAnswer{ID: c.task.ID, Path: path, Missing: true})

		return
	}

	// A path outside the worktree and a file that is not there answer the
	// same way: this is a reading of one checkout, and both are "there is no
	// such file here". Telling them apart would tell a caller whether a path
	// outside exists.
	text, err := c.repo.WorktreeFile(c.dir, path)
	if err != nil || len(text) > mostOfAFile {
		// Over the bound answers as missing too. What this is for is opening
		// a diff out by a screenful, and a page that asked a browser to hold
		// and lay out a ten-megabyte file is a page that stops answering.
		answer(w, fileAnswer{ID: c.task.ID, Path: path, Missing: true})

		return
	}

	answer(w, fileAnswer{ID: c.task.ID, Path: path, Text: text})
}

// checkout is a task and the checkout its work is in.
type checkout struct {
	task view.Task
	repo repo.Repo
	dir  string
	// missing says there is nothing to look in. A task nobody has run has
	// no checkout, and that is not a failure to report: it is the ordinary
	// state of every task in the To Do band. Asked anyway, git answers with
	// a chdir error about a path the reader never chose and cannot act on.
	missing bool
}

// checkoutOf is the task a request is about and where its work is, or a
// refusal already written to the reader.
func (s *Server) checkoutOf(w http.ResponseWriter, r *http.Request) (checkout, bool) {
	t, ok := s.find(w, r)
	if !ok {
		return checkout{}, false
	}

	if t.RepoPath == "" {
		return checkout{task: t, missing: true}, true
	}

	dir, err := s.trees.Worktree(t.RepoPath, t.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "find the worktree of "+t.ID, err)

		return checkout{}, false
	}

	if _, err := os.Stat(dir); err != nil {
		return checkout{task: t, missing: true}, true
	}

	one, err := repo.Open(t.RepoPath)
	if err != nil {
		fail(w, http.StatusInternalServerError, "open "+t.RepoPath, err)

		return checkout{}, false
	}

	return checkout{task: t, repo: one, dir: dir}, true
}

// and here
