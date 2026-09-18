# Orbit code style

The rules and the reasons are in `CONTRIBUTING.md`; this is the checklist.

## Files

- **Under 300 lines**, code and comment, blanks excluded. A file over it is two subjects that have not been told apart — split along the seam, never at line 300.
- **Named for what a reader looks for**: the door it opens (`key.go`, `apply.go`, `view.go`) or the subject it holds (`tasks.go`, `repos.go`). A satellite carries its door's name: `overview.go` and `overview_blocks.go`, never `blocks.go`.
- Tests sit beside their source. Fuzz targets are `Fuzz*`.

## Lines

- **100 columns of code.** A string a person reads — a translated sentence, a prompt, a URL — is exempt: breaking one to fit a column makes it worse.
- One blank line between the steps of a function: what it was given, what it checked, what it did, what it answers. No walls of statements.
- Wrap comment prose at about 75 columns, which is where the repository already sits.

## Names

- Packages: one lowercase word, for what they provide. No `util`, `common`, `helpers`.
- No stutter: `cells.Fit`, never `cells.CellsFit`.
- No `Get`: `Language()`, not `GetLanguage()`.
- Length follows scope: `i` in a loop, a sentence's worth at package level. One receiver name per type, everywhere.
- Don't name a variable after its type: `tasks`, not `taskSlice`.

## Signatures

- Never several parameters of the same type in a row — two strings side by side is a swap that compiles. Give them a struct with named fields.
- Two return values, or a struct. Three anonymous ones say nothing about which is which.
- Accept interfaces, return structs. Declare the interface where it is used, with the methods that caller needs, six at most.

## Errors

- Wrap with `%w` and say what was being attempted: `fmt.Errorf("load task %q in %q: %w", id, repo, err)`.
- Handle once: log where it stops, propagate everywhere else. Never both.
- Never discard a return. `_ = f()` carries `//nolint:errcheck // reason`, in words.

## Comments

- Every exported name has a comment starting with that name.
- Explain why. State the fact, not a verdict — `// the flag package stops at the first non-flag argument`, not `// this was a nasty bug`.
- A decision that was weighed and rejected is worth a line: it is the question the next reader will ask.
- **Nothing about the work, only about the code.** A comment describes what is there and why it is not the other thing. It never describes the session that produced it: no note that a mutation run left this statement alive, that a lint rule was waived here, that coverage does not reach this line, that a review asked for this, or that this is the second attempt. Those are true of one afternoon — the mutant gets killed, the rule changes, the line gets covered — and the sentence stays describing a problem that is gone. Where it goes is the pull request.
- **Real names, of the thing.** A file, a test, a variable or a fixture is named for what it is in this project's own words: `supervisor_mouse_test.go`, `twoConversations`, `saidThree`. Never a placeholder — no `zz_scratch_test.go`, no `foo`, no `tmp2`, no `x1` — and never a prefix whose only job is to sort the file last. A name nobody can read is a file nobody opens, and the one thing worse than no test is a test named so that nobody ever looks at what it does not assert.
- **No task ids.** Never `FRA-61`, `ORB-115` or any other name from a tracker, in a comment or anywhere else in the source. Somebody reading the code has no access to it, the issue gets closed, renumbered or moved, and what is left is a reference to nothing. Say the thing itself: "a task written against no repository", not "which FRA-61 made possible". Where a made-up id is needed as an example, the repository already uses `ACME-1` and `PAY-1`.

## Tests

- Table-driven, cases named. No assertion libraries.
- Name the behaviour: `TestADeletedIdIsFreeAgain`, not `TestDelete`.
- The failure message says what was expected and what happened, in the words of the thing under test.
- 90% coverage or better, and a package carries its own suite.
