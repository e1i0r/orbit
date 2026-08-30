<div align="center">

<img src="assets/logo.png" alt="Orbit" width="360">

**Stay in command of the code you did not write.**

[![CI](https://github.com/e1i0r/orbit/actions/workflows/check.yml/badge.svg)](https://github.com/e1i0r/orbit/actions)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8.svg?logo=go)](https://go.dev)
[![TUI](https://img.shields.io/badge/TUI-Bubble%20Tea%20v2-FF5F87.svg)](https://charm.sh)

</div>

---

Orbit is a terminal cockpit for coding agents. Each task runs in its own git worktree, in phases that stop where you tell them to, and everything is written down: the plan, the prompts, the reasoning, the diff, the cost, what the agent refused to do.

It is built against two failure modes:

- **Today** — the agent writes more code than you can read, and you approve it anyway. Orbit keeps you the one who decides what ships.
- **In six months** — nobody on the team knows what is in the repo or why it was decided that way. Orbit keeps the full record of every run next to the code, not in a chat window somebody closed.

  ```bash
  orbit show -repo ~/code/api fix-auth   # every phase, its gate verdicts, its refusals, its cost
  ```

---

## The loop

<img src="assets/loop.gif" alt="the cockpit, the record a finished task kept — its report, its diff, its timeline, what it changed — and the terminal handed to a CLI and handed back with the next task written down" width="720">

1. In your CLI — Claude Code, Codex, OpenCode — you investigate, validate, and build the plan. From there it creates the tasks and hands them to Orbit over MCP.
2. You open the cockpit and turn on autopilot.
3. Orbit runs each task in its own worktree, phase by phase, stopping at the gates. The supervisor checks the result, fixes what is missing, and reports back.
4. You read the diff and ship the PR.

Then you go back to your CLI, and it can read everything that happened while you were away.

---

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/e1i0r/orbit/main/install.sh | bash
```

## Quickstart

**1. Let your CLI talk to Orbit.**

```bash
orbit mcp install
```

Registers the Orbit MCP server in Claude Code, Claude Desktop, Codex, OpenCode and Gemini. Restart the CLI and it can create tasks, run them, read their state and write notes.

**2. Send it work.** Ask your CLI, in plain language: *"create three orbit tasks in ~/code/api for the plan we just wrote"*. Or write one by hand:

```bash
orbit new -repo ~/code/api -id fix-auth "the refresh token is not rotated on login"
```

**3. Open the cockpit.**

```bash
orbit top ~/code
```

Press `A` for autopilot and Orbit starts working the queue. `Enter` opens a task, `0` shows its diff, `p` pauses it.

---

## Your plans, not API keys

Orbit never calls a model. It runs the CLI you already have installed, under the subscription you already pay for — so there is no API key to configure and no second bill per token.

Pick an engine — Claude, Codex, OpenCode — and the model dial fills with that engine's own catalogue. `default` leaves the choice to the CLI. The full list is in [docs/engines.md](docs/engines.md).

---

## Flows and gates

A flow is a list of phases. Each phase names its own engine, model, reasoning effort, thinking mode, prompt, and what it is allowed to touch (`read`, `repo`, `network`). A phase marked `wait` is a **gate**: the run stops there and waits for you.

Four flows ship built in:

| Flow | Phases | For |
| --- | --- | --- |
| `quick` | implement | small changes, minimal overhead |
| `task` | implement → review ⏸ | the default |
| `careful` | implement → review ⏸ → fix | mission-critical work |
| `tdd-fuzz-pr` | plan → implement + fuzz → review + PR ⏸ | test-first, ending in a pull request |

Flows are JSON, and phase names are yours. A release flow is as legitimate as a coding one:

```json
{"name":"ship","phases":[
  {"name":"validate","engine":"claude","model":"opus","permissions":["read"]},
  {"name":"tests","engine":"claude","model":"sonnet","permissions":["repo"]},
  {"name":"pr","engine":"claude","model":"sonnet","feed_output":true,"permissions":["repo"]},
  {"name":"checks","engine":"claude","model":"sonnet","permissions":["repo"]},
  {"name":"merge","engine":"claude","model":"opus","wait":true,"permissions":["repo"]}
]}
```

Drop it in `~/.orbit/flows/`, or build it in the cockpit with `+`. `orbit flows` lists what you have.

Between every pair of phases Orbit asks whether to continue, and the answer is recorded — including "nobody was asked, autopilot was on". A gate that fails or a phase you paused leaves the task in **Needs You**, where it stays until you look at it.

## Why gates

One long session with an agent has a single verification point: the end. By then the wrong assumption from minute three is buried under four hundred lines that all look plausible, and reviewing it costs more than writing it did.

Phases cut the run into pieces that are each small enough to check. A phase that gets it wrong is caught by the next phase or by you, before the one after it builds on the mistake. That is also what makes the record readable later: eleven short steps with a verdict on each, instead of one transcript nobody will open.

## Supervisor and autopilot

The supervisor is a run like any other: it goes out through the same CLI and subscription as your tasks, on whichever engine the dial is set to. It is the second pair of eyes — it reads the result of a run, decides whether it actually did what the task asked, and can go fix what is missing. There is a chat thread you can write into to redirect it — `orbit supervisor "the migration has to be reversible"` — and it persists, so it is still there next session.

Autopilot is the switch between two modes:

- **On** — Orbit picks up the next To Do, runs it through its flow without stopping at the flow's own gates, and lets the supervisor try to resolve whatever comes back needing attention.
- **Off** — it runs what you start and hands everything else back to you.

Autopilot lifts the flow's gates. It does not lift a pause you set by hand. And `unread-cap` (10 by default) is the brake: once that many finished tasks are sitting unread, nothing new starts. The queue cannot outrun you.

## Bidirectional MCP

Orbit is an MCP server with 19 tools, so the flow goes both ways. Your CLI writes tasks down, runs them, reads what happened, pauses and redirects them, adds notes, and manages flows and repositories — without you copying anything between windows. Writing a task and paying to run it stay two separate decisions.

The agents Orbit runs report back through the same protocol. They do not have their stdout scraped and guessed at — they say what they did, in structured events, which is why the record is complete enough to be worth reading in six months.

```bash
orbit mcp install          # register in every client found
orbit mcp                  # run the server on stdin/stdout
```

## Integrations

- **CLIs:** Claude Code, Codex, OpenCode — run natively, with your own subscription.
- **MCP clients:** Claude Code, Claude Desktop, Codex, OpenCode, Gemini.
- **GitHub:** `orbit pr`, `orbit merge`, `orbit close-pr` — pull requests from a task's worktree, through `gh`.
- **Issue trackers:** paste a Linear, Jira, GitHub or GitLab URL into the composer and Orbit recognises the reference and writes it into the task prompt.

---

## The cockpit

`orbit top` is one window over every repository. Tasks sit in four bands — **To Do**, **Needs You**, **Running**, **Done** — and move between them on their own as runs progress.

Open a task and eleven tabs cover it: `overview`, `flow`, `gates`, `cost`, `refused`, `timeline`, `report`, `artifacts`, `notes`, `diff`, `thinking`.

Full keybindings in [docs/cockpit.md](docs/cockpit.md). `?` inside the cockpit shows them too.

Everything the cockpit does has a command behind it — `orbit new`, `run`, `pause`, `resume`, `list`, `show`, `read`, `pr`, `merge`, `cancel`, `note`, `direct`, `supervisor`, `settings`. Run `orbit help` for the full list. State lives in `$ORBIT_HOME`, or `~/.orbit` when that is unset.

```bash
orbit settings                    # show every setting and its value
orbit settings autopilot on
orbit settings theme frauddi
```

---

## Project status

Orbit is at `v0.1.x` and is used daily by its author. It is young, and the version number means it.

| | |
| --- | --- |
| **Stable** | the cockpit, flows and gates, worktree isolation, the record, the MCP server, GitHub pull requests |
| **In use, still moving** | the supervisor and autopilot, quota tracking, the flow designer |
| **Engines that work today** | Claude Code, Codex, OpenCode |
| **Platforms** | macOS and Linux |
| **Not there yet** | issue bodies are not fetched from trackers, only the reference is read; storage is flat files, a SQLite move is planned |

---

## Details

**Other ways to install:** with Go 1.26+, `go install github.com/e1i0r/orbit/cmd/orbit@latest`. From source, `git clone https://github.com/e1i0r/orbit.git && cd orbit && make build && sudo mv orbit /usr/local/bin/`.

**Languages:** English and Spanish, switched with `orbit settings language es`.

**Themes:** `frauddi` (default), `monokai`, `tokyo-night`, `dracula`, `nord`, `catppuccin`.

<details>
<summary><b>Repository layout</b></summary>

```
orbit/
├── cmd/orbit/        # the binary
└── internal/
    ├── arch/         # the import map, enforced as a test
    ├── board/        # queues, bands, board state
    ├── cli/          # subcommands
    ├── engine/       # adapters for Claude Code, Codex, OpenCode
    ├── flow/         # flow schema, resolver, validator
    ├── logger/       # orbit.log and errors.log
    ├── mcp/          # the MCP server and its 19 tools
    ├── quota/        # rate limit windows
    ├── record/       # append-only JSONL event store
    ├── repo/         # git worktrees, status, diffs
    ├── store/        # state root and persistence
    ├── supervisor/   # the supervisor thread
    ├── task/         # lifecycle, executor, gates
    ├── tracker/      # Linear, Jira, GitHub, GitLab references
    ├── ui/           # the Bubble Tea cockpit
    ├── view/         # read models
    └── words/        # en / es catalogues
```

</details>

**Quality bar:** no Go file over 300 lines, ≥90% test coverage as the target, no silently discarded errors. Fuzz tests, golden-file rendering tests, and an architecture test that fails the build on an import that is not on the map.

```bash
make check      # gofmt, vet, lint and tests on macOS and Linux
make test
make coverage
```

## Contributing

Issues and pull requests are welcome. `make check` has to pass, and new behaviour needs a test that fails without it.

## License

Apache 2.0. See [LICENSE](LICENSE).
