package engine

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/e1i0r/orbit/internal/logger"
)

// logs is the log and the errors file, as text, after fn has run against a
// logger of its own. The logger is a package-level global, so no test here
// runs in parallel with another.
func logs(t *testing.T, fn func()) (string, string) {
	t.Helper()

	dir := t.TempDir()
	all, bad := filepath.Join(dir, "orbit.log"), filepath.Join(dir, "errors.log")

	if err := logger.Init(all, bad); err != nil {
		t.Fatalf("logger.Init: %v", err)
	}

	fn()

	if err := logger.CloseGlobal(); err != nil {
		t.Fatalf("logger.CloseGlobal: %v", err)
	}

	return read(t, all), read(t, bad)
}

func read(t *testing.T, path string) string {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(b)
}

// toy is a spec that runs a shell script this test wrote, named toy and put
// on PATH. It stands in for the three real engines because what is under
// test here is run, which is the one copy all three of them share.
func toy(t *testing.T, script string) spec {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "toy"), []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	t.Setenv("PATH", dir+":/bin:/usr/bin")

	return spec{
		name: "toy",
		args: func(Request) ([]string, error) { return nil, nil },
		parse: func(r io.Reader, _ func(StreamEvent)) (Result, error) {
			b, err := io.ReadAll(r)
			if err != nil {
				return Result{}, err
			}

			return Result{
				Output:    strings.TrimSpace(string(b)),
				SessionID: "sess-1",
				Cost:      0.25,
				Thoughts:  []string{"one", "two"},
			}, nil
		},
	}
}

// TestWhatWasRunIsWrittenDownBeforeItIsWaitedFor. The binary is the whole
// point of the line: a machine with two claudes installed answers "which of
// them was it" with an absolute path and with nothing else.
func TestWhatWasRunIsWrittenDownBeforeItIsWaitedFor(t *testing.T) {
	s := toy(t, "echo hello")
	dir := t.TempDir()

	all, bad := logs(t, func() {
		req := Request{
			Dir:         dir,
			Model:       "gpt-5.6",
			Effort:      "high",
			Permissions: []string{PermissionRead},
			Prompt:      "12345678",
		}
		if _, err := s.run(context.Background(), req); err != nil {
			t.Errorf("run: %v", err)
		}
	})

	for _, want := range []string{
		"[INFO] [engine/toy] running ",
		"/toy in " + strconv.Quote(dir),
		`model="gpt-5.6"`,
		`effort="high"`,
		"permissions=[read]",
		"resume=false",
		"prompt=8B",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("the log does not say %q:\n%s", want, all)
		}
	}

	if bad != "" {
		t.Errorf("a run that worked wrote to the errors file:\n%s", bad)
	}
}

// TestWhatCameBackIsWrittenDownToo, and counted rather than copied: cost and
// duration are the two facts about a run that live nowhere else at all.
func TestWhatCameBackIsWrittenDownToo(t *testing.T) {
	s := toy(t, "echo hello")

	all, _ := logs(t, func() {
		if _, err := s.run(context.Background(), Request{Dir: t.TempDir()}); err != nil {
			t.Errorf("run: %v", err)
		}
	})

	want := `[INFO] [engine/toy] answered after `
	if !strings.Contains(all, want) {
		t.Fatalf("the log does not say %q:\n%s", want, all)
	}

	for _, part := range []string{
		`session="sess-1"`, "cost=0.2500", "output=5B", "thoughts=2", "tools=0", "refusals=0",
	} {
		if !strings.Contains(all, part) {
			t.Errorf("the log does not say %q:\n%s", part, all)
		}
	}
}

