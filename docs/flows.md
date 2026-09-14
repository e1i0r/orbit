# Flows you write yourself

A flow is a list of phases. `F` in the cockpit lists the ones this machine can
run and writes new ones.

<img src="../assets/flow-flows.gif" alt="the flows screen: the five that ship and the ones written here, each phase with its engine, its model and whether it stops for a person, and the designer that writes a new one" width="900">

## What a phase carries

Each one names its own engine, model, reasoning effort, thinking mode, the
prompt it is given, and what it is allowed to touch — `read`, `repo`,
`network`. A phase marked `wait` is a gate: the run stops in front of it and
waits for you.

That is the whole schema. There is nothing else to learn.

## The five that ship

| Flow | Phases | For |
| --- | --- | --- |
| `quick` | implement | small changes, minimal overhead |
| `task` | implement → review ⏸ | the default |
| `careful` | implement → review ⏸ → fix | mission-critical work |
| `coverage` | implement → until it passes ↻ → review ⏸ | a loop that goes round until a check is green |
| `tdd-fuzz-pr` | plan → implement + fuzz → review + PR ⏸ | test-first, ending in a pull request |

The screen shows each one opened out: every phase, what it runs on, and which
of them stop. A flow you cannot read before you pay for it is a flow you are
trusting rather than choosing.

## Writing one

`n` on that screen opens the designer. Or write the JSON — they are files in
`~/.orbit/flows/`, and saving one under a built-in's name covers it:

```json
{"name":"ship","phases":[
  {"name":"validate","engine":"claude","model":"opus","permissions":["read"]},
  {"name":"tests","engine":"claude","model":"sonnet","permissions":["repo"]},
  {"name":"pr","engine":"claude","model":"sonnet","feed_output":true,"permissions":["repo"]},
  {"name":"checks","engine":"claude","model":"sonnet","permissions":["repo"]},
  {"name":"merge","engine":"claude","model":"opus","wait":true,"permissions":["repo"]}
]}
```

The phase names are yours. Nothing in Orbit assumes a flow is about writing
code — a release flow, a migration flow, a triage flow are all the same shape,
and they all get the same record, the same gates and the same cockpit.

## Why the phases are small

One long session with an agent has a single verification point: the end. By
then the wrong assumption from minute three is buried under four hundred lines
that all look plausible.

A flow is how you say where the checks go. Put a gate where you would have
looked anyway, and the run stops there — see [a run, end to end](run.md).

```bash
orbit flows                       # what this machine can run
orbit board new -repo ~/code/api x "…"   # written against the default flow
orbit settings flow careful       # or change which one that is
```

---

Next: [a run, end to end](run.md) · [autopilot](autopilot.md)
