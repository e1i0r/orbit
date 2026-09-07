---
name: orbit-testing
description: >-
  How Orbit is tested: a package carries its own suite, tests are named after the
  behaviour they hold, 90% coverage, fuzzing for every parser, golden files for the
  window, and the architecture tests that fail a build rather than a review.
---

# Testing

A test here says what the software promises. `TestADeletedIdIsFreeAgain` is a
sentence somebody can check; `TestDelete` is a file name. Name the behaviour, and
when it breaks the failure will already say what was lost.

## What a test looks like

```go
func TestAnIssueWithNoBodyIsAskedForOnce(t *testing.T) {
	f := newFile()

	next, _ := m.tookIssue(issueReadMsg{id: "FRA-71", issue: tracker.Issue{ID: "FRA-71"}})

	if after := asModel(t, next); after.needsBody() {
		t.Error("the form would ask the tracker about that issue again")
	}
}
```

- Table-driven where there are cases, with the table naming each one.
- No assertion library: `if got != want { t.Errorf(...) }` says everything.
- The message names what was expected and what happened, in the words of the thing
  under test — not `assert failed`.
- A comment above the test says what went wrong the day it was written, if something
  did. That is why the test exists and it is what a reader needs first.

## Where a test lives

**A package carries its own suite.** A screen that is a package is tested by building
its `Env` and calling its doors — no window, no `Model`:

```go
e := env(t, newFile())
s, out := Open(e).Point(1).Key(tea.KeyPressMsg{Code: tea.KeyRight}, e)
```

What genuinely tests the window's wiring — a click reaching a verb, a golden frame —
stays in `internal/ui`.

## The six kinds, and when each applies

Every change brings the ones that fit it. Most bring two or three.

1. **Unit** — always. One behaviour, named after itself, in the package that owns it.
2. **Property-based** — where there is a law rather than an answer: a round-trip
   through the store is lossless, a task's cost never decreases, layout never returns
   a negative coordinate or a row wider than the terminal. Generate the inputs and
   assert the law.
3. **Fuzzing** (`testing.F`) — where bytes arrive from outside: engine streams, event
   decoders, quota readings, terminal fitters. They must never panic.
   ```bash
   make fuzz PKG=./internal/engine/... FOR=2m
   ```
   Commit what the corpus finds: a crash found once is a case the suite keeps.
4. **Integration** — where the seam is the subject: a task walking a whole flow with
   its gates and holds, a command that opens the store and writes the record, the
   window driven through its ports. Not a unit test in a costume.
5. **Mutation** — on the package the change touched, before the pull request:
   ```bash
   make mutate PKG=./internal/ui/settings/...
   ```
   It rewrites the code under the tests and asks whether they notice. A mutant that
   lives is a statement no test disagrees with — kill it, or write down why it does
   not matter. This is the check that coverage cannot make: a line can run and prove
   nothing.
6. **Adversarial** — the case somebody wrote trying to break it. Empty input. The same
   message twice. An answer that arrives after the reader pressed esc. A file deleted
   between the listing and the read. A value that is legal and absurd — a negative
   cap, a flow named `../etc/passwd`. Nearly every bug this project has shipped was one
   of these, and each is now a test with a comment saying what happened.

Golden frames for the window are regenerated on purpose:
```bash
go test ./internal/ui -update
```

## Environment

A suite decides what unset means before it runs: `internal/cli` and `internal/task`
unset `ORBIT_TASK`, `ORBIT_WORKSPACE` and `ORBIT_HOME` in `TestMain`, because a test
inside an Orbit run would otherwise inherit the run's workspace and read the operator's
real checkouts. A test that wants one set uses `t.Setenv`, which is scoped to it.

A suite whose code can start a run guards `TestMain` against being the child:

```go
if len(os.Args) > 1 && os.Args[1] == "run" {
	os.Exit(0)
}
```

`task.Start` spawns `os.Executable()`, which under `go test` is the test binary — and
the flag package stops at the first non-flag argument, so without the guard the child
runs the whole suite again, including the test that started it.

## Running it

```bash
make check      # fmt, vet (twice), lint, tests, tidy — read its exit status
make coverage   # fails under 90% total: a gate, not a report
go test -race ./...
```
