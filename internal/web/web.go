// Package web serves the record to a browser.
//
// It is a second reader of internal/view, beside internal/ui/panes and not
// on top of them. The panes answer lines for a terminal — they are handed a
// width, a set of folds and a callback, and they give back styled rows;
// serving those to a browser would be painting ANSI in HTML and throwing
// away the one thing a browser is for. What both read is the same fold of
// the same record.
//
// The server answers JSON and nothing else. What the page does with it is
// the page's, and keeping the two apart is what lets the diagrams arrive
// later without this file changing.
package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/view"
)

// Reader is what the server needs of the board: the rows, and one task's
// record. It is declared here and not in internal/board because it is this
// package's need — two methods of the many that Reader has.
type Reader interface {
	Refresh() (board.Board, board.Changed, error)
	Log(repoPath, id string) ([]view.Entry, error)
}

// Worktrees answers where a task's checkout of a repository is, which is the
// one thing this package cannot work out for itself: it is a hash under the
// state root, and internal/store is what knows it.
type Worktrees interface {
	Worktree(repoPath, id string) (string, error)
}

// Server is the handler and what it reads through.
type Server struct {
	board Reader
	trees Worktrees
	root  string
}

// New is the server, over one board and one state root.
func New(r Reader, trees Worktrees, root string) *Server {
	return &Server{board: r, trees: trees, root: root}
}

// Handler is every route, mounted.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/board", s.serveBoard)
	mux.HandleFunc("GET /api/tasks/{id}", s.serveTask)
	mux.HandleFunc("GET /api/tasks/{id}/diff", s.serveDiff)
	mux.HandleFunc("GET /", s.servePage)

	return mux
}

// serveBoard is every task, in its band.
func (s *Server) serveBoard(w http.ResponseWriter, _ *http.Request) {
	b, _, err := s.board.Refresh()
	if err != nil {
		fail(w, http.StatusInternalServerError, "read the board", err)

		return
	}

	answer(w, boardOf(b, s.root))
}

// serveTask is one task and everything the record says about it.
func (s *Server) serveTask(w http.ResponseWriter, r *http.Request) {
	t, ok := s.find(w, r)
	if !ok {
		return
	}

	entries, err := s.board.Log(t.RepoPath, t.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "read the record of "+t.ID, err)

		return
	}

	answer(w, taskOf(t, entries))
}

// serveDiff is what the task changed, as git wrote it.
//
// The text is handed over as it came. Turning it into hunks and files is the
// page's job: a browser has a whole language for that and this has none.
func (s *Server) serveDiff(w http.ResponseWriter, r *http.Request) {
	t, ok := s.find(w, r)
	if !ok {
		return
	}

	if t.RepoPath == "" {
		answer(w, diffAnswer{ID: t.ID, Missing: true})

		return
	}

	dir, err := s.trees.Worktree(t.RepoPath, t.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "find the worktree of "+t.ID, err)

		return
	}

	// A task nobody has run has no checkout, and that is not a failure to
	// report: it is the ordinary state of every task in the To Do band.
	// Asked anyway, git answers with a chdir error about a path the reader
	// never chose and cannot act on.
	if _, err := os.Stat(dir); err != nil {
		answer(w, diffAnswer{ID: t.ID, Missing: true})

		return
	}

	one, err := repo.Open(t.RepoPath)
	if err != nil {
		fail(w, http.StatusInternalServerError, "open "+t.RepoPath, err)

		return
	}

	text, err := one.WorktreeDiff(dir)
	if err != nil {
		answer(w, diffAnswer{ID: t.ID, Failed: err.Error()})

		return
	}

	answer(w, diffAnswer{ID: t.ID, Text: text, Empty: strings.TrimSpace(text) == ""})
}

// find is the task a request is about, or a refusal written to the reader.
func (s *Server) find(w http.ResponseWriter, r *http.Request) (view.Task, bool) {
	id := r.PathValue("id")

	b, _, err := s.board.Refresh()
	if err != nil {
		fail(w, http.StatusInternalServerError, "read the board", err)

		return view.Task{}, false
	}

	for _, t := range b.Tasks {
		if t.ID == id {
			return t, true
		}
	}

	fail(w, http.StatusNotFound, fmt.Sprintf("no task %q on the board", id), nil)

	return view.Task{}, false
}

// answer writes one value as JSON.
func answer(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if err := json.NewEncoder(w).Encode(v); err != nil {
		// The status is already written by here, so there is nowhere to
		// report this but the log the caller keeps.
		_ = err //nolint:wsl // see above
	}
}

// fail says what went wrong in the shape every route says it.
func fail(w http.ResponseWriter, status int, said string, err error) {
	if err != nil {
		said += ": " + err.Error()
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	said = strings.TrimSpace(said)
	//nolint:errcheck // the status is written by here; there is nowhere left to report this
	_ = json.NewEncoder(w).Encode(map[string]string{"error": said})
}
