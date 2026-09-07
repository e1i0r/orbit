# The supervisor

A second pair of eyes on the work, and a thread you steer it in. `S` in the
cockpit.

<img src="../assets/flow-supervisor.gif" alt="the supervisor thread: a directive typed into it, the supervisor reading a finished run and going to fix what was missing, and a fact it learned appearing in the column beside it" width="900">

## What it is

A run like any other. It goes out through the same CLI and the same
subscription as your tasks, on whichever engine the dial is set to. There is
no separate model and no separate bill.

What it does is the job you would do yourself: read the result of a run,
decide whether it actually did what the task asked, and go fix what is
missing. With [autopilot](autopilot.md) on, it is what tries to resolve a task
that came back needing attention before you are asked to look at it.

## The thread

`S` opens a conversation that persists. Write into it and everything the
supervisor does from then on carries what you said:

```bash
orbit supervisor "the migration has to be reversible"
orbit supervisor "stop opening PRs against main"
```

It is still there next session, and next month. That is the difference between
telling an agent something and telling Orbit something: the first is gone when
the window closes.

| | |
| --- | --- |
| `S` | open the thread |
| type, `⏎` | say something to it |
| `orbit supervisor` | read it from a script |
| `orbit supervisor -retract <n>` | take a line back |

## How it learns

Down the side of the thread is the column of what Orbit knows. A conclusion
the supervisor reaches — this repository owns no migrations, this check is
flaky, that directory is money — goes in as a fact with its scope and its
source, and reaches the next run's prompt before it works.

Which means the conversation is not chat. What you say in it becomes something
the runs carry. Correcting the supervisor once is different from correcting an
agent every session, and the column beside the thread is where you see the
difference land. See [what Orbit knows](knowledge.md).

## Directing one task

The thread is standing instruction. For one run there is `a` on the task, and:

```bash
orbit direct -repo ~/code/api -restart fix-auth "use the existing retry helper"
```

which interrupts what is going and records the directive against that task.

---

Next: [what Orbit knows](knowledge.md) · [autopilot](autopilot.md)
