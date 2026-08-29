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
