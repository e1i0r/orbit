package task

// cancel_test.go is at the file size ceiling, so the one branch it does not
// reach — Kill's own call to Alive failing, rather than Alive answering "not
// running" — lives here instead.

import "testing"

func TestKillAliveErrorPropagates(t *testing.T) {
	s, r := fixture(t)

	bad := Task{ID: "has/slash", Repo: r}
	if err := Kill(s, bad); err == nil {
		t.Error("Kill with a slash in the id should have failed")
	}
}
