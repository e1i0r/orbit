package engine

// agy, run for real — and the fake reading its own output the way the real
// engines read theirs.
//
// Both read zero. agy is the fourth adapter and the newest, so it is the one
// where a spec wired to the wrong parser would go unnoticed: every other
// engine's Run is exercised, and this one shares none of their code paths
// beyond the spec it hands over.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// standingIn puts a shell script named agy on PATH, printing those lines.
func standingIn(t *testing.T, lines string) string {
	t.Helper()

	dir := t.TempDir()

	script := "#!/bin/sh\ncat <<'EOF'\n" + lines + "\nEOF\n"
	if err := os.WriteFile(filepath.Join(dir, "agy"), []byte(script), 0o755); err != nil {
		t.Fatalf("write the stand-in: %v", err)
	}

	t.Setenv("PATH", dir+":/bin:/usr/bin")

	return dir
}

// TestAgyIsRunAndItsStreamIsRead. The spec is where an adapter is wired to
// its parser, and a spec pointing at the wrong one answers an empty result
// for a run that said plenty.
func TestAgyIsRunAndItsStreamIsRead(t *testing.T) {
	stream := `{"event":"init","conversation_id":"agy-sess","init":{"cwd":"/tmp"}}
{"event":"result","result":{"conversation_id":"agy-sess","status":"DONE",` +
		`"response":"the handler is refactored"}}`

	dir := standingIn(t, stream)

	out, err := NewAgy().Run(context.Background(), Request{
		Prompt:      "refactor the handler",
		Dir:         dir,
		Permissions: []string{PermissionRepo},
	})
	if err != nil {
		t.Fatalf("run agy: %v", err)
	}

	if out.Output != "the handler is refactored" {
		t.Errorf("it answered %q, want what the stream said", out.Output)
	}

	// The session id is what --conversation resumes with, and a run that
	// lost it is a run nobody can pick up.
	if out.SessionID != "agy-sess" {
		t.Errorf("the session is %q, want the one the stream opened with", out.SessionID)
	}
}

// TestAgyThatIsNotOnThisMachineSaysSo, rather than answering an empty run.
func TestAgyThatIsNotOnThisMachineSaysSo(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	req := Request{Prompt: "do the thing", Permissions: []string{PermissionRepo}}

	if _, err := NewAgy().Run(context.Background(), req); err == nil {
		t.Error("a machine with no agy on it ran one")
	}
}

// TestAFakeReadsItsOwnOutputTheWayARealEngineReadsTheirs. A test that wants
// a run that ran out writes the words a provider would, and a fake with a
// rule of its own would be a test agreeing with the fake rather than with
// the thing it stands in for.
func TestAFakeReadsItsOwnOutputTheWayARealEngineReadsTheirs(t *testing.T) {
	cases := []struct {
		name string
		out  Result
		err  error
		want bool
	}{
		{
			name: "the provider said the allowance is gone",
			out:  Result{Output: "Claude AI usage limit reached"},
			want: true,
		},
		{
			name: "it came back as an error instead",
			err:  errors.New("You've reached your usage limit"),
			want: true,
		},
		{
			name: "an ordinary answer",
			out:  Result{Output: "the handler is refactored"},
			want: false,
		},
		{
			name: "an ordinary failure",
			err:  errors.New("exit status 1"),
			want: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := &Fake{}

			if got := f.RanOut(c.out, c.err); got != c.want {
				t.Errorf("the fake reads it as ran-out %v, want %v", got, c.want)
			}

			// And the real engines answer the same thing about the same
			// words, which is the whole point of the fake not having a
			// rule of its own.
			if got := NewClaude().RanOut(c.out, c.err); got != c.want {
				t.Errorf("claude reads the same words as ran-out %v, want %v", got, c.want)
			}
		})
	}
}
