# Autopilot

`A` on the board. Orbit takes the next To Do and works the queue.

<img src="../assets/flow-autopilot.gif" alt="autopilot turned on, the queue moving on its own, and a task that needed a person landing in needs you" width="900">

## The two modes

**On.** Orbit runs the next To Do through its flow without stopping at the
flow's gates, and lets [the supervisor](supervisor.md) resolve what comes
back needing attention.

**Off.** It runs what you start and hands the rest back to you.

## What it does not lift

Autopilot lifts the flow's gates. Three things it leaves alone:

- a pause you set by hand, because that decision is yours
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

## What it is recorded as

Every gate autopilot lifts goes into the record as a decision nobody was
asked about. Six months later the timeline says which phases a person let
through and which the switch did. Those are different facts about how much
the result was checked.

---

Next: [many tasks at once](parallel.md) · [the supervisor](supervisor.md)
