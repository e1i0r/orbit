package mcp

// What a model asks for when it is wrong about the machine.
//
// This is the surface an agent drives, and an agent is the adversarial case
// by construction: it has never seen the board, it invents ids that look
// plausible, it calls the same tool twice because the first answer scrolled
// past, and it reaches for a checkout nobody told it about. None of that is
// malice and all of it arrives.
//
// What every case below asserts is the same thing twice: the tool refuses,
// and the refusal says enough for the model to do something else. A "no"
// with nothing in it is a model that tries the same call again — and a
// refusal the model reads as a success is worse than either, because it goes
// on to tell somebody the work is done.

import (
	"strings"
	"testing"
)

// nothingHappened is the tools that change something. Each one is asked for
// with an id nothing knows, and each one has to say so.
func nothingHappened() []struct {
	tool string
	args map[string]any
} {
	return []struct {
		tool string
		args map[string]any
	}{
		{"orbit_inspect_task", map[string]any{"task_id": "GHOST-1"}},
		{"orbit_retry_task", map[string]any{"task_id": "GHOST-1"}},
		{"orbit_pause_task", map[string]any{"task_id": "GHOST-1"}},
		{"orbit_cancel_task", map[string]any{"task_id": "GHOST-1"}},
		{"orbit_requeue_task", map[string]any{"task_id": "GHOST-1"}},
		{"orbit_add_note", map[string]any{"task_id": "GHOST-1", "text": "anything"}},
		{"orbit_direct_task", map[string]any{"task_id": "GHOST-1", "message": "carry on"}},
	}
}

// TestATaskTheModelInventedIsRefusedByName.
//
// An agent that has not read the board guesses an id that looks like the
// ones it has seen. Every tool that acts on a task has to say it does not
// know that one, in words carrying the id — a refusal that did not say back
// what was asked for is one the model cannot tell from a machine that is
// down.
func TestATaskTheModelInventedIsRefusedByName(t *testing.T) {
	for _, c := range nothingHappened() {
		t.Run(c.tool, func(t *testing.T) {
			_, sn, _ := oneRepo(t)

			said := refused(t, sn, c.tool, c.args)

			if !strings.Contains(said, "GHOST-1") {
				t.Errorf("it said %q, want the id it was asked about", said)
			}
		})
	}
}

// TestATaskWithNoIdAtAllIsRefused. An agent that filled the argument in from
// an empty variable sends the empty string, and a tool that read it as "any
// task" would act on whatever sorted first.
func TestATaskWithNoIdAtAllIsRefused(t *testing.T) {
	for _, c := range nothingHappened() {
		t.Run(c.tool, func(t *testing.T) {
			_, sn, _ := oneRepo(t)

			args := map[string]any{}
			for k, v := range c.args {
				args[k] = v
			}

			args["task_id"] = ""

			refused(t, sn, c.tool, args)
		})
	}
}

// TestAnIdThatIsAPathIsNotAWayOutOfTheStateRoot. `../../etc/passwd` looks
// like an id to something that has only ever seen ids, and a tool that
// joined it to a directory would be reading whatever it landed on.
func TestAnIdThatIsAPathIsNotAWayOutOfTheStateRoot(t *testing.T) {
	_, sn, _ := oneRepo(t)

	for _, id := range []string{"../../etc/passwd", "..", "./.", "a/b/c"} {
		said := refused(t, sn, "orbit_inspect_task", map[string]any{"task_id": id})
		if said == "" {
			t.Errorf("%q was refused with nothing said", id)
		}
	}
}

// TestAnArgumentOfTheWrongShapeIsRefusedRatherThanCoerced. A model sends a
// number where a name goes, or an object where a string goes, and a tool
// that took whatever fmt made of it would act on the word "map[]".
func TestAnArgumentOfTheWrongShapeIsRefusedRatherThanCoerced(t *testing.T) {
	_, sn, _ := oneRepo(t)

	wrong := []map[string]any{
		{"task_id": 42},
		{"task_id": map[string]any{"id": "ACME-1"}},
		{"task_id": []any{"ACME-1"}},
		{"task_id": true},
		{"task_id": nil},
	}

	for _, args := range wrong {
		res := sn.Call("orbit_inspect_task", args)
		if !res.IsError {
			t.Errorf("a task_id of %T was acted on: %s", args["task_id"], text(t, res))
		}
	}
}

// TestABandTheModelMadeUpIsRefusedWithTheOnesThereAre. The four bands are a
// closed list and an agent guesses "in_progress"; a filter that silently
// matched nothing would have it report an empty board.
func TestABandTheModelMadeUpIsRefusedWithTheOnesThereAre(t *testing.T) {
	_, sn, _ := oneRepo(t)

	said := refused(t, sn, "orbit_list_tasks", map[string]any{"band": "in_progress"})

	if !strings.Contains(said, "in_progress") {
		t.Errorf("it said %q, want the band it was asked for", said)
	}

	for _, real := range bandNames() {
		if !strings.Contains(said, real) {
			t.Errorf("it said %q, want %q among the bands there are", said, real)
		}
	}
}

// TestACheckoutTheBoardHasNeverHeardOfIsTheCallersOwnWords. A path the board
// does not know matches nothing, which is the true answer — and what comes
// back is what the model asked for rather than a name this package invented.
func TestACheckoutTheBoardHasNeverHeardOfIsTheCallersOwnWords(t *testing.T) {
	_, sn, r := oneRepo(t)

	sb, err := sn.readBoard()
	if err != nil {
		t.Fatalf("read the board: %v", err)
	}

	defer sb.close()

	if got := repoNamed(sb.board, r.Path); got != r.Name {
		t.Errorf("the checkout at %q is called %q, want %q", r.Path, got, r.Name)
	}

	if got := repoNamed(sb.board, "/nowhere/at/all"); got != "/nowhere/at/all" {
		t.Errorf("a path the board never heard of came back as %q, want the caller's own words", got)
	}
}

// TestTheSameToolCalledTwiceAnswersTwice. An agent calls again because the
// first answer scrolled past its context, and a tool that failed the second
// time — or acted twice — would make a retry a bug.
func TestTheSameToolCalledTwiceAnswersTwice(t *testing.T) {
	_, sn, _ := oneRepo(t)

	for _, tool := range []string{"orbit_get_board_summary", "orbit_list_tasks", "orbit_list_repos"} {
		t.Run(tool, func(t *testing.T) {
			first := call(t, sn, tool, map[string]any{})
			second := call(t, sn, tool, map[string]any{})

			if len(first) != len(second) {
				t.Errorf("the same question answered %d fields and then %d", len(first), len(second))
			}
		})
	}
}

// TestABandThisPackageDoesNotKnowIsSaidByItsNumber. An invention would be
// worse than an obviously wrong token: a slug this build made up for a band
// a newer one wrote is a filter that matches the wrong rows silently.
func TestABandThisPackageDoesNotKnowIsSaidByItsNumber(t *testing.T) {
	said := bandSlug(99)

	if said == "" {
		t.Fatal("a band this build does not know is named nothing")
	}

	for _, known := range bandNames() {
		if said == known {
			t.Errorf("a band this build does not know is called %q, which is a real one", said)
		}
	}
}
