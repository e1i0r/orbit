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
vhs assets/tapes/flow-run.tape
vhs assets/tapes/flow-parallel.tape
vhs assets/tapes/flow-cli.tape
make posters
```

Re-run `make demo` between takes. The board state decides where the arrow
keys land, and a task a previous take left in Done puts every keystroke one
row out.

## What cannot be scripted

The CLI beat. `c` hands the terminal to the engine's own program, and that
program asks a person before it touches anything — which is the point of the
key and is not something a tape can answer. Record generously, then cut the
wait out in post:

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
