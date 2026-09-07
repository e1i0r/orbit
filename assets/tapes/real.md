# The three that need a run going

`make tapes` shoots five of the nine against a seeded board. Three of them
cannot be seeded, because what they show is movement: a run walking its
phases, three of them at once, and the terminal handed to a CLI and taken
back. A fabricated run marker does not move, and Orbit reconciles it away the
moment it notices the process is not there.

So these are shot against real runs, and they cost what a real run costs.

## Before

```bash
make demo                       # the board, the repositories, the state root
export ORBIT_DEMO_HOME=$PWD/.demo/home
export ORBIT_DEMO_BIN=$PWD
export ORBIT_HOME=$ORBIT_DEMO_HOME
orbit settings engine claude    # or whichever is installed and logged in
orbit settings model sonnet     # the cheap one: these are demonstrations
```

The tasks the tapes start are the two in **To Do** and the one in **Needs
You**, and they are small on purpose. The video is about the phases, the
board and the handover — not about the change.

## Shooting them

```bash
# A run, in two takes: the start, then the gate it stops at. A phase takes
# minutes and the section is forty seconds, so what is shot is the two ends of
# it — with the real minutes in between, off camera.
vhs assets/tapes/flow-run-a.tape
orbit run -repo ~/code/ledger LED-11        # wait for it to reach the gate
vhs assets/tapes/flow-run-b.tape
printf "file flow-run-a.mp4\nfile flow-run-b.mp4\n" > site/parts.txt
ffmpeg -f concat -safe 0 -i site/parts.txt -c copy site/flow-run.mp4
ffmpeg -i site/flow-run.mp4 -c:v libvpx-vp9 -b:v 0 -crf 34 site/flow-run.webm
ffmpeg -i site/flow-run.mp4 -vf "fps=12,scale=1200:-1:flags=lanczos" assets/flow-run.gif

vhs assets/tapes/flow-parallel.tape
vhs assets/tapes/flow-cli.tape
make posters
```

Re-run `make demo` between takes. The board state decides where the arrow
keys land, and a task a previous take left in Done puts every keystroke one
row out.

## What cannot be scripted

The CLI beat, and it is worth saying exactly where it breaks.

`c` does hand the terminal over, and the engine's own program opens in the
task's worktree. What it does first is ask a person whether this directory is
trusted, and that question is the point of the key rather than something in
the way of it. The tape answers it — `Down`, `Enter` — because a recording
that stops there records nothing; on a machine where the directory is already
trusted those two keystrokes go to the prompt line instead, and the take is
shot again.

Two things about the take that has to be cut afterwards:

- The engine prints its own startup warnings, and they name paths from
  whatever settings file it found. Those are somebody's private directories.
  **Watch the first frames of every take and cut them off.**
- A model that takes a minute to answer takes a minute. Shoot generously and
  keep the part where it reads the record and writes back to it, which is what
  the section is about.

```bash
ffmpeg -ss 38 -to 96 -i take.mp4 -c:v libx264 -crf 22 -an site/flow-cli.mp4
```

And to cut a wait out of the middle instead:

```bash
ffmpeg -i take.mp4 -ss 0    -to 12 -c copy part-a.mp4
ffmpeg -i take.mp4 -ss 40          -c copy part-b.mp4
printf "file part-a.mp4\nfile part-b.mp4\n" > parts.txt
ffmpeg -f concat -i parts.txt -c copy site/flow-cli.mp4
```

The `Sleep` lines in that tape are generous for the same reason: a model that
takes forty seconds to answer is a model, not a bug, and a tape that gave it
ten would record the question and not the answer.

## What the takes are checked against

The claim each one makes is in its page and in its section on the landing.
A recording that no longer shows what the sentence beside it says is a
recording to shoot again — that is the whole reason the tapes are in the
repository rather than in somebody's shell history.
