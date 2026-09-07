---
name: orbit-workflow
description: >-
  How a change is made in Orbit: the layer map and where it is written down, the order
  of work from reading to make check, worktree isolation for runs, and what never
  leaves the machine without being asked for.
---

# Making a change

## The layers

`internal/arch/layers_test.go` is the map, and every entry has the argument for it
written beside it. Read it there rather than from a copy: a table in a skill file goes
stale, and that one fails the build when it is wrong.

What the map is really made of is its absences. `internal/ui` cannot reach
`internal/record` or `internal/supervisor`, so the window can only learn things through
the ports it was handed; `internal/board` cannot reach `internal/supervisor`, so a
board refresh can never start a conversation. Adding an import between packages is a
decision to argue in the pull request, and the comment beside the new entry is where
it is argued.

## The order of work

1. **Read the doors.** `internal/arch/doors_test.go` lists what each package is entered
   by. It is the index: it tells you where the thing you need lives without opening
   twenty files.
2. **Read the file you are about to change, and its neighbours.** Match their shape —
   comment density, naming, how much is explained. A file that reads differently is one
   a reviewer has to learn twice.
3. **Write the change and the tests that would have caught it.** Not tests that agree
   with the code — tests that fail if it is wrong. Which kinds a change brings is in
   the `orbit-testing` skill: unit always, property-based where there is an invariant,
   fuzzing where bytes arrive from outside, integration where the seam is the subject,
   `make mutate PKG=...` on what you touched, and the adversarial case somebody wrote
   trying to break it.
4. **Say it in the reader's language.** Every user-facing sentence goes through
   `p.T("key", "the English")`, and `internal/words/lang/es.json` gets the same key with
   the same source. `TestEveryTranslationKeyIsHonest` checks that the two agree.
5. **Write it down.** Actionable failures and state changes go through
   `internal/logger` with a subsystem tag — `logger.Info("task/run", …)`. Log where the
   error stops travelling, and propagate it wrapped everywhere else. Never both.
6. **`make coverage`.** It fails under 90% — a package that arrived without its own
   suite is what usually spends it.
7. **`make check`.** Read its exit status, not the output of something you piped it
   into. It runs gofmt, vet twice (once for linux), golangci-lint, every test and
   `go mod tidy`.

## Runs and worktrees

A task runs in a git worktree of its own under the state root, on a branch named after
the task. That is what lets several tasks touch the same repository at once, and it is
why `orbit run` takes `-repo` rather than reading the current directory: a run that
inherits whatever directory the window happened to be in is a run that writes into the
wrong checkout.

`task.Start` spawns the orbit binary again, in its own process group, with the state
root passed in the environment rather than inherited. A test suite whose code can reach
`Start` must guard its `TestMain` — under `go test`, `os.Executable()` is the test
binary.

## What never happens without being asked

Never `git push`, never open or merge a pull request, and never change repository
settings. Work stays local and is committed when the operator says so. They read the
diff before anything leaves the machine — that is the whole arrangement, and Orbit is a
tool for people who want to keep it.
