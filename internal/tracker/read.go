package tracker

// Reading what an issue actually says.
//
// Parsing a URL gives an id, and a slug gives something like a title. Neither
// gives the body — and the body is the task. A run started from a URL alone
// is handed a note telling it to go and look the issue up, which is a thing
// an engine can do at a terminal where somebody approves its tool calls, and
// cannot do in a headless run where those calls are auto-denied. It then
// either invents the requirements or does nothing; both cost a run.
//
// So the body is fetched here when there is a key to fetch it with, and when
// there is not, Read says so plainly and the window refuses to write a task
// whose whole content is an instruction nobody can follow.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// ErrNoKey says the issue could be named but not read: there is no
// credential for the tracker it lives in.
var ErrNoKey = errors.New("no api key for this tracker")

// readTimeout is how long the fetch is given. The reader is sitting in front
// of a form waiting for it.
const readTimeout = 10 * time.Second

// Read is the issue with its title and body filled in, as far as this
// machine can see them.
//
// What comes back is always usable: on any failure the issue is returned as
// it was parsed, with the error beside it, so a caller can write the task
// anyway and say what is missing.
func Read(ctx context.Context, iss Issue) (Issue, error) {
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	// One tracker so far. Linear is the one with a credential this program
	// knows how to use; the others are named by their URLs and read by
	// whoever writes the task.
	if !strings.EqualFold(iss.Kind, "linear") {
		return iss, fmt.Errorf("%s: %w", iss.Kind, ErrNoKey)
	}

	key := strings.TrimSpace(os.Getenv("LINEAR_API_KEY"))
	if key == "" {
		return iss, fmt.Errorf("linear %s: %w", iss.ID, ErrNoKey)
	}

	title, body, err := FetchLinear(ctx, key, iss.ID)
	if err != nil {
		return iss, err
	}

	if title != "" {
		iss.Title = title
	}

	iss.Description = body

	return iss, nil
}

// Readable reports whether this machine has what it needs to read the body
// of an issue of this kind. It is what a form asks before it offers to.
func Readable(kind string) bool {
	if strings.EqualFold(kind, "linear") {
		return strings.TrimSpace(os.Getenv("LINEAR_API_KEY")) != ""
	}

	return false
}
