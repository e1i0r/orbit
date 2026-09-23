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
orbit supervisor say "the migration has to be reversible"
orbit supervisor say "stop opening PRs against main"
```

It is still there next session, and next month. That is the difference between
telling an agent something and telling Orbit something: the first is gone when
the window closes.

| | |
| --- | --- |
| `S` | open the thread |
| type, `⏎` | say something to it |
| `orbit supervisor` | read it from a script |
| `orbit supervisor retract -line <n>` | take a line back |

## How it learns

Down the side of the thread is the column of what Orbit knows. A conclusion
the supervisor reaches, whether this repository owns no migrations, this
check is flaky or that directory is money, goes in as a fact with its scope and its
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

## The decision engine

A run that stops at a gate waits for you. With a decision engine the
supervisor can read what it did and answer in about half a second, and
gates it is sure about stop waiting.

It needs two things and does nothing with one of them: `TYPESAFE_API_KEY`
in the environment, and the setting switched on. See [the
environment](env.md).

```bash
orbit settings set decisions shadow   # it writes down what it would decide
orbit settings set decisions on       # it acts on what it is sure about
orbit settings set decision-floor 80  # how sure that has to be, 50 to 99
```

Start on shadow. It asks at every gate and writes the verdict into the
record, and acts on nothing, so a few tasks tell you whether it agrees
with you before it decides anything.

What it does on `on` depends on the switch beside it. With autopilot off
it can let a gate go that it is sure the work has cleared. With
[autopilot](autopilot.md) on it does the opposite: autopilot lifts every
gate, and the engine holds the runs it is sure need a person. One of them
interrupts you less and the other interrupts you more, and only the first
can be wrong in a way that costs you something.

Anything it is not sure enough about, anything it cannot reach, and every
answer that is not a plain verdict leave the run exactly as it was.

What it reads at a gate is the task and the last report a phase wrote. A
phase's report stands until that phase runs again, so a task you move
back to a gate is judged on the work it already has. Before, every new
attempt started with nothing to read, and a task retried at review was
answered "again" over an implement that had reported everything green.

---

Next: [what Orbit knows](knowledge.md) · [autopilot](autopilot.md)
