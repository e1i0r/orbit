# Orbit: how to work in this repository

`CONTRIBUTING.md` is the source of truth for how this code is written, and it is short. Read it. What follows is the same rules in the form an agent needs them: what to check before writing, and what will fail if you do not.

To read further: [Practical Go](https://dave.cheney.net/practical-go).

---

## Before you write

1. **Find the door.** Every package is entered through the files that export its names — a screen through `Open`, `Key`, `Apply`, `View`; a reader through `Read`. `internal/arch/doors_test.go` lists them per package, and that list is the index: read it before opening files, and it will tell you where the thing you need lives.
2. **Read the layers.** `internal/arch/layers_test.go` says which packages may import which, with the argument for each. If what you are about to write needs a new import between packages, that is a decision to argue in the pull request, not a line to add quietly.
3. **Read the neighbours.** Match the file you are in: its comment density, its naming, its shape. A file that reads differently from the ones beside it is a file a reviewer has to learn twice.

## What will fail if you get it wrong

`make check` runs all of it. Read its exit status, never the output of something you piped it into.

| Rule | What holds it |
|---|---|
| 300 lines per file, code and comment | `TestNoFileOverTheCeiling` |
| 100 columns per line of code, strings exempt | `TestCodeStaysInsideTheColumn` (a ratchet: it may go down, never up) |
| Only doors export | `TestEveryExportedNameIsBehindADoor` |
| Interfaces of six methods or fewer | `TestInterfacesStaySmall` |
| No `util`/`common`/`helpers` packages | `TestNoPackageIsADrawer` |
| No `GetThing()` getters | `TestNoGetterSaysGet` |
| Imports follow the layer map | `TestImportsFollowTheLayers` |
| Every translation key is used and honest | `TestEveryTranslationKeyIsHonest` |
| The window measures cells, not bytes | `TestUIMeasuresCellsNotBytes` |
| Colours are named in `internal/ui/theme` | `TestColoursLiveInTheTheme` |
| Coverage at or above 90%, or the build fails | `make coverage` |

## What a change brings with it

Tests are not an afterthought and not one kind. Bring the ones that fit the change:

1. **Unit** — always, in the package that owns the behaviour, named after it.
2. **Property-based** — where there is an invariant: a lossless round-trip, a cost
   that never decreases, a layout that never goes negative. Assert the law, not one
   example of it.
3. **Fuzzing** — where bytes arrive from outside: parsers, stream decoders, fitters.
   `make fuzz PKG=./internal/engine/... FOR=2m`, and commit the corpus it finds.
4. **Integration** — where the seam is the subject: a whole flow, a command that
   opens the store and writes the record.
5. **Mutation** — on the package you touched, before the pull request:
   `make mutate PKG=./internal/ui/settings/...`. A surviving mutant is a statement no
   test disagrees with. Kill it, or say why it does not matter.
6. **Adversarial** — the case written to break it: empty, twice, out of order, after
   the reader left, deleted between the listing and the read. Most bugs this project
   shipped were one of those.

## The five that are judgement, not tests

1. **Nothing is hidden.** No error is swallowed or softened. `_ = f()` needs a `//nolint:errcheck // reason` that says why in words.
2. **Handle an error once.** Wrap it with `%w` and the sentence saying what was being attempted, and return it. Log it only where it stops travelling — the command that gives up, the window that puts it in the band — with its subsystem tag: `logger.Error("cli/run", …)`. Never both.
3. **Say it once, plainly.** Every sentence the operator sees goes through `p.T("key", "the English")`, is short, and says what happened and what to do. No adjectives, no apologies.
4. **Comments explain why.** State the fact about the code, not a verdict on it. Where a decision was made — an alternative rejected, a constant that came from measurement, a guard that exists because something happened — say so, because the code cannot.
5. **Interfaces are declared by whoever needs them**, with the two or three methods that caller uses. Not by whoever implements them.

## Git

Never `git push`, never open a pull request, and never merge unless you were asked to in words. Work stays local and committed only when the operator says so — they review before anything leaves the machine.
