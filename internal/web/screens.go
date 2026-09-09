package web

// The screens beside the board: the flows Orbit can walk, what it has been
// told, what has been said to the supervisor, the engines this machine can
// run, and the repositories under the root.
//
// Five readings and one file, because they are one idea: each answers "what
// is set up here", none of them is about a task, and each is a dozen lines
// over a port. Split apart they would be five files of one function.
//
// A port nobody filled answers an empty reading rather than an error. A
// window built without a store is a window that cannot know, and saying so
// is the honest answer — a 500 would say Orbit is broken.

import (
	"net/http"
	"sort"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/flow"
	"github.com/e1i0r/orbit/internal/view"
)

// flowsAnswer is every flow a task can be started under.
type flowsAnswer struct {
	Flows []flowShape `json:"flows"`
}

// flowShape is one flow as it is written, with no run behind it: the phases
// carry no standing, because a definition has not happened to anything.
type flowShape struct {
	Name        string        `json:"name"`
	Origin      string        `json:"origin"`
	Description string        `json:"description,omitempty"`
	Attempts    int           `json:"attempts"`
	DiffBudget  int           `json:"diffBudget,omitempty"`
	Phases      []phaseAnswer `json:"phases,omitempty"`
	Failed      string        `json:"failed,omitempty"`
}

// serveFlows is every flow, built-in and written.
func (s *Server) serveFlows(w http.ResponseWriter, _ *http.Request) {
	out := flowsAnswer{Flows: []flowShape{}}

	for _, listed := range flow.List(s.flows) {
		one := flowShape{Name: listed.Name, Origin: originName(listed.Origin)}

		f, err := flow.Resolve(s.flows, listed.Name)
		if err != nil {
			// Listed and unreadable is a real state — a file with a typo in
			// it — and the reader who has to fix it is the one looking at
			// this screen.
			one.Failed = err.Error()
			out.Flows = append(out.Flows, one)

			continue
		}

		one.Description, one.Attempts, one.DiffBudget = f.Description, f.AttemptCap(), f.DiffBudget
		for i := range f.Phases {
			one.Phases = append(one.Phases, phaseShape(f.Phases[i]))
		}

		out.Flows = append(out.Flows, one)
	}

	answer(w, out)
}

// originName is where a flow came from, in a word.
func originName(o flow.Origin) string {
	switch o {
	case flow.OriginBuiltin:
		return "builtin"
	case flow.OriginUser:
		return "yours"
	case flow.OriginShadow:
		return "shadow"
	case flow.OriginUnknown:
		return "unknown"
	}

	return "unknown"
}

// knowledgeAnswer is everything Orbit has been told.
type knowledgeAnswer struct {
	Facts []Fact `json:"facts"`
	// Read says a store was there to ask. No facts and no store are
	// different sentences, and the screen says a different thing about each.
	Read bool `json:"read"`
}

// serveKnowledge is what Orbit knows.
func (s *Server) serveKnowledge(w http.ResponseWriter, _ *http.Request) {
	if s.knows == nil {
		answer(w, knowledgeAnswer{Facts: []Fact{}})

		return
	}

	facts := s.knows.Facts()
	if facts == nil {
		facts = []Fact{}
	}

	answer(w, knowledgeAnswer{Facts: facts, Read: true})
}

// supervisorAnswer is the thread: the conversations, and every turn.
type supervisorAnswer struct {
	Chats  []Chat `json:"chats"`
	Said   []Said `json:"said"`
	Read   bool   `json:"read"`
	Failed string `json:"failed,omitempty"`
}

// serveSupervisor is what has been said to the supervisor and what it
// answered.
func (s *Server) serveSupervisor(w http.ResponseWriter, _ *http.Request) {
	out := supervisorAnswer{Chats: []Chat{}, Said: []Said{}}
	if s.talks == nil {
		answer(w, out)

		return
	}

	chats, said, err := s.talks.Thread()
	if err != nil {
		out.Failed = err.Error()
		answer(w, out)

		return
	}

	out.Read = true

	if chats != nil {
		out.Chats = chats
	}

	if said != nil {
		out.Said = said
	}

	answer(w, out)
}

// enginesAnswer is the catalogue and which of it this machine can run.
type enginesAnswer struct {
	Engines []EngineInfo `json:"engines"`
	// Settled is the one a run uses when nothing names another.
	Settled string `json:"settled,omitempty"`
	Read    bool   `json:"read"`
}

// serveEngines is every engine this build knows.
func (s *Server) serveEngines(w http.ResponseWriter, _ *http.Request) {
	if s.roster == nil {
		answer(w, enginesAnswer{Engines: []EngineInfo{}})

		return
	}

	list := s.roster.Engines()
	if list == nil {
		list = []EngineInfo{}
	}

	answer(w, enginesAnswer{Engines: list, Settled: s.roster.Settled(), Read: true})
}

// reposAnswer is every repository under the root, with the work in it.
type reposAnswer struct {
	Root  string       `json:"root"`
	Repos []repoDetail `json:"repos"`
}

// repoDetail is one repository: where it is, and how its tasks stand.
type repoDetail struct {
	Name string `json:"name"`
	Path string `json:"path"`
	// Tasks is how many the board holds for it, and Bands the same number
	// broken up the way the board breaks it up.
	Tasks int            `json:"tasks"`
	Bands map[string]int `json:"bands"`
}

// serveRepos is the repositories the board was read over.
//
// Off the board and not off the disk: what a reader wants here is which
// repositories Orbit is watching and what is happening in each, and the
// board has already walked them. A second walk would be a second answer.
func (s *Server) serveRepos(w http.ResponseWriter, _ *http.Request) {
	b, _, err := s.board.Refresh()
	if err != nil {
		fail(w, http.StatusInternalServerError, "read the board", err)

		return
	}

	answer(w, reposAnswer{Root: s.root, Repos: reposOf(b)})
}

// reposOf counts the board's tasks into the repositories they belong to.
func reposOf(b board.Board) []repoDetail {
	in := map[string]*repoDetail{}
	out := make([]repoDetail, 0, len(b.RepoList))

	for _, r := range b.RepoList {
		one := repoDetail{Name: r.Name, Path: r.Path, Bands: map[string]int{}}
		out = append(out, one)
	}

	for i := range out {
		in[out[i].Name] = &out[i]
	}

	for _, t := range b.Tasks {
		for _, name := range namesOf(t) {
			one, held := in[name]
			if !held {
				continue
			}

			one.Tasks++
			one.Bands[bandName(view.BandOf(t))]++
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })

	return out
}

// namesOf is every repository one task has been worked in, and the one it
// names when the record kept no list.
func namesOf(t view.Task) []string {
	if len(t.Repos) > 0 {
		return t.Repos
	}

	if t.Repo != "" {
		return []string{t.Repo}
	}

	return nil
}