// TestTheWaitIsTimedFromBeforeTheRunAndNotAfterIt. A duration measured on
// the wrong side of cmd.Run is a number that is always zero, reads like a
// fact, and would have made every engine in this file look instantaneous.
func TestTheWaitIsTimedFromBeforeTheRunAndNotAfterIt(t *testing.T) {
	s := toy(t, "sleep 0.2\necho hello")

	all, _ := logs(t, func() {
		if _, err := s.run(context.Background(), Request{Dir: t.TempDir()}); err != nil {
			t.Errorf("run: %v", err)
		}
	})

	// The unit is whatever the duration printed itself as, and not
	// milliseconds: a machine busy running the rest of this suite takes
	// more than a second over the same sleep, and Go writes that as "1.4s".
	// A test that demanded "ms" failed there while nothing was wrong, which
	// is the one thing a test must never do.
	m := regexp.MustCompile(`answered after ([^ ]+): `).FindStringSubmatch(all)
	if m == nil {
		t.Fatalf("no duration in the log:\n%s", all)
	}

	took, err := time.ParseDuration(m[1])
	if err != nil {
		t.Fatalf("ParseDuration(%q): %v", m[1], err)
	}

	if took < 150*time.Millisecond {
		t.Errorf("a run that slept 200ms was logged as %s", took)
	}
}

// TestAnEngineThatDiedSaysWhyInTheErrorsFile, in the words the program
// itself used: stderr is the half of a failed run that says what to do
// about it, and it is nowhere in the record either.
func TestAnEngineThatDiedSaysWhyInTheErrorsFile(t *testing.T) {
	s := toy(t, "echo 'no credit left' >&2\nexit 3")

	_, bad := logs(t, func() {
		if _, err := s.run(context.Background(), Request{Dir: t.TempDir()}); err == nil {
			t.Error("run of a script that exited 3 returned no error")
		}
	})

	for _, want := range []string{
		"[ERROR] [engine/toy] failed after ", "no credit left", "exit status 3",
	} {
		if !strings.Contains(bad, want) {
			t.Errorf("the errors file does not say %q:\n%s", want, bad)
		}
	}
}

// TestARunThatNeverStartedIsNotARunThatFailed.
//
// Two things are being said here. The first is that a missing binary
// reaches the errors file at all: it is the single most common failure a
// new reader hits, and before this it was returned to the caller and
// written down nowhere. The second is that it does not read as a run — no
// engine was started, no money was spent, and "failed after 1ms" would have
// somebody looking for a model's answer that never existed.
func TestARunThatNeverStartedIsNotARunThatFailed(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	missing := spec{name: "toy", args: func(Request) ([]string, error) { return nil, nil }}
	refuses := spec{
		name: "toy",
		args: func(Request) ([]string, error) { return nil, errors.New("no sandbox says that") },
	}

	for _, c := range []struct {
		s    spec
		want string
	}{
		{missing, "is not installed, or not on PATH"},
		{refuses, "no sandbox says that"},
	} {
		all, bad := logs(t, func() {
			if _, err := c.s.run(context.Background(), Request{Dir: t.TempDir()}); err == nil {
				t.Errorf("run returned no error, want %q", c.want)
			}
		})

		if !strings.Contains(bad, "[ERROR] [engine/toy] did not start: ") || !strings.Contains(bad, c.want) {
			t.Errorf("the errors file does not say why nothing ran:\n%s", bad)
		}

		for _, absent := range []string{"running ", "failed after ", "answered after "} {
			if strings.Contains(all, absent) {
				t.Errorf("a run that never started was logged as %q:\n%s", absent, all)
			}
		}
	}
}

// TestNeitherThePromptNorTheAnswerIsCopiedIntoTheLog. Both are already in
// the record, in full, and both are pages long. A log with a page in it is
// a log nobody greps twice.
func TestNeitherThePromptNorTheAnswerIsCopiedIntoTheLog(t *testing.T) {
	s := toy(t, "echo the-answer-in-full")

	all, bad := logs(t, func() {
		req := Request{Dir: t.TempDir(), Prompt: "the-prompt-in-full"}
		if _, err := s.run(context.Background(), req); err != nil {
			t.Errorf("run: %v", err)
		}
	})

	for _, secret := range []string{"the-prompt-in-full", "the-answer-in-full"} {
		if strings.Contains(all+bad, secret) {
			t.Errorf("%q was copied into the log:\n%s", secret, all+bad)
		}
	}
}
