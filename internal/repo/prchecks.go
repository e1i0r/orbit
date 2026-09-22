package repo

// What GitHub's own checks say about a pull request, before it is merged.
//
// `gh pr merge` refuses a branch whose required checks have not passed, and
// what it prints when it does is a line of gh's own that Orbit passed
// straight through. A reader who pressed M got "merging the pull request of
// FRA-128 failed: exit status 1" and had to go and look. Read first, and
// the refusal can name which check is red and how many are still running.

import (
	"encoding/json"
	"fmt"
	"strings"
)

// CheckRun is one of GitHub's checks on a pull request.
//
// Bucket is gh's own word for where the check stands: pass, fail, pending,
// skipping or cancel. It is taken rather than derived because gh already
// collapses every forge's conclusions into those five, and a second
// mapping here would disagree with `gh pr checks` on somebody's screen.
type CheckRun struct {
	Name   string `json:"name"`
	Bucket string `json:"bucket"`
}

// The buckets that stop a merge, in gh's spelling.
const (
	checkFailed  = "fail"
	checkPending = "pending"
)

// Failed is whether this check is one nobody should merge over.
func (c CheckRun) Failed() bool { return c.Bucket == checkFailed }

// Pending is whether it has not finished.
func (c CheckRun) Pending() bool { return c.Bucket == checkPending }

// PRChecks is every check GitHub ran on this branch's pull request.
//
// A pull request with no checks at all answers nothing and no error: a
// repository without CI is an ordinary repository, and refusing to merge
// in one because nothing reported would be this reading inventing a rule
// GitHub does not have.
func (r Repo) PRChecks(wtDir, branch string) ([]CheckRun, error) {
	// gh exits non-zero when a check is red, which is an answer and not a
	// failure: the output is still the listing. Only an empty output with
	// an error is a reading that did not happen.
	out, err := gh(wtDir, "pr", "checks", branch, "--json", "name,bucket")
	if strings.TrimSpace(out) == "" {
		if err != nil {
			return nil, fmt.Errorf("read the checks on %q: %w", branch, err)
		}

		return nil, nil
	}

	var runs []CheckRun
	if err := json.Unmarshal([]byte(out), &runs); err != nil {
		return nil, fmt.Errorf("read the checks on %q: %w", branch, err)
	}

	return runs, nil
}

// Blocking is the checks that stand between this pull request and a merge:
// the red ones first, because a red check is a decision and a pending one
// is only a wait.
func Blocking(runs []CheckRun) (red, waiting []string) {
	for _, c := range runs {
		switch {
		case c.Failed():
			red = append(red, c.Name)
		case c.Pending():
			waiting = append(waiting, c.Name)
		}
	}

	return red, waiting
}
