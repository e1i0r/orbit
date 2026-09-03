package repo

// What a reviewer said on the pull request, read back so a phase can answer
// it.

import (
	"encoding/json"
	"fmt"
	"strings"
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
// review bodies and the comments left on lines, oldest first.
//
// Unresolved ones only would be the better question and gh cannot answer it
// without the GraphQL API and a token scope Orbit does not ask for. Reading
// them all is the honest version of what can be read — a comment that was
// already dealt with reads as one more thing to check rather than as one
// nobody noticed, and that is the safe direction to be wrong in.
func (r Repo) ReviewComments(wtDir, branch string) ([]Comment, error) {
	out, err := gh(wtDir, "pr", "view", branch, "--json", "reviews,comments")
	if err != nil {
		return nil, fmt.Errorf("read the reviews of %q: %w", branch, err)
	}

	var body struct {
		Reviews []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			Body string `json:"body"`
			URL  string `json:"url"`
		} `json:"reviews"`
		Comments []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			Body string `json:"body"`
			URL  string `json:"url"`
			Path string `json:"path"`
			Line int    `json:"line"`
		} `json:"comments"`
	}

	if err := json.Unmarshal([]byte(out), &body); err != nil {
		return nil, fmt.Errorf("read what gh answered about %q: %w", branch, err)
	}

	var found []Comment

	for _, rv := range body.Reviews {
		if strings.TrimSpace(rv.Body) == "" {
			// An approval with no words is not a comment to answer. It is
			// the most common review there is, and listing it would put "\
			// nothing to do" in front of the things there are to do.
			continue
		}

		found = append(found, Comment{Author: rv.Author.Login, Body: strings.TrimSpace(rv.Body), URL: rv.URL})
	}

	for _, c := range body.Comments {
		if strings.TrimSpace(c.Body) == "" {
			continue
		}

		found = append(found, Comment{
			Author: c.Author.Login, Body: strings.TrimSpace(c.Body),
			Path: c.Path, Line: c.Line, URL: c.URL,
		})
	}

	return found, nil
}
