# Autopilot

`A` on the board. Orbit takes the next To Do and works the queue.

<img src="../assets/flow-autopilot.gif" alt="autopilot turned on, the queue moving on its own, and a task that needed a person landing in needs you" width="900">

## The two modes

**On.** Orbit runs the next To Do through its flow without stopping at the
flow's gates, and lets [the supervisor](supervisor.md) resolve what comes
back needing attention.

**Off.** It runs what you start and hands the rest back to you.

## What it does not lift

Autopilot lifts the flow's gates. Four things it leaves alone:

- a pause you set by hand, because that decision is yours
- a run the decision engine held because it is sure a person is needed
  (`decisions on`); it waits for your resume, skip or cancel
- a task marked critical, which waits for `orbit permit` before anything
  irreversible
- the unread cap

## The brake

`unread-cap` is 5. Once five finished tasks sit unread, nothing new starts.

The queue cannot outrun you. An agent that works all night is worth having
only if the morning's reading is still possible, and ten unread runs is more
than anyone gets through in one sitting.

```bash
orbit settings set unread-cap 4      # a shorter leash
orbit settings set autopilot on      # the same switch, from a script
orbit settings clear unread-cap      # back to the one Orbit ships with
```

`d` marks a task read and gives you the slot back.

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

Every gate autopilot lifts goes into the record as a decision nobody was
asked about. Six months later the timeline says which phases a person let
through and which the switch did. Those are different facts about how much
the result was checked.

---

Next: [many tasks at once](parallel.md) · [the supervisor](supervisor.md)
