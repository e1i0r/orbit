package task

// The branch a task's work is done on, named once and read back from the
// record.
//
// It used to be `"orbit/" + id`, spelled out in four files, and that had
// two faults a single evening found both of them.
//
// A name that is a pure function of the id is a name a second task with
// the same id inherits. Deleting a task takes its worktree away and leaves
// the branch, so writing FRA-128 again put a new run in a worktree
// checked out on a stranger's three commits — and the engine, finding work
// already there, started cherry-picking it. The run was not from scratch
// and nothing on any screen said so.
//
// Four spellings of one name is the other. Three of them would keep
// working while the fourth opened a pull request against a branch that
// does not exist, and nothing in the build could say which.
//
// So: one function, and a suffix nothing can guess. The suffix is written
// into task.created and read back from there, so every attempt at one task
// agrees about it, and a task written again under an id somebody used
// before gets a branch of its own.

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/e1i0r/orbit/internal/record"
	"github.com/e1i0r/orbit/internal/store"
)

// branchPrefix is what marks a branch as Orbit's, so a reader listing
// branches can tell the ones a run made from the ones they made.
const branchPrefix = "orbit/"

// suffixBytes is how much randomness the name carries. Four hex characters
// out of two bytes: enough that rewriting the same id twice in one evening
// does not collide, and short enough that the branch is still something a
// person reads out loud.
const suffixBytes = 2

// Branch is the branch this task's work is on.
//
// A task whose record carries no branch is one written before names had a
// suffix, and its branch is what it has always been. That fallback is not
// tidiness: those tasks have branches pushed and pull requests open
// against them, and a new name would orphan both.
func Branch(t Task) string {
	if named := strings.TrimSpace(t.BranchName); named != "" {
		return named
	}

	return branchPrefix + t.ID
}

// newBranch is the name a task being written down now gets.
//
// A failure to read the machine's randomness answers the plain name rather
// than an error. It is the name this program used for its whole life, the
// collision it risks needs somebody to have deleted a task and written the
// same id again, and refusing to create a task over it would be trading a
// rare confusion for a certain one.
func newBranch(id string) string {
	b := make([]byte, suffixBytes)
	if _, err := rand.Read(b); err != nil {
		return branchPrefix + id
	}

	return branchPrefix + id + "-" + hex.EncodeToString(b)
}

// writtenBranch is the name in this task's own record, and empty for a task
// written before there was one. It reads the same way writtenFlow does.
func writtenBranch(s *store.Store, t Task) string {
	events, err := Events(s, t)
	if err != nil {
		return ""
	}

	name := ""

	for _, e := range events {
		if e.Kind != record.TaskCreated {
			continue
		}

		if recorded, ok := e.Data["branch"]; ok {
			name = recorded
		}
	}

	return name
}
