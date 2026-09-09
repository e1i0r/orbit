package web

// The things a reader can do, rather than read about.
//
// One route, and one port behind it. What a verb is, what it takes and what
// it refuses is internal/verb's, declared once for the four ways in to
// Orbit; this package's whole part in it is turning a request into what the
// verb was asked with and the answer back into JSON. There is deliberately
// no list of verbs here — a list would be this package's own opinion about
// the vocabulary, and the four opinions drifting apart is what the
// declaration exists to stop.
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
	"fmt"
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
	// Saw is what a reading read, where the verb was one. A page with a
	// screen of its own for it reads that screen's route instead; this is
	// what the rest answer with.
	Saw any `json:"saw,omitempty"`
}

// mountVerbs is every verb, on the task it is about or on the board.
func (s *Server) mountVerbs(mux *http.ServeMux) {
	// A verb about one task takes it in the path, which is the shape the
	// page was already using when each verb had a route written by hand.
	mux.HandleFunc("POST /api/tasks/{id}/{verb}", s.serveVerb)

	// And one about the board itself takes none. Writing a task is here
	// rather than beside the others because there is no task yet.
	mux.HandleFunc("POST /api/do/{verb}", s.serveVerb)

	// A reading changes nothing, so it is a GET and needs no guard. The
	// screens with a shape of their own have routes of their own; this is
	// how the rest are read, and how any verb added later is read before
	// anybody draws it.
	mux.HandleFunc("GET /api/read/{verb}", s.serveRead)
	mux.HandleFunc("GET /api/read/{verb}/{id}", s.serveRead)
}

// serveVerb asks for one verb by the name in the path.
//
// A refusal from the verb is 409 and not 500. "This task is already being
// run", "the board is over its unread cap" — these are answers to the
// reader, about the state of their own machine, and a 500 would tell them
// Orbit is broken when it is working exactly as designed.
func (s *Server) serveVerb(w http.ResponseWriter, r *http.Request) {
	if !sameSite(w, r) {
		return
	}

	name := r.PathValue("verb")
	if s.asks == nil {
		fail(w, http.StatusNotImplemented, "this build cannot "+name, nil)

		return
	}

	// An empty body is the ordinary case: most verbs carry nothing, and a
	// page that sent none meant "do this to that task".
	var sent map[string]any

	_ = json.NewDecoder(r.Body).Decode(&sent) //nolint:errcheck // see above

	in := Asked{Args: words(sent)}

	if r.PathValue("id") != "" {
		t, ok := s.find(w, r)
		if !ok {
			return
		}

		in.Task, in.Repo = t.ID, t.RepoPath
	}

	out, err := s.asks.Ask(name, in)
	if err != nil {
		fail(w, http.StatusConflict, err.Error(), nil)

		return
	}

	answer(w, didAnswer{Did: name, Said: out.Said, Of: out.Of, Saw: out.Saw})
}

// serveRead asks for a reading.
func (s *Server) serveRead(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("verb")
	if s.asks == nil {
		fail(w, http.StatusNotImplemented, "this build cannot read "+name, nil)

		return
	}

	in := Asked{}

	if r.PathValue("id") != "" {
		t, ok := s.find(w, r)
		if !ok {
			return
		}

		in.Task, in.Repo = t.ID, t.RepoPath
	}

	// Whatever else was asked for arrives in the query, which is where a
	// reading's own arguments belong: a GET with a body is a request half
	// the world drops on the way.
	in.Args = map[string]string{}
	for k, v := range r.URL.Query() {
		in.Args[k] = v[0]
	}

	out, err := s.asks.Ask(name, in)
	if err != nil {
		fail(w, http.StatusConflict, err.Error(), nil)

		return
	}

	answer(w, didAnswer{Did: name, Said: out.Said, Saw: out.Saw})
}

// words is what was sent, as the strings a verb takes.
//
// A page sends JSON, so a switch arrives as true and a count as a number;
// what a verb takes is text it parses itself, one way, for all four ways in.
// Anything with a shape of its own is dropped rather than guessed at.
func words(sent map[string]any) map[string]string {
	out := make(map[string]string, len(sent))

	for k, v := range sent {
		switch t := v.(type) {
		case string:
			out[k] = t
		case bool, float64:
			out[k] = fmt.Sprint(t)
		}
	}

	return out
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
