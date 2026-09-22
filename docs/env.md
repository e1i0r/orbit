# The environment

`internal/env/env.go` lists every variable Orbit reads, and whoever needs
one reads it from there. None of them is a setting. `orbit settings` holds
what you should be able to read back off a screen. This holds the keys, and
the handful of paths that decide where Orbit is looking.

## Where Orbit keeps its state

| | what it decides | example |
| :--- | :--- | :--- |
| `ORBIT_HOME` | the state root: the record, the settings, the worktrees | `~/.orbit` |
| `ORBIT_LANG` | the language for one command, over the settings file | `es` |
| `ORBIT_WORKSPACE` | the directory a task may reach across when it joins more than one repository | `~/work` |

```bash
# a second board, kept apart from your own
ORBIT_HOME=~/.orbit-demo orbit top ~/work/payments

# one command in English on a machine set to Spanish
ORBIT_LANG=en orbit quota
```

Orbit sets `ORBIT_TASK` and `ORBIT_SUPERVISOR_ENGINE` itself, on the
processes it starts: the id of the task being run, and the engine a
supervisor is running under. A run reads them. You never write them.

## The keys

Every key is optional. Each one turns on something Orbit can do, and
without it that thing is not offered at all. No key ever stops a run.

| | what it buys | without it |
| :--- | :--- | :--- |
| `LINEAR_API_KEY` | a task written from a Linear URL arrives with the issue's body in it | the form says it cannot read the issue and asks you to write what has to be done |
| `TYPESAFE_API_KEY` | the decision engine judges a stopped run in half a second, so gates it is sure about stop waiting for you | the supervisor works the way it always has |
| `ANTHROPIC_API_KEY` | Orbit can tell claude is billed per token here, and read what has been spent | claude reads as a subscription, and the allowance is whatever the engine reports |
| `ORBIT_TELEGRAM_TOKEN` | `orbit chat` answers over Telegram | the chat answers at the terminal only |

```bash
# in the shell you start orbit from, or in your profile
export LINEAR_API_KEY=lin_api_…
export TYPESAFE_API_KEY=apikey_…
export ANTHROPIC_API_KEY=sk-ant-…
export ORBIT_TELEGRAM_TOKEN=123456:AA…

orbit top ~/work/payments
```

A key is read at the moment it is used. A shell that exports one after the
window is already open is a window with no key, so start it again.

## Proxies, and where codex keeps its things

| | what it decides |
| :--- | :--- |
| `ANTHROPIC_BASE_URL` | where claude's allowance is read from, when a proxy stands in front of it |
| `OPENAI_BASE_URL` | the same for codex |
| `CODEX_HOME` | where codex keeps its own state, which is where its rollouts and transcripts are read |

```bash
# a proxy that reports what each model has spent
export ANTHROPIC_BASE_URL=http://localhost:8787
```

## Why no key is a setting

`orbit settings` prints its table to a terminal, to a screen and into a
chat. A secret that can be printed is a secret that will be. So the
settings hold what you should be able to read back, like the account
`orbit chat` answers or how sure the decision engine has to be, and the
environment holds the rest.

Turning the decision engine on takes both a key and a setting:

```bash
orbit settings set decisions shadow   # it writes down what it would decide
orbit settings set decisions on       # it acts on what it is sure about
```

Either one alone does nothing, and says so rather than looking switched on.
