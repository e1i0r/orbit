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
orbit task start -repo ~/code/api fix-auth &
orbit task start -repo ~/code/api add-index &
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
`orbit join` opens another repository for a task that turns out to need one,
a change to the API and its client, in one task, with both worktrees under it.

## What limits it

**The queue.** Three runs go at once; start a fourth and it waits, marked
`queued · 1st in line`, and starts on its own when one finishes. Starting
five tasks no longer means five engines and five `make check`s fighting
for one machine.

```bash
orbit settings set max-running 5      # how many go at once
orbit settings set memory-ceiling 80  # no new run above 80% memory
```

A run also waits while the machine's memory is over the ceiling (85% unless
you set it). `x` takes a waiting task out of the queue, `b` puts it back in
To Do; on the command line, `orbit task cancel` and `orbit task requeue`.

The queue keeps moving with the window closed. When something is waiting,
Orbit starts a small service of its own, `orbit __queue`, that starts the
next run as a slot frees up and goes away when nothing is left. There is
nothing to install.

**Priority.** Everything a run starts, and every CLI session the window
opens, runs at the lowest priority your system has. On an idle machine
that changes nothing; when you are using it, a lint that wants every core
gives way, and the window keeps answering.

The engines' own rate limits are the other ceiling, and `Q` shows what is
left of each one's window before you start something that will hit it. The
unread cap is the last limit, and it is deliberate: see
[autopilot](autopilot.md).

---

Next: [reading what it did](reading.md)
