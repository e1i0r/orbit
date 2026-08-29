# The cockpit

`orbit top [dir]` opens one window over every repository under a directory.

## Bands

Tasks sit in four bands and move between them on their own as runs progress:
**To Do**, **Needs You**, **Running**, **Done**.

A run that fails a gate, or one you paused by hand, lands in **Needs You** and
stays there until you look at it. `unread-cap` (10 by default) stops anything
new from starting once that many finished tasks are sitting unread.

## Tabs

Open a task and eleven tabs cover it.

| # | Tab | What it holds |
| --- | --- | --- |
| `1` | `overview` | the task, its flow, where the run is |
| `2` | `flow` | the phases and what each one was given |
| `3` | `gates` | every gate and how it answered |
| `4` | `cost` | what the run spent |
| `5` | `refused` | tool calls the permissions denied |
| `6` | `timeline` | the run as it happens, event by event |
| `7` | `report` | what the engine reported at the end |
| `8` | `artifacts` | files the run produced |
| `9` | `notes` | notes you and the supervisor left |
| `0` | `diff` | the worktree diff |
| `w` | `thinking` | the engine's reasoning blocks |

## Keys

| Key | Where | What |
| :--- | :--- | :--- |
| `↑` `↓` / `k` `j` | board | move through the queue |
| `Enter` | board | open the task |
| `Esc` / `q` | anywhere | back |
| `1`-`9`, `0`, `w` | task | jump to a tab |
| `[` `]` | task | previous / next tab |
| `A` | board | autopilot on and off |
| `n` | board | new task |
| `+` | board / compose | flow designer |
| `F` | task | inspect the flow |
| `k` / `M` | board / task | engine and model dials |
| `E` | task | cycle reasoning effort |
| `t` | task | thinking mode |
| `p` / `u` | task | pause / resume |
| `x` | task | cancel |
| `a` | task | write a note |
| `R` | board | repositories |
| `S` | board | settings |
| `:` | anywhere | command palette |
| `?` | anywhere | help |

## Behind every key

Everything the cockpit does has a command behind it: `orbit new`, `run`,
`pause`, `resume`, `list`, `show`, `read`, `pr`, `merge`, `close-pr`, `cancel`,
`reconcile`, `note`, `direct`, `supervisor`, `settings`. Run `orbit help` for
the full list.
