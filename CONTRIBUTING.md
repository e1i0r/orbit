# Contributing to Orbit

Orbit is a cockpit for supervising coding agents. It is small on purpose and it is meant to be read: somebody arriving at a file should be able to tell what it is for, why it is shaped that way, and what would break if they changed it.

What follows is how the code is written here. Most of it is enforced by `make check` — the test that holds each rule is named beside it. What is not enforced is judgement, and judgement is what review is for.

Everything in the repository is written in English: code, comments, commits, pull requests.

---

## How we work

This is a small codebase that intends to stay readable for years, so the way it is built is itself a decision rather than a habit. Six things describe it, and everything further down is one of them made concrete.

1. **Every rule here was argued, and most of them are tests.** Nothing in this file is a preference somebody typed once: where a rule can fail a build it does, and the test that holds it is named beside it. Where it cannot, the reason is written next to the rule so the next person can disagree with the reason rather than with the rule.
2. **The code says why, not just what.** A comment states the fact about the code — what happens, what would break without it, what was weighed and rejected. Anyone can read *what* the code does; only the person who wrote it knows why it is not the other thing, and that is the part that rots first if it is not written down.
3. **Packages are subjects with doors.** Each one is a thing the world asks something of, entered through the files that export its names, with everything else inside it lowercase. A directory listing is meant to read as an index of what the thing does.
4. **The interface belongs to whoever needs it.** Two or three methods, declared where they are used. Wide interfaces are ports, they are written down as such, and they shrink.
5. **Nothing is handled twice and nothing is hidden.** An error is wrapped and passed on, or logged and stopped — never both. What the operator can see, they can see the whole of.
6. **A change comes with the tests that would have caught it, and coverage never falls below 90%.** Not tests that agree with the code: tests that would fail if the code were wrong, which is a different and harder thing. That is what the mutation runs are for.

