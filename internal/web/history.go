package web

// Everything ever said about a task.
//
// It is answered as the markdown it already is rather than as turns the page
// would have to lay out again: the same text the window draws, the same file
// an engine is handed when a terminal is opened on the task, and the same
// thing that lands on disk when a reader exports it. One conversation, one
// rendering, three places to meet it.

import "net/http"

// historyAnswer is that text, and why there is none when there is none.
type historyAnswer struct {
	ID   string `json:"id"`
	Text string `json:"text,omitempty"`
	// Read says a store was there to ask. No conversation and no store are
	// different sentences, and the page says a different thing about each.
	Read   bool   `json:"read"`
	Failed string `json:"failed,omitempty"`
}

// serveHistory is the conversation about one task.
func (s *Server) serveHistory(w http.ResponseWriter, r *http.Request) {
	t, ok := s.find(w, r)
	if !ok {
		return
	}

	if s.told == nil {
		answer(w, historyAnswer{ID: t.ID})

		return
	}

	text, err := s.told.History(t.ID, t.RepoPath)
	if err != nil {
		answer(w, historyAnswer{ID: t.ID, Failed: err.Error()})

		return
	}

	answer(w, historyAnswer{ID: t.ID, Text: text, Read: true})
}
