package compose

// Where the task starts, which the form picks rather than asks.

import "testing"

// TestTheFormPicksWhereTheTaskStartsAndDoesNotAsk. The repository was the
// first field on this screen. What replaced it is not a smaller field or a
// better default — it is nothing at all, and this is the whole of what the
// form does instead.
func TestTheFormPicksWhereTheTaskStartsAndDoesNotAsk(t *testing.T) {
	e := world(t)

	// The task the cursor was on, because a reader writing a task while
	// looking at another one is usually writing about the same code.
	if got := Open("app", e).repoPath; got != "/r/app" {
		t.Errorf("the form starts in %q, want the checkout of the task the cursor was on", got)
	}

	// A checkout Orbit has never listed, which can only name itself by
	// where it is: the session a reader opened by hand somewhere else.
	if got := startsIn("/elsewhere/ledger", e); got != "/elsewhere/ledger" {
		t.Errorf("a checkout named by path resolved to %q, want the path itself", got)
	}

	// Nothing preferred and nothing to prefer: the first one Orbit knows.
	if got := startsIn("", e); got == "" {
		t.Error("the form found nowhere to start on a board with two repositories")
	}

	// A repository the board knows only by name is somewhere work happened
	// and not somewhere work can be started, so it is passed over.
	named := world(t)
	named.Places = []Place{{Name: "api"}, {Name: "app", Path: "/r/app"}}

	if got := startsIn("api", named); got != "/r/app" {
		t.Errorf("a repository with no checkout offered %q to start in", got)
	}

	// And a board with none has nothing to pick, which is not a refusal:
	// the task is written against no repository and joins its first
	// checkout in whichever phase the work reaches one.
	none := world(t)
	none.Places = nil

	if got := startsIn("payments", none); got != "" {
		t.Errorf("a board with no repositories offered %q to start in", got)
	}

	s := Open("payments", none)
	s.id.SetValue("ACME-9")
	s.text.SetValue("write the importer")

	_, out := s.Submit(false, none)
	if out.Write == nil {
		t.Fatalf("the form refused to write the task: %q", out.Said)
	}

	if out.Write.Repo != "" {
		t.Errorf("the task starts in %q on a board that has nowhere", out.Write.Repo)
	}

	// Where there is one, it travels: this is a task that starts nowhere,
	// not a form that stopped saying where.
	on := Open("app", e)
	on.id.SetValue("ACME-9")
	on.text.SetValue("write the importer")

	if _, out := on.Submit(false, e); out.Write == nil || out.Write.Repo != "/r/app" {
		t.Errorf("the task written is %+v, want the checkout the form picked", out.Write)
	}
}
