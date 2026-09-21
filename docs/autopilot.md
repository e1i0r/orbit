# Autopilot

`A` on the board. Orbit picks up the next To Do and works the queue.

<img src="../assets/flow-autopilot.gif" alt="autopilot turned on, the queue moving on its own, and a task that needed a person landing in needs you" width="900">

## The two modes

**On** — Orbit takes the next To Do, runs it through its flow without stopping
at the flow's own gates, and lets [the supervisor](supervisor.md) try to
resolve whatever comes back needing attention.

**Off** — it runs what you start, and hands everything else back to you.

## What it does not lift

Autopilot lifts the flow's gates. It does **not** lift:

- a pause you set by hand — that is your decision, and a switch does not
  overrule it;
- a task marked critical, which stops before anything irreversible and waits
  for `orbit permit`;
- the unread cap.

## The brake

`unread-cap` is 5 by default. Once that many finished tasks are sitting
unread, nothing new starts.

That is the whole design: the queue cannot outrun you. An agent that can work
all night is only useful if what it produced is still readable in the morning,
and ten unread runs is already more than anyone reads in one sitting.

```bash
orbit settings set unread-cap 4      # a smaller leash
orbit settings set autopilot on      # the same switch, from a script
orbit settings clear unread-cap      # back to the leash Orbit ships with
```

`d` marks a task read and gives you one slot back.

## The other brake: how long a run may take

`run-timeout` is how long a run the window or the queue starts may take
before it is stopped. It is empty by default — no limit — because there is
no honest default: a phase that reads a repository is seconds and one that
writes a migration is an hour.

What it is for is the run nobody is sitting in front of. An engine wedged on
a network read holds its worktree and its slot until somebody notices, and
noticing is the part a person asleep cannot do.

```bash
orbit settings set run-timeout 2h    # nothing the queue starts runs longer
orbit settings clear run-timeout     # no limit again
```

A run you start by hand takes the same limit as a flag: `orbit run -timeout
45m`. The setting is what that flag becomes when the window starts the run.

## What it is recorded as

Every gate autopilot lifts is written into the record as a decision nobody was
asked about. Six months later the timeline says which phases a person let
through and which the switch did — and that is a different fact about how much
the result was checked.

---

Next: [many tasks at once](parallel.md) · [the supervisor](supervisor.md)
