package web

// The verbs, and the guard in front of them.

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/view"
)

// hands is a verbs port that writes down what it was asked for.
type hands struct {
	asked   []string
	refuse  error
	pending []string
	held    bool
}

func (h *hands) Start(id, _ string) error  { return h.note("start", id) }
func (h *hands) Pause(id, _ string) error  { return h.note("pause", id) }
func (h *hands) Resume(id, _ string) error { return h.note("resume", id) }

func (h *hands) Continue(id, _ string) error { return h.note("continue", id) }
func (h *hands) Skip(id, _ string) error     { return h.note("skip", id) }
func (h *hands) Cancel(id, _ string) error   { return h.note("cancel", id) }

func (h *hands) Note(id, _, text string) error { return h.note("note "+text, id) }

func (h *hands) Direct(id, _, text string, restart bool) error {
	if restart {
		return h.note("direct+restart "+text, id)
	}

	return h.note("direct "+text, id)
}

func (h *hands) Requeue(id, _, why string) error { return h.note("requeue "+why, id) }

func (h *hands) Approve(id, _ string) ([]string, error) {
	if err := h.note("approve", id); err != nil {
		return nil, err
	}

	return h.pending, nil
}

func (h *hands) Standing(_, _ string) Standing { return Standing{Held: h.held, Pending: h.pending} }

func (h *hands) note(verb, id string) error {
	if h.refuse != nil {
		return h.refuse
	}

	h.asked = append(h.asked, verb+" "+id)

	return nil
}

// doing sends one verb the way the page sends it.
func doing(t *testing.T, s *Server, verb string, head map[string]string) (int, map[string]any) {
	t.Helper()

	return saying(t, s, verb, "{}", head)
}

// saying is the same, with a body the verb reads.
func saying(
	t *testing.T, s *Server, verb, body string, head map[string]string,
) (int, map[string]any) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/LED-1/"+verb, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	for name, value := range head {
		req.Header.Set(name, value)
	}

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	var out map[string]any

	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("%s answered something that is not JSON: %s", verb, rec.Body.String())
	}

	return rec.Code, out
}

// running is a server over one task and a set of hands.
func running(h *hands) *Server {
	return New(Ports{
		Board: aBoard{tasks: []view.Task{
			{ID: "LED-1", Title: "fix the total", Band: view.Running, Repo: "ledger"},
		}},
		Trees:     nowhere{path: "/nowhere"},
		Verbs:     h,
		Says:      h,
		Standings: h,
		Root:      "/code",
		Files:     built,
	})
}

// TestEveryVerbReachesTheTaskItNames.
func TestEveryVerbReachesTheTaskItNames(t *testing.T) {
	h := &hands{pending: []string{"github.com/x/y"}}
	s := running(h)

	for _, verb := range everyVerb {
		code, body := doing(t, s, verb, nil)
		if code != http.StatusOK {
			t.Fatalf("%s answered %d: %v", verb, code, body)
		}

		if body["did"] != verb {
			t.Errorf("%s answered that it did %v", verb, body["did"])
		}
	}

	for _, verb := range everyVerb {
		if !reached(h.asked, verb) {
			t.Errorf("%s never reached the task: %v", verb, h.asked)
		}
	}
}

// reached is whether the hands were asked for one verb, whatever it carried.
func reached(asked []string, verb string) bool {
	for _, one := range asked {
		if strings.HasPrefix(one, verb+" ") {
			return true
		}
	}

	return false
}

// everyVerb is the whole vocabulary, so that a verb added without a test is
// a verb this list catches.
var everyVerb = []string{
	"start", "pause", "resume", "continue", "skip", "cancel",
	"note", "direct", "requeue", "approve",
}

