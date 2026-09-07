# What Orbit knows

Facts about your code, with a source and a reach, that arrive in the prompt
before the agent works. `K` in the cockpit.

<img src="../assets/flow-knowledge.gif" alt="the knowledge screen: facts by scope, one being corrected, one widened, one turned off, and where each came from" width="900">

## Why not the model's memory

The model forgets between sessions, and forgets when you swap it for another
one. Every CLI keeps its own notes in its own file — `CLAUDE.md`, `AGENTS.md`
— and each of those is a silo that empties the day the engine changes.

What Orbit knows is Orbit's, kept outside all of them. Change the engine and
it still knows that the ledger only appends.

## Where a fact comes from

Four sources, and none of them is "the model thought so":

| | |
| --- | --- |
| **read off the code** | `ledger only appends` — because `Write` inserts and nothing updates |
| **somebody said it** | at a gate, or in the [supervisor](supervisor.md) thread |
| **a lesson** | a gate rejected something, or an attempt failed, and what happened became a fact scoped to what the task touched |
| **an incident** | from production; the shape is there, nothing reads it yet |

A sentence in the agent's context that nobody can trace is indistinguishable
from one the model made up — which is the whole point of keeping this outside
the model. A fact with no source does not get in.

## How far a fact reaches

Six scopes, and the agent reads them in this order, so the last word goes to
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

## Warning or stopping

A fact either says something before the work, or refuses the work.

Refusing needs something that answers yes or no without an opinion in it: a
command, a pattern over the diff, a test that runs. A fact that asks to stop
and brings no check would never fire while reading as though it would — so it
warns instead, and the screen says which of the two it actually is.

## Correcting it

The screen is the point. Every fact is there with its sentence, its scope, its
source, and how many times it has been told. On any of them:

- **correct** the sentence,
- **widen or narrow** the scope,
- **turn it off** — it stays and stops being told, because disagreeing with a
  fact and losing the record that it existed are two different things.

## From your CLI

```
orbit_learn        write a fact down, with its scope, its source, and its check
orbit_knowledge    read what is known about a path
```

So the CLI you plan in can teach Orbit what it just worked out, and the run
tomorrow starts with it.

---

Next: [the supervisor](supervisor.md) · [the CLI, both ways](cli.md)
