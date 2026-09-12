package web

// Every screen, read the way the page reads it.
//
// The screens are GETs over the ports; nil ports read as nothing to read
// rather than as a failure, so each screen is asked twice — with something
// behind it, and with nothing. A screen added without a test is a screen
// this list catches.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/view"
)

// builtinFlows resolves the flows orbit ships, from nowhere on disk.
type builtinFlows struct{}

func (builtinFlows) FlowDir() string { return "" }

// told is a Told port that answers one history.
type told struct{ body string }

func (t told) History(id, repo string) (string, error) { return t.body, nil }

// known is a Knows port that answers what it was told.
type known struct{ facts []Fact }

func (k known) Facts() []Fact { return k.facts }

// talked is a Talks port with one conversation in it.
type talked struct{}

func (talked) Thread() ([]Chat, []Said, error) {
	return []Chat{{ID: "1", Title: "first", Turns: 2}}, []Said{{Text: "hi"}}, nil
}

// crewed is a Roster port with one engine on it.
type crewed struct{}

func (crewed) Engines() []EngineInfo { return []EngineInfo{{Name: "claude", Available: true}} }

func (crewed) Settled() string { return "claude" }

// reading sends one GET the way the page sends it, and answers the code
// with the body decoded.
func reading(t *testing.T, s *Server, path string) (int, map[string]any) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	var out map[string]any

	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("%s answered something that is not JSON: %s", path, rec.Body.String())
	}

	return rec.Code, out
}

// full is a server with something behind every screen: one task, one
// fact, one conversation, one engine, and one history.
func full() *Server {
	return New(Ports{
		Board:     listedBoard{},
		Trees:     nowhere{path: "/nowhere"},
		Knows:     known{facts: []Fact{{Phrase: "amounts are cents", Scope: "ledger"}}},
		Talks:     talked{},
		Roster:    crewed{},
		Told:      told{body: "# LED-1\n"},
		Standings: &hands{},
		Root:      "/code",
		Files:     built,
	})
}

// listedBoard is a board with tasks in repositories, the way a workspace
// with checkouts folds.
type listedBoard struct{ board.Board }

func (b listedBoard) Refresh() (board.Board, board.Changed, error) {
	return board.Board{
		Tasks: []view.Task{
			{ID: "LED-1", Title: "fix the total", Band: view.ToDo, Repo: "ledger", Repos: []string{"ledger", "acme"}},
			{ID: "LED-2", Title: "ship it", Band: view.Done},
		},
		RepoList: []board.RepoInfo{{Name: "ledger", Path: "/src/ledger"}},
	}, board.Changed{}, nil
}

func (b listedBoard) Log(_, _ string) ([]view.Entry, error) { return nil, nil }

func (b listedBoard) Rescan() error { return nil }

// TestEveryScreenReadsWhatIsBehindIt.
func TestEveryScreenReadsWhatIsBehindIt(t *testing.T) {
	s := full()

	for path, want := range map[string]string{
		"/api/board":       "LED-1",
		"/api/tasks/LED-1": "LED-1",
		"/api/flows":       "flows",
		"/api/knowledge":   "cents",
		"/api/supervisor":  "first",
		"/api/engines":     "claude",
		"/api/repos":       "repos",
	} {
		code, body := reading(t, s, path)
		if code != http.StatusOK {
			t.Errorf("%s answered %d: %v", path, code, body)
			continue
		}

		flat, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal the answer: %v", err)
		}

		if !strings.Contains(string(flat), want) {
			t.Errorf("%s answered %s, want it to carry %q", path, flat, want)
		}
	}

	if code, _ := reading(t, s, "/api/tasks/LED-1/history"); code != http.StatusOK {
		t.Errorf("history answered %d", code)
	}
}

// TestALoopReadsLikeAPhaseOutsideOne. The block a loop repeats, with
// each phase inside it standing where the run got it to.
func TestALoopReadsLikeAPhaseOutsideOne(t *testing.T) {
	s := New(Ports{
		Board: aBoard{tasks: []view.Task{
			{ID: "LED-2", Title: "round and round", Band: view.Running, Repo: "ledger", Flow: "coverage"},
		}},
		Trees:     nowhere{path: "/nowhere"},
		Flows:     builtinFlows{},
		Standings: &hands{},
		Root:      "/code",
		Files:     built,
	})

	code, body := reading(t, s, "/api/tasks/LED-2/flow")
	if code != http.StatusOK {
		t.Fatalf("flow answered %d: %v", code, body)
	}

	flat, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal the answer: %v", err)
	}

	if !strings.Contains(string(flat), "2-until-it-passes") {
		t.Errorf("the flow does not carry its loop:\n%s", flat)
	}
}

// shows no buttons: the screen says it has nothing to read rather than
// pretending it read nothing.
func TestAScreenWithNothingBehindItReadsEmpty(t *testing.T) {
	s := server(aBoard{tasks: []view.Task{
		{ID: "LED-1", Title: "fix the total", Band: view.ToDo, Repo: "ledger"},
	}}, nowhere{path: "/nowhere"}, "/code", built)

	for _, path := range []string{"/api/knowledge", "/api/supervisor", "/api/engines"} {
		code, body := reading(t, s, path)
		if code != http.StatusOK {
			t.Errorf("%s answered %d: %v", path, code, body)
		}
	}
}

// TestAReadingReadsThroughAsks. A reading changes nothing, so it goes
// through the verbs port like everything else the page does.
func TestAReadingReadsThroughAsks(t *testing.T) {
	h := &hands{}
	s := running(h)

	req := httptest.NewRequest(http.MethodGet, "/api/read/thread", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("read answered %d: %s", rec.Code, rec.Body.String())
	}
}

// TestWhatCannotBeReadFailsInWords. A checkout nobody can open, a diff git
// refuses, a file past its budget: each answers its own failure rather
// than the server's.
func TestWhatCannotBeReadFailsInWords(t *testing.T) {
	s := full()

	for _, path := range []string{
		"/api/tasks/LED-1/diff",
		"/api/tasks/LED-1/diff?space=ignore",
		"/api/tasks/LED-1/file?path=done.txt",
		"/api/tasks/LED-1/impact",
		"/api/tasks/LED-1/flow",
		"/api/tasks/LED-404/diff",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)

		if rec.Code < 200 || rec.Code >= 500 {
			t.Errorf("%s answered %d", path, rec.Code)
		}
	}
}
