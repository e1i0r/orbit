# The recordings

One tape per flow, and the flow's page and its section on the landing carry
what it shot. They are here rather than in somebody's shell history because a
recording that cannot be shot again is a recording that goes stale silently:
the cockpit moves, and the only way to know a video is lying is to run its
tape.

## Shooting them

```bash
make demo     # seed a board to shoot against, under a state root of its own
make tapes    # run every tape and write assets/ and site/
```

`make demo` builds two repositories under `~/code` and a state root under
`.demo/`, and writes the record with `orbit new`, the migration that runs
before every command, and `orbit supervisor -by`. It calls no engine and it
never touches `~/.orbit`.

Three of the nine cannot be seeded, because what they show is movement: a run
going through its phases, three of them at once, and the terminal handed to a
CLI and taken back. Those are shot against real runs — see `real.md`.

## The rig

`Set FontSize 22 / Width 1880 / Height 700 / Padding 20`, which is 142×24 —
the geometry the shipped assets already use, and wide enough that the board's
own columns are not the thing being tested.

Two things learned the hard way and worth keeping:

- A `Down 6` repeat count does not register. Emit separate `Down` lines with
  a `Sleep` between them.
- The board state decides where the arrows land. Re-seed before every take, or
  a task a previous take created puts every keystroke one row out.
