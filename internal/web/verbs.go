package web

// The things a reader can do to a task, rather than read about it.
//
// Every route here is a POST, and every one of them goes through sameSite
// first. That guard is not ceremony. This server listens on loopback, and a
// page on any website the reader has open can send a form POST to
// 127.0.0.1:7777 — the browser sends it, and only the answer is hidden from
// the attacker. Started runs cost money and cancelled ones lose work, so a
// request that cannot be shown to have come from this page is refused.
//
// Two things make that check hold. A cross-origin request may only carry a
// handful of content types without asking permission first, and none of them
// is JSON: requiring it means a browser has to send a preflight, which this
// server does not answer. And where the browser tells us where the request
// came from — Origin, or Sec-Fetch-Site — that is checked against this host.
// Neither is enough alone; together they close both doors.

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// didAnswer is what a verb says when it worked: what was done, in the words
// the reader is owed. It is deliberately not the new state of the task — the
// verb returns when the word is on disk, and what the run did about it is
// the record's to say, one poll later.
type didAnswer struct {
	Did  string   `json:"did"`
	Said string   `json:"said"`
	Of   []string `json:"of,omitempty"`
}

// mountVerbs is every verb, each on the task it is asked about.
func (s *Server) mountVerbs(mux *http.ServeMux) {
	for _, one := range []struct {
		verb string
		said string
		act  func(id, repo string) error
	}{
		{"start", "started", s.startOne},
		{"pause", "asked to pause at its next phase", s.pauseOne},
		{"resume", "asked to carry on", s.resumeOne},
		{"continue", "let past the gate it was waiting at", s.continueOne},
		{"skip", "let past the phase it was in", s.skipOne},
		{"cancel", "asked the run to stop", s.cancelOne},
	} {
		mux.HandleFunc("POST /api/tasks/{id}/"+one.verb,
			s.doing(one.verb, plain(one.verb, one.said, one.act)))
	}

	mux.HandleFunc("POST /api/tasks/{id}/note", s.doing("note", s.noting))
	mux.HandleFunc("POST /api/tasks/{id}/direct", s.doing("direct", s.directing))
	mux.HandleFunc("POST /api/tasks/{id}/requeue", s.doing("requeue", s.requeueing))
	mux.HandleFunc("POST /api/tasks/{id}/approve", s.doing("approve", s.approving))
}

// The six that answer nothing but "it is on disk". They are methods rather
// than closures over s.verbs because the port is nil until a build fills it,
// and a closure taken at mount time would capture the nil.
func (s *Server) startOne(id, repo string) error  { return s.verbs.Start(id, repo) }
func (s *Server) pauseOne(id, repo string) error  { return s.verbs.Pause(id, repo) }
func (s *Server) resumeOne(id, repo string) error { return s.verbs.Resume(id, repo) }
func (s *Server) skipOne(id, repo string) error   { return s.verbs.Skip(id, repo) }
func (s *Server) cancelOne(id, repo string) error { return s.verbs.Cancel(id, repo) }

func (s *Server) continueOne(id, repo string) error { return s.verbs.Continue(id, repo) }

// asked is what a verb was sent. Three of the ten carry the reader's own
// words, and one of those carries whether it also starts the next run.
type asked struct {
	Text    string `json:"text,omitempty"`
	Restart bool   `json:"restart,omitempty"`
}

// done is what a verb answers: what it did, or why it would not.
type done func(id, repo string, said asked) (didAnswer, error)

// plain is a verb whose whole answer is that it was asked for, and which
// carries nothing the reader typed.
func plain(verb, said string, act func(id, repo string) error) done {
	return func(id, repo string, _ asked) (didAnswer, error) {
		if err := act(id, repo); err != nil {
			return didAnswer{}, err
		}

		return didAnswer{Did: verb, Said: id + " " + said}, nil
	}
}

// noting leaves a word for the phase that starts next.
func (s *Server) noting(id, repo string, said asked) (didAnswer, error) {
	if err := s.says.Note(id, repo, said.Text); err != nil {
		return didAnswer{}, err
	}

	return didAnswer{Did: "note", Said: "noted on " + id}, nil
}

// directing puts a correction on the record and stops the run so that the
// next one starts having read it.
func (s *Server) directing(id, repo string, said asked) (didAnswer, error) {
	if err := s.says.Direct(id, repo, said.Text, said.Restart); err != nil {
		return didAnswer{}, err
	}

	if said.Restart {
		return didAnswer{Did: "direct", Said: id + " redirected and started again"}, nil
	}

	return didAnswer{Did: "direct", Said: id + " redirected — the next run reads it"}, nil
}

// requeueing takes a task back to the queue.
func (s *Server) requeueing(id, repo string, said asked) (didAnswer, error) {
	if err := s.says.Requeue(id, repo, said.Text); err != nil {
		return didAnswer{}, err
	}

	return didAnswer{Did: "requeue", Said: id + " is back in the queue"}, nil
}

// approving says yes to what the dependency gate stopped a run for.
func (s *Server) approving(id, repo string, _ asked) (didAnswer, error) {
	names, err := s.says.Approve(id, repo)
	if err != nil {
		return didAnswer{}, err
	}

	return didAnswer{
		Did:  "approve",
		Said: "approved for " + id + ": " + strings.Join(names, ", "),
		Of:   names,
	}, nil
}

// doing is the one path every verb takes: the guard, the task, the act, and
// what it answered.
//
// A refusal from the verb is 409 and not 500. "This task is already being
// run", "the board is over its unread cap" — these are answers to the
// reader, about the state of their own machine, and a 500 would tell them
// Orbit is broken when it is working exactly as designed.
func (s *Server) doing(what string, act done) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sameSite(w, r) {
			return
		}

		if s.verbs == nil || s.says == nil {
			fail(w, http.StatusNotImplemented, "this build cannot "+what+" a task", nil)

			return
		}

		t, ok := s.find(w, r)
		if !ok {
			return
		}

		var said asked

		// An empty body is the ordinary case: six of the ten carry nothing,
		// and a page that sent none meant "do this to that task".
		_ = json.NewDecoder(r.Body).Decode(&said) //nolint:errcheck // see above

		did, err := act(t.ID, t.RepoPath, said)
		if err != nil {
			fail(w, http.StatusConflict, err.Error(), nil)

			return
		}

		answer(w, did)
	}
}

// sameSite refuses a request that cannot be shown to have come from this
// page, and says so to the reader rather than silently doing nothing.
func sameSite(w http.ResponseWriter, r *http.Request) bool {
	// A cross-origin form can send text/plain, a urlencoded body or a
	// multipart one, and nothing else without asking first. Requiring JSON
	// is what makes a browser ask, and this server never says yes.
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		fail(w, http.StatusUnsupportedMediaType, "send this as application/json", nil)

		return false
	}

	// Chrome and Firefox say where a request came from. When they do, it
	// settles the question on its own.
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
		fail(w, http.StatusForbidden, "this can only be asked for from orbit's own page", nil)

		return false
	}

	// And where an Origin is sent, its host has to be the one this request
	// arrived at. A missing Origin is not suspicious — a same-origin GET-
	// shaped POST from our own fetch may not carry one — and the content
	// type above has already ruled out the cross-origin form.
	if origin := r.Header.Get("Origin"); origin != "" {
		at, err := url.Parse(origin)
		if err != nil || at.Host != r.Host {
			fail(w, http.StatusForbidden, "this can only be asked for from orbit's own page", nil)

			return false
		}
	}

	return true
}
