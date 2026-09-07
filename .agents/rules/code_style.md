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

## Tests

- Table-driven, cases named. No assertion libraries.
- Name the behaviour: `TestADeletedIdIsFreeAgain`, not `TestDelete`.
- The failure message says what was expected and what happened, in the words of the thing under test.
- 90% coverage or better, and a package carries its own suite.