// TestWhatTheReaderTypedReachesTheTask. Three verbs carry words — a note, a
// correction, a reason — and a body that arrived and was dropped would be a
// note the next phase never reads, with nothing saying so.
func TestWhatTheReaderTypedReachesTheTask(t *testing.T) {
	h := &hands{}
	s := running(h)

	code, body := saying(t, s, "note", `{"text":"use cents, not floats"}`, nil)
	if code != http.StatusOK {
		t.Fatalf("a note answered %d: %v", code, body)
	}

	code, body = saying(t, s, "direct", `{"text":"start over","restart":true}`, nil)
	if code != http.StatusOK {
		t.Fatalf("a direct answered %d: %v", code, body)
	}

	code, body = saying(t, s, "requeue", `{"text":"the brief was wrong"}`, nil)
	if code != http.StatusOK {
		t.Fatalf("a requeue answered %d: %v", code, body)
	}

	want := []string{
		"note use cents, not floats LED-1",
		"direct+restart start over LED-1",
		"requeue the brief was wrong LED-1",
	}

	for _, one := range want {
		if !strings.Contains(strings.Join(h.asked, " | "), one) {
			t.Errorf("the hands were asked for %v, want %q in it", h.asked, one)
		}
	}
}

// TestAVerbThatRefusesSaysWhy. "Already being run", "over the unread cap" —
// these are answers about the reader's own machine, so they come back as
// something to read rather than as a server that broke.
func TestAVerbThatRefusesSaysWhy(t *testing.T) {
	s := running(&hands{refuse: errors.New("LED-1 is already being run by process 4")})

	code, body := doing(t, s, "start", nil)
	if code != http.StatusConflict {
		t.Fatalf("a refused start answered %d: %v", code, body)
	}

	said, ok := body["error"].(string)
	if !ok {
		t.Fatalf("a refused start answered no error at all: %v", body)
	}

	if !strings.Contains(said, "already being run") {
		t.Errorf("the refusal came back as %q", said)
	}
}

// TestAVerbIsRefusedFromAnywhereButThisPage. A page on any website the
// reader has open can post to this server; only the answer is hidden from
// it. Runs cost money, so a request that cannot be shown to have come from
// here is refused.
func TestAVerbIsRefusedFromAnywhereButThisPage(t *testing.T) {
	h := &hands{}
	s := running(h)

	for what, head := range map[string]map[string]string{
		"another site":   {"Sec-Fetch-Site": "cross-site"},
		"another origin": {"Origin": "http://evil.example"},
	} {
		code, body := doing(t, s, "cancel", head)
		if code != http.StatusForbidden {
			t.Errorf("a request from %s answered %d: %v", what, code, body)
		}
	}

	if len(h.asked) != 0 {
		t.Errorf("a refused request reached the task anyway: %v", h.asked)
	}
}

// TestAFormPostIsRefused. A cross-origin form may send three content types
// and none of them is JSON, so requiring it is what makes a browser ask
// permission first — and this server never answers that question.
func TestAFormPostIsRefused(t *testing.T) {
	h := &hands{}
	s := running(h)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/LED-1/start",
		strings.NewReader("id=LED-1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("a form post answered %d, want it refused", rec.Code)
	}

	if len(h.asked) != 0 {
		t.Errorf("a form post reached the task: %v", h.asked)
	}
}

// TestAReadingIsNeverAVerb. Every one of these changes what is happening on
// somebody's machine, so none of them may be reached by following a link.
func TestAReadingIsNeverAVerb(t *testing.T) {
	h := &hands{}
	s := running(h)

	for _, verb := range everyVerb {
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/tasks/LED-1/"+verb, nil))

		// The page route answers anything it does not know, so a GET here
		// lands on the window rather than on the verb. What matters is that
		// nothing happened.
		if len(h.asked) != 0 {
			t.Fatalf("GET %s did something: %v", verb, h.asked)
		}
	}
}

// TestABuildWithNoHandsShowsNoButtons, and refuses rather than pretending.
func TestABuildWithNoHandsShowsNoButtons(t *testing.T) {
	s := New(Ports{
		Board: aBoard{tasks: []view.Task{{ID: "LED-1", Band: view.ToDo, Repo: "ledger"}}},
		Trees: nowhere{path: "/nowhere"},
		Root:  "/code",
		Files: built,
	})

	code, body := doing(t, s, "start", nil)
	if code != http.StatusNotImplemented {
		t.Errorf("a build with no verbs answered %d: %v", code, body)
	}
}
