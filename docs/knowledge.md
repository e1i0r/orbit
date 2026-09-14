# What Orbit knows

Rules about your code, with a name, a source and a place, that arrive in the
prompt before the agent works. `K` in the cockpit.

<img src="../assets/flow-knowledge.gif" alt="the knowledge screen: facts by scope, one being corrected, one widened, one turned off, and where each came from" width="900">

## Why not the model's memory

The model forgets between sessions, and forgets when you swap it for another
one. Every CLI keeps its own notes in its own file — `CLAUDE.md`, `AGENTS.md`
— and each of those is a silo that empties the day the engine changes.

What Orbit knows is Orbit's, kept outside all of them. Change the engine and
it still knows that the ledger only appends.

## Three ways it learns

**You say it.** Tell the supervisor *"never push a pull request without the
tests passing"* and Orbit notices the sentence was a rule and offers it back.
It notices by shape — a hand-written list of openings in two languages — not
by asking a model, so it costs nothing and never surprises anybody.

**A model finds it.** An agent that hits a wall mid-task offers what it found
through `orbit_learn`. It offers; it does not write. One flow, and not two: a
rule that holds for you and not for the model is not a rule.

**You keep saying it.** The rules you mean to lay down you type. The other
half is what you would never think to say because you do not know you do it —
"add fuzz testing" at six tasks in a row is a rule nobody ever enunciated.
Orbit groups what you repeat, and a cheap model writes the rule in your words:

```bash
orbit rules repeated                    # what you keep telling runs
orbit rules draft -engine claude        # and the rule it amounts to
```

All three land in the same place.

## The tray

Nothing reaches a prompt without you having seen it. `orbit rules` is the one
question you have when you sit down — what do I have to decide — answered with
both halves of it:

```
  1  2026-09-13 18:52  ACME-1 · model  internal/db  the migrations are generated

e2dada18 acme                 coverage stays above 90%
         the repo has never been past 80, we are fixing that first
```

The numbered ones are sentences nobody has answered; the named ones are rules
that were sent to be looked at again.

```bash
orbit rules keep 1                      # in your words, or in better ones
orbit rules keep -in internal/db 1      # and somewhere narrower
orbit rules keep -check "make test" 1   # and make it stop the work
orbit rules drop 1                      # it was not a rule
```

## Where a rule lives

Six reaches, and the agent reads them in this order, so the last word goes to
the one closest to what is about to be touched:

```
everything          "PRs are written in English"
a language          "in Go, never discard an error with _"
a repository        "this service owns no migrations"
a directory         "everything under billing/ is money; round half to even"
a file              "schema.sql is generated — edit the generator"
a symbol            "Charge() is called from the webhook and must stay idempotent"
```

Two of them — everything, and a language — are not paths at all, so they cut
across the chain instead of hanging from it.

A rule said at a run **arrives knowing where the work was**: Orbit sees which
folder the task has been changing and the sentence comes with it, so keeping
it puts it there. Typing `-in` wins over that, and `-in .` is how you say the
whole checkout when the folder is not what you meant.

Files travel. A rule about a repository sits in `<repo>/.orbit/knowledge/`, so
whoever clones the project gets it, and one about to start steering an agent
arrives in a diff somebody reviews. A rule about everything belongs to no
checkout and stays on this machine.

## Where a rule stands

| | in the prompt? |
| --- | --- |
| **active** — confirmed and working | yes |
| **paused** — stopped by you, with a reason written down | no |
| **off** — you decided against it. It stays, and stops being told. | no |

```bash
orbit rules -state active
orbit rules pause -rule e2dada18 -why "the repo has never been past 80"
orbit rules resume -rule e2dada18
```

**A pause carries its reason.** A pause with no reason is a switch, and a
switch is what forced switching a rule off when what was needed was something
else — *"skip this while we get the coverage up"* is not disagreeing with it.
The reason is what you read when you come back, and the only thing that will
tell you whether it made sense.

## When a rule stops you

You are in the middle of something else. Two things are cheap and reversible,
and nothing else is offered:

- **skip it** — `s` on the task, or `orbit task skip`. You get past this once
  and the rule stays on.
- **pause it** — `p` on the knowledge screen, or `orbit rules pause`. It stops
  applying, and you say what for.

Either one sends the rule to be looked at again, **the first time**. Nobody
skips a rule they agree with: if you skipped it, you have already said
something without saying it.

Waiting for a decision is separate from where a rule stands, because skipping
one leaves it applying and pausing one does not, and both ask you the same
question.

Switching a rule off and rewording it are not offered in the moment. They
decide a rule's fate, and that is not a decision taken in a hurry with a task
half done.

## Warning or stopping

A rule either says something before the work, or refuses the work.

Refusing needs something that answers yes or no without an opinion in it: a
command, a pattern over the diff, a test that runs. A rule that asks to stop
and brings no check would never fire while reading as though it would — so it
warns instead, and the screen says which of the two it actually is.

The command is set by whoever keeps the rule and never by a model: a check
runs on every future phase in that repository, and a wrong or slow one is an
hour of a task spent on something nobody agreed to.

## A name of its own

Every rule Orbit writes gets a name, inside the file:

```
---
id: 875c38ec
scope: dir
source: human
path: internal/db
---

the migrations are generated, never hand-edited
```

Coined once and never again — not by correcting the sentence, not by moving
the place, not by turning it off. Everything else about a rule can change, and
the file is named after what it says, so without this nothing that comes after
could tell it was the same rule.

What it has been through is in the record rather than in the file, because it
only ever grows and a history in the file would leave a diff in your checkout
every time a gate ran:

```bash
orbit rules history -rule 875c38ec
```

```
2026-09-13 18:52:34  kept     operator
2026-09-13 18:52:35  failed   ACME-3 · test
2026-09-13 18:52:36  failed   ACME-7 · test
2026-09-13 18:52:37  skipped  ACME-7 · test
2026-09-13 18:52:40  paused   ACME-7 · test    the repo has never been past 80
```

It reads by itself: the rule refused work twice in the test phase, you got
past it once, then you stopped it. A gate that *passed* is not written down —
a rule that works is silent and a rule in the way is not, so what is kept is
the friction.

A rule you wrote by hand has no name until Orbit writes it, and is read
anyway: a header of two lines works, because writing these by hand is half the
reason they are files.

## Correcting it

The screen is the point. Every rule is there with its sentence, its place, its
source and where it stands. On any of them: correct the sentence, widen or
narrow the place, turn it off — it stays and stops being told, because
disagreeing with a rule and losing the record that it existed are two
different things.

## From the CLI you plan in

Two of Orbit's MCP tools, not shell commands — your CLI calls them once
`orbit mcp install` has registered the server. See [the CLI, both
ways](cli.md).

| Tool | What it does |
| --- | --- |
| `orbit_learn` | offer a rule about this task's code; it waits in the tray for you |
| `orbit_knowledge` | read what is already agreed before planning |

So the CLI you plan in can offer what it just worked out, and the run tomorrow
starts with it — once you have said yes.

---

Next: [the supervisor](supervisor.md) · [the map](map.md)
