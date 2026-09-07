<div align="center">

<img src="assets/logo.png" alt="Orbit" width="360">

**Stay in command of the code you did not write.**

[![CI](https://github.com/e1i0r/orbit/actions/workflows/check.yml/badge.svg)](https://github.com/e1i0r/orbit/actions)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8.svg?logo=go)](https://go.dev)
[![TUI](https://img.shields.io/badge/TUI-Bubble%20Tea%20v2-FF5F87.svg)](https://charm.sh)

</div>

---

A terminal cockpit for coding agents. Each task runs in its own git worktree, in phases that stop where you tell them to, and everything is written down: the plan, the prompts, the reasoning, the diff, the cost, what the agent refused to do.

It runs the CLI you already have — Claude Code, Codex, OpenCode, Antigravity — under the subscription you already pay for. No API key, no second bill.

```bash
curl -fsSL https://raw.githubusercontent.com/e1i0r/orbit/main/install.sh | bash
```

<img src="assets/loop.gif" alt="the cockpit, the record a finished task kept, and the terminal handed to a CLI and handed back" width="720">

---

## The flows

Nine things you will actually do, each with its own page and its own recording.

| | | |
| --- | --- | --- |
| 1 | [**Getting started**](docs/start.md) | install, register the MCP server, add a repository, write the first task |
| 2 | [**What you can do**](docs/menus.md) | `m` on anything: every verb, including the ones it will not let you press, with the reason |
| 3 | [**A run, end to end**](docs/run.md) | a task through its flow: phases, gates, what stops it and what lets it go |
| 4 | [**Autopilot**](docs/autopilot.md) | the queue works itself; the brake that keeps it from outrunning you |
| 5 | [**Many tasks at once**](docs/parallel.md) | one worktree each, running side by side, in one window |
| 6 | [**Reading what it did**](docs/reading.md) | the report, the diff, and what the change reaches that the diff cannot show |
| 7 | [**The CLI, both ways**](docs/cli.md) | hand the terminal to your CLI and take it back; the MCP server it talks through |
| 8 | [**The supervisor**](docs/supervisor.md) | the second pair of eyes, and the thread you steer it in |
| 9 | [**What Orbit knows**](docs/knowledge.md) | the facts it carries into every prompt, and how you correct them |

Reference: [keys and screens](docs/cockpit.md) · [engines and models](docs/engines.md)

---

## Status

`v0.1.x`, used daily by its author. It is young and the version number means it.

| | |
| --- | --- |
| **Stable** | the cockpit, flows and gates, worktree isolation, the record in SQLite, the MCP server, GitHub pull requests |
| **In use, still moving** | the supervisor, autopilot, what Orbit knows, the impact reading, quota tracking, the flow designer |
| **Engines** | Claude Code, Codex, OpenCode, Antigravity (`agy`) |
| **Platforms** | macOS and Linux |
| **Not there yet** | the impact reading is per file, not per symbol; issue bodies come from Linear and no other tracker yet |

Other ways to install: `go install github.com/e1i0r/orbit/cmd/orbit@latest` with Go 1.26+, or `make build` from source.
English and Spanish (`orbit settings language es`). Six themes. State lives in `$ORBIT_HOME`, or `~/.orbit`.

## Contributing

Issues and pull requests are welcome. [CONTRIBUTING.md](CONTRIBUTING.md) is how this code is written, and it is short. `make check` has to pass, and new behaviour needs a test that fails without it.

## License

Apache 2.0. See [LICENSE](LICENSE).
