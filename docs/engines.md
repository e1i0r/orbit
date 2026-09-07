# Engines, models and reasoning

Orbit never calls a model. It runs the CLI you already have installed, under the
subscription you already pay for. These are the catalogues those CLIs offer, as
Orbit's dials present them.

Every dial also offers `default`, which passes no flag at all and leaves the
choice to the CLI.

## Claude

Runs `claude`.

| | |
| --- | --- |
| **Models** | `opus`, `sonnet`, `haiku` |
| **Reasoning effort** | `low`, `medium`, `high`, `xhigh`, `max` |

## Codex

Runs `codex exec`.

| | |
| --- | --- |
| **Models** | `gpt-5.6-terra`, `gpt-5.6-luna`, `gpt-5.5`, `gpt-5.4-mini` |
| **Reasoning effort** | `none`, `low`, `medium`, `high`, `xhigh` |

`max` is absent on purpose: only `gpt-5.6-terra` and `gpt-5.6-luna` accept it,
and the effort dial is not per-model, so offering it would give a position that
fails depending on where a different dial is pointing.

## OpenCode

Runs `opencode run`. Model ids carry the `opencode/` prefix; the dial shows them
without it.

| | |
| --- | --- |
| **Models, paid** | `claude-opus-5`, `claude-sonnet-5`, `gpt-5.3-codex`, `gemini-3.1-pro`, `grok-4.6` |
| **Models, free** | `nemotron-3-ultra-free`, `nemotron-3.5-lightning-free`, `mimo-v2.5-free`, `ling-3.0-flash-fin-free`, `hy3-free`, `muse-spark-1.2-contributor-free` |
| **Reasoning effort** | `minimal`, `medium`, `high` |

## Antigravity

Runs `agy`. It is named for the program and not the product: the engine name in
a record is also what the window runs when you ask for a session, and nothing
on the machine answers to `antigravity`.

| | |
| --- | --- |
| **Models** | `gemini-3.8-flash`, `gemini-3.7-flash`, `gemini-3.6-flash` and `gemini-3.1-pro`, each in the reasoning tiers Gemini names inside the model; plus `claude-sonnet-4-6`, `claude-opus-4-6-thinking`, `gpt-oss-120b-medium` |
| **Reasoning effort** | `low`, `medium`, `high` |

Gemini names its reasoning inside the model — a family appears three times, high,
medium and low — while `--effort` says the same thing beside it. Both are offered
as they are printed, because what a dial shows and what the binary is told have
to be the same string. Refresh the catalogue with `agy models`.

There is no thinking dial for agy, and that is not an omission: its models think,
the stream counts the thinking tokens on every step, and how much is asked for by
effort. A second dial would be a second name for the one beside it.

**No transcript.** agy keeps each conversation in a SQLite database of its own
whose steps are protobuf blobs with no schema shipped. Which conversation belongs
to a worktree is answerable; what was said in it is not. Orbit says so rather
than guessing — walking that wire format field by field would be a guess written
into a task's record as if it were an account of the session.

## Thinking

The flow designer sets a phase's thinking mode to `adaptive`, `on` or `off`.
The composer sets a budget instead: `adaptive`, `off`, `4000`, `8000`, `max`.

## Permissions

Each phase declares what it may touch. The engine adapter turns that into
whatever sandbox flag its CLI understands.

| Permission | What it grants |
| --- | --- |
| `read` | read the worktree, write nothing |
| `repo` | write inside the task's worktree |
| `network` | reach the network |

Codex has no sandbox that opens the network without also granting writes, so a
Codex phase asking for `network` alone is refused rather than run under a
posture the record would describe wrongly.

## Defaults

```bash
orbit settings engine claude
orbit settings model opus
```

A phase that names its own engine or model overrides these. A phase that names
neither takes them.
