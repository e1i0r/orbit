package repo

// What a reviewer said on the pull request, read back so a phase can answer
// it.

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Comment is one thing a reviewer wrote: who wrote it, where, and what they
// said.
//
// Path and Line are empty on a comment about the pull request as a whole,
// which is most of the ones that matter — "this needs a test" is rarely
// attached to a line.
type Comment struct {
	Author string
	Path   string
	Line   int
	Body   string
	URL    string
}

// Where is the place a comment is about, as a person would say it.
func (c Comment) Where() string {
	if c.Path == "" {
		return "the pull request"
	}

	if c.Line == 0 {
		return c.Path
	}

	return fmt.Sprintf("%s:%d", c.Path, c.Line)
}

// ReviewComments is everything said on the pull request of a branch: the
// review bodies, the comments on the conversation, and the ones left on
// lines — oldest first.
//
// The line comments come from the REST endpoint and not from `gh pr view`.
// That view has no path and no line to give: its `comments` are the
// conversation, and its `reviews` are the summaries above them, so Path and
// Line were zero on every comment Orbit read and a phase answering a review
// was told that every remark was about the pull request as a whole.
//
// Unresolved ones only would be the better question and gh cannot answer it
// without the GraphQL API and a token scope Orbit does not ask for. Reading
// them all is the honest version of what can be read — a comment that was
// already dealt with reads as one more thing to check rather than as one
// nobody noticed, and that is the safe direction to be wrong in.
func (r Repo) ReviewComments(wtDir, branch string) ([]Comment, error) {
	out, err := gh(wtDir, "pr", "view", branch, "--json", "number,reviews,comments")
	if err != nil {
		return nil, fmt.Errorf("read the reviews of %q: %w", branch, err)
	}

	var body struct {
		Number  int `json:"number"`
		Reviews []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			Body      string    `json:"body"`
			URL       string    `json:"url"`
			CreatedAt time.Time `json:"submittedAt"`
		} `json:"reviews"`
		Comments []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			Body      string    `json:"body"`
			URL       string    `json:"url"`
			CreatedAt time.Time `json:"createdAt"`
		} `json:"comments"`
	}

	if err := json.Unmarshal([]byte(out), &body); err != nil {
		return nil, fmt.Errorf("read what gh answered about %q: %w", branch, err)
	}

	var said []timed

	for _, rv := range body.Reviews {
		if strings.TrimSpace(rv.Body) == "" {
			// An approval with no words is not a comment to answer. It is
			// the most common review there is, and listing it would put "\
			// nothing to do" in front of the things there are to do.
			continue
		}

		said = append(said, timed{
			at:      rv.CreatedAt,
			comment: Comment{Author: rv.Author.Login, Body: strings.TrimSpace(rv.Body), URL: rv.URL},
		})
	}

	for _, c := range body.Comments {
		if strings.TrimSpace(c.Body) == "" {
			continue
		}

		said = append(said, timed{
			at:      c.CreatedAt,
			comment: Comment{Author: c.Author.Login, Body: strings.TrimSpace(c.Body), URL: c.URL},
		})
	}

	onLines, err := r.lineComments(wtDir, body.Number)
	if err != nil {
		return nil, err
	}

	said = append(said, onLines...)

	// Oldest first, which is what the doc above promises and what a phase
	// answering them wants: appended by kind, the summaries came before the
	// remarks they were written about.
	sort.SliceStable(said, func(i, j int) bool { return said[i].at.Before(said[j].at) })

	found := make([]Comment, 0, len(said))
	for _, one := range said {
		found = append(found, one.comment)
	}

	return found, nil
}

// timed is a comment beside when it was written, for the one sort.
type timed struct {
	at      time.Time
	comment Comment
}

// lineComments is what reviewers wrote against particular lines.
//
// A pull request with none answers an empty array, which is the ordinary
// case and not a failure.
func (r Repo) lineComments(wtDir string, number int) ([]timed, error) {
	if number == 0 {
		return nil, nil
	}

	out, err := gh(wtDir, "api", "--paginate",
		fmt.Sprintf("repos/{owner}/{repo}/pulls/%d/comments", number))
	if err != nil {
		return nil, fmt.Errorf("read the line comments of pull request %d: %w", number, err)
	}

	var body []struct {
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		Body      string    `json:"body"`
		Path      string    `json:"path"`
		Line      int       `json:"line"`
		URL       string    `json:"html_url"`
		CreatedAt time.Time `json:"created_at"`
	}

	if err := json.Unmarshal([]byte(out), &body); err != nil {
		return nil, fmt.Errorf("read what gh answered about the lines of %d: %w", number, err)
	}

	var found []timed

	for _, c := range body {
		if strings.TrimSpace(c.Body) == "" {
			continue
		}

		found = append(found, timed{
			at: c.CreatedAt,
			comment: Comment{
				Author: c.User.Login, Body: strings.TrimSpace(c.Body),
				Path: c.Path, Line: c.Line, URL: c.URL,
			},
		})
	}

	return found, nil
}
