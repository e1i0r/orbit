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

The same from a script: `orbit task start`, and `task pause`, `resume`, `skip`,
`cancel`, `requeue`, `note` under the same word.

## Its own worktree

Every task runs in a git worktree of its own, on a branch named after it.
That is what lets several tasks touch the same repository at once — see
[many tasks at once](parallel.md) — and it is why nothing a run does is in
your checkout until you say so.

## When it ends badly

Two of the ways are told apart on purpose, because they send you to do
opposite things.

**It broke.** The engine fell over — a bad prompt, a tool that was not there,
a crash. The row says `failed: implement` and what you do is go and look.

**It ran out.** The engine had nothing left to spend. The row says
`claude ran out: implement · back in 2h`, and what you do is wait, or hand
the same work to another engine.

Until these were apart, a task said only that something had broken and you
had to open the log to find out which. The engine is the only thing that
knows: `exec` gives a program one way to say it stopped — a non-zero exit —
and every one of these CLIs prints the provider's own refusal above it, so
each engine reads its own.

An engine that says nothing Orbit recognises is written down as broken, which
is what used to happen to all of them. Nothing gets worse by not knowing.

The window says when the allowance comes back because it already knows —
that number is in the header on every frame — and being told an engine ran
out without being told for how long is half an answer.

## When the same phase runs again

A phase that ran out, broke, or was turned back by a gate gets another run.
The work it did is still in the worktree — that is never lost — but the
account of it was, and the next engine used to open a folder of half-finished
changes with nothing saying what had already been tried.

So the second attempt is handed a short section before it starts:

```
## The attempt before you

It ran out of tokens before it could finish. What follows is read off the
record and off the worktree — what happened, not what was meant to.

### What the worktree holds now (7 files, this phase and every one before it)

- `internal/task/sofar.go` +180 −0
- `internal/task/run_helpers.go` +6 −2
…

### Commands it ran (3)

- `go build ./...`
- `make check`
…

### What it was not allowed to do (1)

- `WebFetch`
```

Every line of it is read off the record or off the disk. Nothing is written
by a model, and nothing is inferred — a summary that says something was done
when it was not is worse than no summary, because the one reading it builds
on top.

That is also why the files are credited to the worktree rather than to the
attempt. The diff cannot say which phase wrote which line, so the heading
does not pretend it can. The commands and the refusals *are* the attempt's,
because the record says when each one happened.

A first attempt is told none of this. There is no attempt before it, and an
empty heading is a question the engine would spend a turn answering.

It costs nothing: no engine is asked, no tokens are spent. The same is true
of switching engine halfway — the new one has no session to resume, and this
is what it reads instead.

---

Next: [autopilot](autopilot.md) · [reading what it did](reading.md)
