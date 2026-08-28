package engine

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// A spec is how one program is driven, and it is the whole of what differs
// between the engines.
//
// The three adapters had a Run each and the three were the same sixty lines:
// tee stdout into a parser and a bounded buffer at once, wait for both, and
// decide which of the two halves is the answer. Only four things actually
// differ between claude, codex and opencode — where the program is, what its
// command line looks like, what environment it needs, and how its output is
// read — so those four are what a spec carries, and everything around them
// is written once, below, in run.
//
// This is the shape the package comment has always asked for read the other
// way round. A shim that flattened the engines into one command line would
// be the thing that comment forbids; a spec states each engine's differences
// out loud and in one place, so that "codex spells effort as a config
// override and opencode spells it --variant" is a line of code somebody can
// read rather than a fact buried in a sixty-line copy.
type spec struct {
	// name is what the record calls this engine, and also the program to
	// run: every engine so far ships a binary named after itself.
	name string

	// dirs are the places to look for that program when PATH does not have
	// it, relative to the reader's home directory.
	//
	// An engine's installer puts its binary somewhere and adds that
	// somewhere to a shell profile, which means a PATH is only as current
	// as the session that exported it. Orbit started from a terminal older
	// than the install sees a machine without the engine — and said so, on
	// the engine screen, to a reader with the engine open in the next pane.
	dirs []string

	// args turns a request into a command line, or refuses to.
	//
	// It refuses rather than dropping what it cannot express: a posture no
	// argv can state is a run that must not start, which is the rule the
	// whole of permission.go exists to keep.
	args func(Request) ([]string, error)

	// env is what to add to the environment for a run, empty for most.
	env func(Request) []string

	// parse reads what the program printed.
	//
	// Each engine gets its own, because each prints its own shape: session
	// id is session_id to claude, thread_id to codex and sessionID to
	// opencode, and a parser that knows one of those reads the other two as
	// noise. Pointing all three at claude's parser is what made every codex
	// and opencode run record no session and no cost.
	parse func(io.Reader, func(StreamEvent)) (Result, error)
}

// locate is the program this spec runs, as an absolute path.
//
// The answer is the same one whether it is asked to draw a screen or to
// start a run, which is the point of asking it here: the engine screen used
// to ask exec.LookPath and the run used to ask exec.Command, so a machine
// where those two disagreed got a dial it could not use or, worse, a dial
// that looked fine and a run that died. What is found here is what is
// executed, by absolute path.
func (s spec) locate() (string, error) {
	if path, err := exec.LookPath(s.name); err == nil {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("looking for %s: %w", s.name, err)
	}

	for _, dir := range s.dirs {
		candidate := filepath.Join(home, dir, s.name)
		if executable(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("%s is not installed, or not on PATH", s.name)
}

// executable is whether a path is a file this process could run.
func executable(path string) bool {
	info, err := os.Stat(path)

	return err == nil && !info.IsDir() && info.Mode().Perm()&0o111 != 0
}

// run starts the program this spec describes and returns what it reported.
//
// This is the single copy of what all three adapters did. stdout goes to two
// places at once: a parser, which turns the engine's own event stream into a
// Result and calls back as events arrive, and a bounded buffer, which is
// what is left to report if the parser cannot read the stream at all.
func (s spec) run(ctx context.Context, req Request) (Result, error) {
	args, err := s.args(req)
	if err != nil {
		return Result{}, s.wrap(req, err)
	}

	bin, err := s.locate()
	if err != nil {
		return Result{}, s.wrap(req, err)
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = req.Dir

	if s.env != nil {
		if extra := s.env(req); len(extra) > 0 {
			cmd.Env = append(cmd.Environ(), extra...)
		}
	}

	stdout := &boundedBuffer{max: maxStream}
	stderr := &boundedBuffer{max: maxStderr}
	pr, pw := io.Pipe()
	cmd.Stdout = io.MultiWriter(stdout, pw)
	cmd.Stderr = stderr

	var (
		streamResult Result
		parseErr     error
	)

	done := make(chan struct{})
	go func() {
		defer close(done)

		streamResult, parseErr = s.parse(pr, req.OnEvent)
		if closeErr := pr.Close(); closeErr != nil && parseErr == nil {
			parseErr = closeErr
		}
	}()

	runErr := cmd.Run()
	if closeErr := pw.Close(); closeErr != nil && runErr == nil {
		runErr = closeErr
	}

	<-done

	return s.report(req, streamResult, stdout, stderr, runErr, parseErr)
}

// report decides which of the two halves of a finished run is the answer.
//
// The rule is that nothing the parser did manage to read is thrown away.
// The old plain-text fallback returned Result{Output: raw} and dropped the
// session id, the thoughts and the tool calls that had already been parsed
// off the same stream — so a run whose last line arrived malformed reported
// as a run that had done nothing but print.
func (s spec) report(
	req Request, out Result, stdout, stderr *boundedBuffer, runErr, parseErr error,
) (Result, error) {
	raw := strings.TrimSpace(stdout.String())

	if runErr != nil {
		// The run died before the engine summarised it, so there is no
		// terminal result object and the raw stream is the only evidence
		// of what happened before it stopped.
		if parseErr != nil && out.Output == "" {
			out.Output = raw
		}

		out.Output = noteDropped(out.Output, stdout.dropped)
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return out, fmt.Errorf("%s in %q: %s: %w", s.name, req.Dir, msg, runErr)
		}

		return out, s.wrap(req, runErr)
	}

	if parseErr != nil {
		// The process exited zero and still said nothing this adapter
		// understands. If it printed something, that text is the honest
		// answer and it is kept alongside whatever was parsed; if it
		// printed nothing at all, the stream's shape has moved under us
		// and reporting it is the only way that is ever noticed.
		if raw == "" {
			return out, s.wrap(req, parseErr)
		}

		if out.Output == "" {
			out.Output = raw
		}
	}

	out.Output = noteDropped(out.Output, stdout.dropped)

	return out, nil
}

// wrap names the engine and the worktree on every error this package
// returns, because a failure a reader sees in the band has to say which of
// three programs failed and where.
func (s spec) wrap(req Request, err error) error {
	return fmt.Errorf("%s in %q: %w", s.name, req.Dir, err)
}
