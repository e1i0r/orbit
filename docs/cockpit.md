# The cockpit

`orbit top [dir]` opens one window over every repository under a directory.

Nothing here has to be memorised. Press `m` on whatever the cursor is on and
the menu lists every verb that applies to it, including the ones it will not
let you press and the reason each is refused. `?` is the whole sheet.

## Bands

Tasks sit in four bands and move between them on their own as runs progress:
**To Do**, **Needs You**, **Running**, **Done**.

A run that fails a gate, or one you paused by hand, lands in **Needs You** and
stays there until you look at it. `unread-cap` (10 by default) stops anything
new from starting once that many finished tasks are sitting unread.

## Tabs

Open a task and twelve tabs cover it.

| # | Tab | What it holds |
| --- | --- | --- |
| `1` | `overview` | the task as it was written, the figures, where the run is |
| `2` | `flow` | the phases as a tree, and what each one was given |
| `3` | `gates` | every gate, whether it passed, and what it ran |
| `4` | `cost` | what each phase spent, and what it adds up to |
| `5` | `refused` | tool calls the sandbox denied, and the rules it denied them by |
| `6` | `timeline` | the run event by event, seamed between attempts |
| `7` | `report` | what the engine wrote about the change |
| `8` | `artifacts` | every file the run left, and what each one is |
| `9` | `notes` | notes you filed, sessions beside the run, questions it asked |
| `0` | `diff` | the worktree diff, one card per file |
| `i` | `impact` | what the change reaches that the diff cannot show |
| `w` | `thinking` | the reasoning the engine showed its work in |

`tab` and `shift+tab` walk them. Inside a pane, `e` opens every row at once and
`v` shows what was written down rather than what was made of it.

## On the board

| Key | What |
| :--- | :--- |
| `↑` `↓` / `k` `j` | move through the queue |
| `g` `G` | first, last |
| `⏎` | open the task, or fold the band under the cursor |
| `m` | the menu: everything that can be done to this row |
| `n` | start a run |
| `N` | write a task |
| `p` `r` `s` | pause, resume, skip the phase it is waiting in front of |
| `x` `b` | cancel, put it back in To Do |
| `a` | leave a note for it |
| `d` `D` | mark read, delete |
| `c` | hand the terminal to your CLI in this task's worktree |
| `t` `h` | take the keyboard from a run, hand it back |
| `A` | autopilot on and off |
| `/` `:` | filter the board, command palette |
| `?` `q` | help, quit |

## The screens

Capitals open a screen; small letters do something to the task under the
cursor. That is the whole of how the two halves are told apart.

| Key | Screen |
| :--- | :--- |
| `R` | repositories |
| `F` | flows, and the designer |
| `M` | engine and model knobs |
| `Q` | what is left of each engine's quota |
| `S` | the supervisor thread |
| `K` | what Orbit knows |
| `L` | language |

## On a task's screen

The deliver keys are the toolbar at the foot of the overview, and each is the
key printed against it there.

| Key | What |
| :--- | :--- |
| `p` `u` | create the pull request, update its branch |
| `M` `X` | merge it, close it |
| `C` `T` | fix the failing checks, ask for more tests |
| `R` `D` | bring the review comments back, ask for a deep review |
| `a` | leave a note |
| `k` `E` `t` | engine and model, reasoning effort, thinking mode |
| `F` | the flow this task runs |
| `o` | open the file under the diff in `$EDITOR` |
| `z` `Z` | fold every section (overview), collapse every file (diff) |
| `r` | on impact and diff: read it again |

## Settings

`:` then `settings` opens them. Arrows walk the dials, `e` types a value that
is on no dial, and `x` puts one back to what Orbit ships.

Putting one back is its own gesture and not typing the default in by hand,
because what a setting comes as is Orbit's to know: a person who set the
unread cap to 40 and wants the leash back should not have to remember that it
was 5. The same from a terminal, and from a browser tab where every row has a
reset:

```bash
orbit settings                       # every setting and what it is set to
orbit settings set unread-cap 40
orbit settings clear unread-cap      # unread-cap is back to 5
```

A setting that was already what it ships as says so rather than pretending
something happened.

## Behind every key

Everything the cockpit does has a command behind it: `orbit board new`, `task start`,
`pause`, `resume`, `skip`, `list`, `show`, `read`, `pr`, `merge`, `close-pr`,
`cancel`, `requeue`, `note`, `direct`, `approve`, `permit`, `critical`,
`resolve`, `supervisor`, `settings`, `export`, `digest`. Run `orbit help` for
the full list.
