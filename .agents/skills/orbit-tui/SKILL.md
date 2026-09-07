---
name: orbit-tui
description: >-
  How the cockpit is put together: the packages under internal/ui and what each is
  entered by, the twelve inspector tabs, pure layout geometry, mouse routing, and the
  rule that the window measures cells and never bytes.
---

# The cockpit

Bubble Tea v2 and Lip Gloss. `internal/ui` is the window — the `Model`, the screens
that still live in it, and the ports it is handed. Everything under it is a package
of its own, entered through its doors.

## Where things live

| Package | What it is | Entered by |
|---|---|---|
| `internal/ui/theme` | every colour: roles, palettes, the paper each surface is drawn on, the code lexer | `Paint`, `Text`, `Surface`, `Pill`, `LexCode` |
| `internal/ui/cells` | measuring and cutting in cells | `Fit`, `Fill`, `Lines`, `PadRight` |
| `internal/ui/keymap` | which key does what, and which verbs a task offers with the reason each refusal gives | `New`, `Affordances`, `Why` |
| `internal/ui/typing` | the field somebody types into: value, caret, selection, wrapping, painting | `New`, `Insert`, `Wrap`, `PaintCells` |
| `internal/ui/patch` | a diff read: which files, how much, what the record says each change was for | `Files`, `Stats`, `Rationales` |
| `internal/ui/settings` | the settings screen, whole | `Open`, `Rows`, `Key`, `Apply`, `View` |
| `internal/ui/prompt` | what the window asks an engine for | `Deliver`, `Phase`, `FlowDraft` |
| `internal/ui/layout` | pure geometry: frames, columns, bounds. No Bubble Tea, no Lip Gloss | `Frame`, `Columns` |
| `internal/ui/prose` | how a block of text is set, and the shapes a screen is built out of | `Section`, `Meta`, `Quote`, `Strip`, `Card`, `Fields`, `Badge`, `Chip` |
| `internal/ui/panes` | the twelve panes of the task screen: one door each, drawn from an Env | `Overview`, `Pipeline`, `Gates`, `Cost`, `Refused`, `Timeline`, `Report`, `Artifacts`, `Notes`, `Diff`, `Impact`, `Thinking` |
| `internal/ui/menu` | what can be done to the thing under the pointer, including what cannot | `Open`, `Key`, `Choose`, `Enter`, `View`, `Hit` |
| `internal/ui/clip`, `spoken`, `upgrade` | the pasteboard, the operator's gestures, the release check | one door each |

`internal/arch/doors_test.go` is the full list and it is the index of the window:
read it before opening files.

## The rules that are particular to the window

1. **Cells, never bytes.** No `len()` on a string and no `s[a:b]` slicing anywhere in
   `internal/ui` — a wide rune is two columns and a combining mark is none.
   `TestUIMeasuresCellsNotBytes` fails on it. Use `internal/ui/cells`.
2. **Geometry is pure.** Frame and column arithmetic lives in `internal/ui/layout`,
   which imports no Bubble Tea and no Lip Gloss, so the numbers can never become a
   function of anything but the numbers they were given.
3. **A screen that is a package never sees the `Model`.** It is handed an `Env` with
   the little world it needs and answers with an `Out` — a sentence for the band,
   whether to close, a `tea.Cmd`. The window does what it asked for in
   `internal/ui/screens.go`, which translates and never decides.
4. **Every gesture reports.** A verb says what it started when it starts and what
   happened when it lands, in the band, through `p.T(...)`. A long wait says what is
   being waited for. Nothing happens silently.
5. **Mouse and keyboard both reach everything.** Targets are registered as
   `Target{Kind, ID, Pane, Field}` and routed in `mouseroute.go`; a screen that can
   only be driven by one of the two is unfinished.

## The task view

Twelve tabs: overview (1), flow (2), gates (3), cost (4), refused (5), timeline (6),
report (7), artifacts (8), notes (9), diff (0), impact (i), thinking (w). Each is a
door of `internal/ui/panes`, handed an `Env` — the run, its record, the room it has
and what the reader has folded — and nothing else: a pane can reach no port and
decide nothing about the task it is drawing. Adding one means `tabNames()` **and** the
descriptions in `paneMenu()` — a tab with no line in the menu is the one row a reader
opened the menu to understand.

## Themes

`frauddi` is the default. The palette is process-wide state, which is the one global
in the drawing path and is written down as such in `CONTRIBUTING.md`.
