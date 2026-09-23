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
| `l` | `map` | the repository as a tree, lit where the task changed it; click a file to open its diff |
| `i` | `impact` | what the change reaches that the diff cannot show |
| `w` | `thinking` | the reasoning the engine showed its work in |

`tab` and `shift+tab` walk them. Inside a pane, `e` opens every row at once and
`v` shows what was written down rather than what was made of it.

### Buttons on the flow tree

Open a node on `flow` and the last thing under it is a button. The icon says
what the press does, and the wording follows it.

| Button | On a node that |
| --- | --- |
| `▶ run it` | never ran |
| `↻ run it again` | finished |
| `↻ try it again` | broke, or was stopped |
| `■ stop it` | is running now |

The verbs you asked for by hand hang off the foot of the same tree and carry
the same button. One still out offers `■ stop it`, which ends the wait: the
cockpit stops holding the verb open, and whatever the supervisor says later
still lands in its thread. One that came back offers `↻ ask for it again`.

While a verb the supervisor carries is out, its node lists the last five
things the supervisor did for it, the newest turning: `Bash: git push`,
then `Bash: gh pr create`. The bar says the one in hand, and the timeline
keeps every one of them as a `step`.

A verb is carried by the window it was asked in. If that window closes
before the answer comes back, nothing is working on it any more, and the
tree says it broke: on the next tick in a window that is open, or as soon
as the next one opens. It never sits on `in progress` with nothing behind it.

Whatever is happening right now, a phase or a verb, turns with the spinner
and says `in progress`, in the theme's live colour. A pull request verb that
came back carries the pull request's link on its line.

There is no letter on them. They sit a press away from the delivery keys,
and a cheap gesture next to an expensive one is a trap. On this tab the
arrows walk the tree instead: `↑` and `↓` move a cursor from node to node
and onto an open node's button, going round from the last to the first,
and `↵` opens or closes the node it is on or presses the button. The bar
says which of the two `↵` will do.

## On the board

| Key | What |
| :--- | :--- |
| `↑` `↓` / `k` `j` | move through the queue |
| `g` `G` | first, last |
| `⏎` | open the task, or fold the band under the cursor |
| `m` | the task's menu: everything that can be done to this row |
| `M` | the board's menu: the commands that are about no task |
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

Capitals open a screen. Small letters do something to the task under the
cursor. That is the whole of how the two halves are told apart.

| Key | Screen |
| :--- | :--- |
| `R` | repositories |
| `F` | flows, and the designer |
| `E` | engine and model knobs |
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

The task's own verbs work here too, on the task you are reading and not on
whichever row the board was left on: `r` `s` `x` `b` `h` `d` resume, skip,
cancel, put back in To Do, hand back and mark read. `p`, `t` and `D` mean
something else on this screen, so pause, take the keyboard and delete are
reached through `m`, whose entries act on this task the same way.

## Settings

`:` then `settings` opens them. Arrows walk the dials, `e` types a value that
is on no dial, and `x` puts one back to what Orbit ships.

They are listed in groups, here and in `orbit settings`: **Queue** (how many
runs go at once, memory, autopilot, the unread cap), **Tasks** (engine, model,
effort, thinking, flow, how long a run may take), **Spending**, **Decisions**
(the decision engine and how sure it has to be), **Notifications**,
**Appearance** and **Maintenance**. The ones that only mean something together
sit together.

Every setting Orbit has is a row here. That is a claim the build keeps rather
than a promise somebody remembers: the table is read off the same declaration
`orbit settings` prints from, and a test fails if a setting is declared that
the window draws no row for. It used to be a list written out by hand in the
window's own code, and six settings were added to Orbit without ever reaching
it, among them whether Orbit may interrupt you, and which account may
command it over a chat.

Four of the rows offer nothing to choose from: a chat id, the two budgets
and the quota floor, where no list of values is the list anybody wants. Those
show what they hold, and `e` or a click opens the line to type into. What
each will accept is checked in one place for every way in, so a number the
terminal refuses is a number the window refuses.

There are more dials than a terminal has rows, so the table scrolls: the
arrows and the wheel bring whatever the cursor is on into view, one setting a
notch, and the title and the line of keys stay where they are while it moves.
A click is measured from where the table now starts. Before that it was
measured from the top of the screen, which on a scrolled table turned the
dial of whichever row used to be drawn there.

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
