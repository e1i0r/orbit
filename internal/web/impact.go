package web

// What a change reaches beyond the files it touched.
//
// Three claims of three different weights, and the page is told which is
// which by keeping them apart in the answer: the history's word on what
// usually comes along, the sentences the tests among those files are named
// after, and the engine's own account of what its change asks and promises.
// Nothing here was run.
//
// The cockpit's pane opens with a fourth section — the flow's checks run on
// both sides of the change — and that one is missing here on purpose. It is
// a verb and not a reading: it checks out the base, runs somebody's test
// suite twice, and costs minutes. A GET that does that is a GET that a page
// refresh can set going, so it waits for the verbs.

import (
	"net/http"

	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/view"
)

// impactAnswer is that reading for one task.
type impactAnswer struct {
	ID      string `json:"id"`
	Missing bool   `json:"missing,omitempty"`
	Failed  string `json:"failed,omitempty"`
	// Changed is what the work touched and Commits how many commits the
	// history was read from. A repository with nine commits in it says so,
	// rather than letting three coincidences look like a pattern.
	Changed   []string         `json:"changed,omitempty"`
	Commits   int              `json:"commits"`
	Coupled   []coupledAnswer  `json:"coupled,omitempty"`
	Contracts []contractAnswer `json:"contracts,omitempty"`
	Delta     *deltaAnswer     `json:"delta,omitempty"`
}

// coupledAnswer is one file the history says follows another, and did not
// this time.
type coupledAnswer struct {
	File  string  `json:"file"`
	With  string  `json:"with"`
	Times int     `json:"times"`
	Of    int     `json:"of"`
	Ratio float64 `json:"ratio"`
}

// contractAnswer is one thing a test says it holds, by its own name.
type contractAnswer struct {
	File string `json:"file"`
	Says string `json:"says"`
}

// serveImpact is what this task's change reaches.
func (s *Server) serveImpact(w http.ResponseWriter, r *http.Request) {
	c, ok := s.checkoutOf(w, r)
	if !ok {
		return
	}

	if c.missing {
		answer(w, impactAnswer{ID: c.task.ID, Missing: true})

		return
	}

	entries, err := s.board.Log(c.task.RepoPath, c.task.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "read the record of "+c.task.ID, err)

		return
	}

	// The history is read and not written to, so a repository that will not
	// answer is a reading that failed rather than a server that broke.
	got, err := c.repo.Impact(c.dir)
	if err != nil {
		answer(w, impactAnswer{ID: c.task.ID, Failed: err.Error()})

		return
	}

	answer(w, impactOf(c.task.ID, got, entries))
}

// impactOf is the reading and the engine's claim, side by side.
func impactOf(id string, got repo.Impact, entries []view.Entry) impactAnswer {
	out := impactAnswer{
		ID: id, Changed: got.Changed, Commits: got.Commits, Delta: lastDelta(entries),
	}

	for _, c := range got.Coupled {
		out.Coupled = append(out.Coupled, coupledAnswer{
			File: c.File, With: c.With, Times: c.Times, Of: c.Of, Ratio: c.Ratio(),
		})
	}

	for _, c := range got.Contracts {
		out.Contracts = append(out.Contracts, contractAnswer{File: c.File, Says: c.Says})
	}

	return out
}

// lastDelta is the one the attempt that stands wrote, and nothing when no
// attempt wrote one.
//
// The last, for the reason the report shows the last: a task run three times
// said this three times, and the two before it are about work that was
// thrown away.
func lastDelta(entries []view.Entry) *deltaAnswer {
	for i := len(entries) - 1; i >= 0; i-- {
		if d := deltaOf(entries[i]); d != nil {
			return d
		}
	}

	return nil
}
