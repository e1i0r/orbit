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
	"io/fs"
	"net/http"
	"strings"

	"github.com/e1i0r/orbit/internal/board"
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

// Flows is where the flow files a reader wrote live. It is flow.Source under
// another name, declared here because it is this package's need, and it is
// what lets the flow route answer the file on disk rather than only the one
// built into the binary.
type Flows interface {
	FlowDir() string
}

// Ports is everything the server reads through, and where it reads it.
//
// A struct rather than five arguments, because three of them are interfaces
// and one value satisfies two: Board and Trees are the same reader today, so
// a call that swapped them would still compile and would still be wrong.
type Ports struct {
	Board Reader
	Trees Worktrees
	Flows Flows
	// Knows, Talks and Roster are the three screens that read something
	// other than the record. Any of them may be nil, and the screen then
	// says it has nothing to read rather than pretending it read nothing:
	// see ports.go.
	Knows  Knows
	Talks  Talks
	Roster Roster
	// Verbs is what a reader can do rather than read, and Standings which
	// of it is worth offering. Nil is a window that shows no buttons — see
	// ports.go.
	Verbs     Verbs
	Says      Says
	Told      Told
	Standings Standings
	Root      string
	// Files is the built window. It is passed in rather than embedded here
	// so that this package can be tested without one, and so that the only
	// thing that knows where the build output lives is the package it lives
	// in.
	Files fs.FS
}

// Server is the handler and what it reads through.
type Server struct {
	board  Reader
	trees  Worktrees
	flows  Flows
	knows  Knows
	talks  Talks
	roster Roster
	verbs  Verbs
	says   Says
	told   Told
	stands Standings
	root   string
	files  fs.FS
}

// New is the server, over one board and one state root.
func New(p Ports) *Server {
	return &Server{
		board: p.Board, trees: p.Trees, flows: p.Flows, knows: p.Knows,
		talks: p.Talks, roster: p.Roster, verbs: p.Verbs, says: p.Says,
		told: p.Told, stands: p.Standings,
		root: p.Root, files: p.Files,
	}
}

// Handler is every route, mounted.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/board", s.serveBoard)
	mux.HandleFunc("GET /api/tasks/{id}", s.serveTask)
	mux.HandleFunc("GET /api/tasks/{id}/diff", s.serveDiff)
	mux.HandleFunc("GET /api/tasks/{id}/file", s.serveFile)
	mux.HandleFunc("GET /api/tasks/{id}/flow", s.serveFlow)
	mux.HandleFunc("GET /api/tasks/{id}/impact", s.serveImpact)
	mux.HandleFunc("GET /api/tasks/{id}/history", s.serveHistory)
	mux.HandleFunc("GET /api/flows", s.serveFlows)
	mux.HandleFunc("GET /api/knowledge", s.serveKnowledge)
	mux.HandleFunc("GET /api/supervisor", s.serveSupervisor)
	mux.HandleFunc("GET /api/engines", s.serveEngines)
	mux.HandleFunc("GET /api/repos", s.serveRepos)
	s.mountVerbs(mux)

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

	answer(w, taskOf(t, entries, s.standing(t)))
}

// standing is what this task can be asked for, and nothing at all in a
// build with no verbs to ask it: a window that shows no buttons.
func (s *Server) standing(t view.Task) Standing {
	if s.stands == nil {
		return Standing{}
	}

	return s.stands.Standing(t.ID, t.RepoPath)
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
//
// Never cached. Every route here is a reading of a record that is being
// written to while the page is open, and a browser holding yesterday's
// answer shows a run that has since finished as though it were still going.
// It is the same rule the page itself is served under: the hashed assets are
// immutable and everything that says what is true right now is not.
func answer(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

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
