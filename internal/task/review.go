package task

// What a reviewer said, brought into the record so a phase can answer it.

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/repo"
	"github.com/e1i0r/orbit/internal/store"
)

// Review writes down what reviewers said on a task's pull requests, and
// answers how many comments it found.
//
// The comments become events rather than a message pasted into a prompt.
// What a reviewer asked for is a fact about the task — it outlives the run
// that answers it, and a reader six months later wants to know a human asked
// for the change as much as they want the change — so it goes where every
// other fact about a task goes.
//
// They are consumed the way a note is: the phase that runs next is told, and
// the phase after it is not told again. A comment repeated into every phase
// of a flow is a comment three phases answer separately.
func Review(s *store.Store, t Task, r repo.Repo, comments []repo.Comment) (int, error) {
	for _, c := range comments {
		data := map[string]string{"by": c.Author, "where": c.Where(), "repo": r.Name}
		if c.URL != "" {
			data["url"] = c.URL
		}

		text, _ := captured(c.Body)
		if err := emit(s, t, record.Event{Kind: record.ReviewComment, Text: text, Data: data}); err != nil {
			return 0, err
		}
	}

	return len(comments), nil
}

// unansweredReviews is what reviewers have said that no phase has been told
// about yet, in the words they wrote.
//
// The same rule notes keep, and for the same reason: a phase.started is what
// clears them, because a phase that ran was handed them and answering the
// same comment twice is how two fixes for one remark end up in a diff.
func unansweredReviews(s *store.Store, t Task) []string {
	events, err := Events(s, t)
	if err != nil {
		return nil
	}

	var said []string

	for _, e := range events {
		switch e.Kind {
		case record.PhaseStarted:
			said = nil
		case record.ReviewComment:
			if strings.TrimSpace(e.Text) != "" {
				said = append(said, reviewLine(e))
			}
		}
	}

	return said
}

// reviewLine is one comment as a phase reads it: who, where, and what.
//
// Who wrote it is kept because it is the difference between a rule and an
// opinion — a phase answering a maintainer and a phase answering a passing
// remark should not weigh them the same, and only the name says which is
// which.
func reviewLine(e record.Event) string {
	who := e.Data["by"]
	if who == "" {
		who = "a reviewer"
	}

	where := e.Data["where"]
	if where == "" {
		where = "the pull request"
	}

	return fmt.Sprintf("%s, on %s: %s", who, where, strings.TrimSpace(e.Text))
}
