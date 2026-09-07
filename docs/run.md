# A run, end to end

A task is not one long session. It is a list of phases, each small enough to
check, with a verdict written down after every one.

<img src="../assets/flow-run.gif" alt="a task started from the board, moving phase by phase through its flow, stopping at a gate, and the gate let go" width="900">

## Why phases

One long session with an agent has a single verification point: the end. By
then the wrong assumption from minute three is buried under four hundred lines
that all look plausible, and reviewing it costs more than writing it did.

Phases cut the run into pieces. A phase that gets it wrong is caught by the
next phase or by you, before the one after it builds on the mistake. It is
also what makes the record readable later: eleven short steps with a verdict
on each, instead of one transcript nobody will open.

## What a flow is

A list of phases, each naming its own engine, model, reasoning effort,
thinking mode, prompt, and what it is allowed to touch. Five ship built in —
`quick`, `task`, `careful`, `coverage`, `tdd-fuzz-pr` — and you write your own:
[flows you write yourself](flows.md).

## Gates

A phase marked `wait` is a gate: the run stops in front of it and waits for
you. Between every pair of phases Orbit asks whether to continue, and the
answer is recorded — including "nobody was asked, autopilot was on".

A gate that fails, or a phase you paused by hand, leaves the task in **Needs
You**, where it stays until you look at it.

| Key | What |
| :--- | :--- |
| `n` | start the run |
| `p` `r` | pause it, let it go |
| `s` | skip the phase it is waiting in front of |
| `x` `b` | cancel it, put it back in To Do |
| `a` | leave a note the next phase will read |

The same from a script: `orbit run`, `pause`, `resume`, `skip`, `cancel`,
`requeue`, `note`.

## Its own worktree

Every task runs in a git worktree of its own, on a branch named after it.
That is what lets several tasks touch the same repository at once — see
[many tasks at once](parallel.md) — and it is why nothing a run does is in
your checkout until you say so.

---

Next: [autopilot](autopilot.md) · [reading what it did](reading.md)
