# Many tasks at once

Three runs going, in three worktrees, in one window.

<img src="../assets/flow-parallel.gif" alt="several tasks running side by side in the running band, each in its own worktree, moving through their phases" width="900">

## One worktree each

Every task runs in a git worktree of its own, under the state root, on a
branch named after the task. Nothing they do touches your checkout, and
nothing they do touches each other's.

That is what makes the parallelism safe rather than exciting: two agents
editing the same file in the same directory is a race, and two agents editing
their own checkouts is just two branches.

```bash
orbit run -repo ~/code/api fix-auth &
orbit run -repo ~/code/api add-index &
```

Or turn on [autopilot](autopilot.md) and let it take them off the queue.

## Watching them

The board is the answer: the **Running** band carries one row per live run,
each with the phase it is in, what it is doing right now, how long it has been
going and what it has spent. `Enter` opens any of them without stopping the
others.

The window redraws as they move. A task that fails a gate drops into **Needs
You** while the others carry on.

## Several repositories

`orbit top ~/code` is one window over every repository under that directory,
so the queue is not per project. A task can also reach a second checkout:
`orbit join` opens another repository for a task that turns out to need one —
a change to the API and its client, in one task, with both worktrees under it.

## What limits it

Not Orbit. The engine's own rate limits are the ceiling, and `Q` shows what is
left of each one's window before you start something that will hit it. The
unread cap is the other limit, and it is deliberate: see
[autopilot](autopilot.md).

---

Next: [reading what it did](reading.md)