To read further: [Practical Go](https://dave.cheney.net/practical-go).

---

## What Orbit promises the person watching

Orbit exists to be believed. Everything below follows from that, and none of it is negotiable:

- **Nothing is hidden.** No error is swallowed, softened, or replaced by a cheerful sentence. If a check failed, the window says it failed and what it said. If Orbit does not know something, it says it does not know rather than guessing on the reader's behalf.
- **Everything the operator can see, they can see the whole of.** A verb that starts something says so when it starts and again when it lands. A long wait says what is being waited for. A key that cannot be pressed says why, in words, in the reader's own language.
- **Everything is written down.** Every actionable failure and every state change goes through `internal/logger` with its subsystem tag — `logger.Error("cli/run", …)` — into `orbit.log` and `errors.log`. Nothing writes to stdout or stderr behind the window's back: that corrupts the terminal and loses the record at once.
- **Said once, said plainly.** A sentence on screen is short, direct and in the reader's language: what happened, what it means, what to do. No apologies, no adjectives, no exclamation marks. The band has one line and the reader is mid-task.
- **The record is verbatim.** What an engine said is written down as it said it. Nothing here paraphrases a tool's output, and a summary always sits beside the thing it summarises rather than replacing it.

Two measurements hold the shape of the code itself:

- **300 lines per file**, code and comment, blanks excluded (`TestNoFileOverTheCeiling`).
- **100 columns per line** of code. A sentence a person reads — a translated string, a prompt, a URL — is exempt, because breaking one to fit a column makes it worse. `TestCodeStaysInsideTheColumn` holds this as a ratchet: the count it allows is the count that exists today, and it may go down but never up.

---

## The shape of a package

**A package is entered through its doors.** A door is an action the world can ask for — `Open`, `Key`, `Apply`, `View` — or the vocabulary those actions share. Every other file in the directory is a satellite: it holds the workings of one door, is named after it, and exports nothing.

```
internal/ui/settings/
  settings.go   State, Env, Store, Out       ← the vocabulary the doors share
  rows.go       Rows                          ← the table, which the mouse also needs
  key.go        Key, Cycle, Edit              ← the keyboard
  apply.go      Apply                         ← writing one setting down
  view.go       View                          ← drawing it
```

Go has no visibility inside a package — every file sees every other — so this would be a convention nobody could hold. It is a test instead: `TestEveryExportedNameIsBehindADoor` fails when a satellite exports anything. The list of doors per package lives in `internal/arch/doors_test.go`, and it doubles as the index of the package: reading it tells you what the thing does without opening a file.

**Only doors export.** Inside a package everything else is lowercase — a satellite that exports something is a second entrance nobody agreed to, and `TestEveryExportedNameIsBehindADoor` says so by name. How many doors a package has is decided by how many things the world asks it for: one for a screen that only draws itself, five for one that is opened, typed into, clicked on, written through and drawn.

**And the meeting place is a file too.** The window does not scatter calls into a package across the twenty files that happen to need them: one file — `internal/ui/screens.go` — builds the little world each screen was written against and does what the screen asked for when it hands the answer back. Several such files when the subjects are genuinely different, one while there is one subject. What that file may not do is decide: a branch in there is a decision living one package away from the one that made it, and the first bug it causes is looked for in the wrong place.

Two decisions about this shape, written down so nobody has to wonder whether they were an oversight. **Packages here are smaller than "fewer, larger packages" would suggest**: `internal/ui` was one package of 163 files, and a name collision anywhere in it was a collision everywhere — but never one package per type, which is the thing that advice is warning about. And **one mutable global stays**: the theme in `internal/ui/theme`. A palette belongs to the process, not to a window, and threading it through every draw call would put a parameter on hundreds of functions to make one global honest. It is the only one in the drawing path, and saying so here is what keeps it the only one.

A package is named for what it provides, never for what it contains. There is no `util`, no `common`, no `helpers` — `TestNoPackageIsADrawer` fails on those names, because a drawer is where code goes when nobody has decided what it is.

**Files stay under 300 lines** of code and comment, blank lines excluded (`TestNoFileOverTheCeiling`). A file over the ceiling is not a formatting problem; it is two subjects that have not been told apart yet. Split it along the seam, not at line 300.

**Which packages may import which** is written down in `internal/arch/layers_test.go`, with the argument for each entry beside it. The load-bearing part of that map is the absences: `internal/ui` cannot reach `internal/record` or `internal/supervisor`, so the window can only learn things through the ports it was handed. Adding an import that is not on the list fails `TestImportsFollowTheLayers`; adding it to the list is a decision, and the comment next to it is where that decision is argued.

---

## Interfaces

**Small, and declared by whoever needs them.** An interface is a promise the caller has to keep, and every method on it is one more thing a test's fake has to answer. Six methods is the ceiling (`TestInterfacesStaySmall`); an interface that wants a seventh usually wants to be two.

Declare an interface where it is *used*, not where it is implemented. The settings screen says what it needs of a settings file:

```go
// reader is the settings the table shows.
type reader interface {
	Language() string
	Autopilot() bool
	...
}
```

and the window passes whatever satisfies it. The file's own shape stays its own business, and a door that can only read cannot write by accident.

The wide interfaces that remain are the ports, and each one is written down in `wideInterfaces` with the reason it is allowed to be wide. They shrink as screens move into packages of their own: a screen that declares the three methods it needs is three methods the port no longer has to carry.

**Accept interfaces, return structs.** A function that takes `Reader` and returns `*Task` can be used by callers who have neither.

---

## Errors

**Wrap what you did not raise:** `fmt.Errorf("read the board: %w", err)`. The sentence says what was being attempted, not what went wrong — the wrapped error already says that.

**Handle an error once.** Logging it *and* returning it is handling it twice, and the reader of the log cannot tell whether anybody acted on it. The rule here:

- Log where the error stops travelling: the command that decides what happens, the run that gives up, the window that puts a sentence in the band.
- Propagate everywhere else, wrapped, and say nothing.

Every log line carries its subsystem: `logger.Error("cli/run", ...)`, `logger.Warn("engine/claude", ...)`. `internal/logger` writes to `orbit.log` and `errors.log`; nothing writes to stderr behind the window's back.

**Never discard a return.** `_ = f()` is only allowed where the answer genuinely cannot be acted on, and it carries a `//nolint:errcheck // reason` saying why in words — "the answer is already made", not "ignore".

---

## Naming

- **Packages**: one lowercase word, no underscores, named for what they provide. `theme`, `patch`, `typing`, `spoken`.
- **No stutter.** `cells.Fit`, never `cells.CellsFit`. If the package name is in the identifier, one of the two is wrong.
- **No `Get`.** A getter is named after what it answers: `Language()`, not `GetLanguage()` (`TestNoGetterSaysGet`).
- **Length follows scope.** A loop variable is `i`; a package-level function is a sentence's worth of name. A receiver is one or two letters and the same letters on every method of the type.
- **Files** are named for what a reader would look for: the door they open, or the subject they hold. A satellite is named after its door — `overview.go` and `overview_blocks.go`, not `blocks.go`.

---

## Comments

Comments explain **why**, and state facts rather than verdicts. `// the flag package stops at the first non-flag argument, so the child read none of what followed` is a comment; `// this was a terrible bug` is not.

Every exported name has a comment that starts with its own name. Where a decision was made — an alternative weighed and rejected, a constant that came from measurement, a guard that exists because of something that actually happened — the comment says so, because the next reader's first question is "why is this here" and the code cannot answer it.

---

## Tests

How they are written, first:

- **Table-driven**, with the table naming its cases. No assertion frameworks: `if got != want { t.Errorf(...) }` says everything.
- **Name the behaviour, not the function**: `TestADeletedIdIsFreeAgain`, not `TestDelete`.
- The failure message says what was expected and what happened, in the words of the thing under test.
- A package carries its own suite. A screen that is a package is tested by building its `Env` and calling its doors — no window, no `Model`.
- A test that needs the environment sets it with `t.Setenv`; a suite that must not inherit one says so in `TestMain`.
- **Coverage is `≥ 90%`, and `make coverage` fails under it** — it is a gate, not a report. A number printed and ignored is a number that drifts, and the day somebody notices it is 60% nobody knows which change spent it. Coverage says a line *ran*, which is the floor and not the goal: what says the line *matters* is the mutation run.

### The six kinds, and when each one applies

Every change brings the kinds that fit it. Most bring two or three; nothing brings all six every time, and a change that brings none is a change nobody can refactor later.

1. **Unit** — always. One behaviour, named, in the package that owns it.
2. **Property-based** — whenever something has an invariant rather than an answer: a round-trip that must be lossless, a cost that may never decrease, a layout that may never produce a negative coordinate or a row wider than the terminal. Assert the law over generated inputs, not one example of it.
3. **Fuzzing** (`testing.F`) — whenever bytes arrive from outside: engine streams, event decoders, quota readings, anything that parses. It must never panic on arbitrary input. `make fuzz PKG=./internal/engine/... FOR=2m`, and the corpus entries it finds are committed — a crash found once is a case the suite keeps.
4. **Integration** — when the thing being changed is the seam between parts: a task walking a whole flow, a command that opens the store and writes the record, a window driven through its ports. Only where the seam is the subject; a unit test wearing an integration costume is slower and proves less.
5. **Mutation** — on the package a change touched, before the pull request: `make mutate PKG=./internal/ui/settings/...`. It changes the code under the tests and asks whether they notice. A mutant that survives is a statement no test disagrees with, which is a test watching without looking. Kill it or write down why it does not matter.
6. **Adversarial** — the case written by somebody trying to break it, not to confirm it. The empty input, the one that arrives twice, the answer that comes back after the reader left, the file deleted between the listing and the read, the value that is legal and absurd. Most of the bugs this project has actually shipped were one of those, and each one is now a test with a comment saying what happened.

Golden files for the window are regenerated with `go test ./internal/ui -update` when the drawing changed on purpose — and the pull request says so.

---

## Concurrency

Never start a goroutine without knowing how it ends. If it outlives the call that started it, the comment above it says who stops it and when — `internal/task.Start` waits on its child in a goroutine and explains, in place, why the alternative leaves a zombie that answers `kill(pid, 0)`.

Leave concurrency to the caller where you can: a function that returns a value is usable by a caller who wants it in a goroutine; a function that starts one is not.

---

## Before you open a pull request

```bash
make check   # gofmt, vet (twice, once for linux), golangci-lint, tests, go.mod tidy
```

`make check` is the definition of done — read its exit status, not the output of something you piped it into. If the window's drawing changed on purpose, regenerate the golden files with `go test ./internal/ui -update` and say so in the pull request.

Then:

1. Branch from `main` with a name that says what the batch is: `ui/split-1`, `fix/task-stream-parsing`.
2. Write the commit message as prose: what changed, and why it is the right change. The subject line is one line in the imperative.
3. Open the pull request against `main` with what it does, what it fixes, and how a reviewer can check it.

---

## Development setup

- **Go 1.26+**, **git**, and `golangci-lint` (`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`).

```bash
git clone https://github.com/e1i0r/orbit.git
cd orbit
make build
./orbit top .
```

Install it where your shell will find it with `make install PREFIX=$HOME/.local`.

### The landing page

`site/index.html` and `site/es/index.html` are **generated**. Editing them
directly is lost work: they are written by `make site` from one template and
two catalogues of sentences.

| | |
| --- | --- |
| `web/page.tmpl.html` | the markup, once |
| `web/en.json`, `web/es.json` | every sentence, in both languages |

Change one of those, run `make site`, and commit what it wrote. `make check`
fails when the committed pages are not what the template says, and it fails
when a sentence exists in one language and not the other.

---

## Code of conduct

This project is released with a [Contributor Code of Conduct](CODE_OF_CONDUCT.md). By participating you agree to abide by its terms.
