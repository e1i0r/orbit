# Getting started

Four commands. The only one you have to think about is the third.

<img src="../assets/flow-start.gif" alt="installing orbit, registering the MCP server, opening the cockpit over a directory of repositories, and writing the first task" width="900">

## 1. Install

```bash
curl -fsSL https://raw.githubusercontent.com/e1i0r/orbit/main/install.sh | bash
```

With Go 1.26+, `go install github.com/e1i0r/orbit/cmd/orbit@latest` does the
same. macOS and Linux.

## 2. Let your CLI talk to Orbit

```bash
orbit mcp install
```

Registers the Orbit MCP server in every client found — Claude Code, Claude
Desktop, Codex, OpenCode, Gemini. Restart the CLI and it can write tasks, run
them, read what happened and leave notes without you copying anything between
windows. What it can do is in [the CLI page](cli.md).

## 3. Point it at your code

```bash
orbit top ~/code
```

That is the cockpit, over every git repository under `~/code`. There is no
list to register: the directory you open it on is the answer, and `R` shows
what it found.

Orbit runs the CLI you already have under the subscription you already pay
for. Press `M` and pick an engine; the model dial fills with that engine's own
catalogue. Nothing is called until you start a run — see
[engines](engines.md).

## 4. Write the first task

In the cockpit, `N` opens the form. A title, what has to be done, the
repository, and the flow it runs under.

From the terminal:

```bash
orbit new -repo ~/code/api -id fix-auth "the refresh token is not rotated on login"
```

Or ask your CLI, in plain language: *"create three orbit tasks in ~/code/api
for the plan we just wrote"*.

Then `n` on the row runs it, or `A` turns on [autopilot](autopilot.md) and it
starts on its own.

## Where things live

State is in `$ORBIT_HOME`, or `~/.orbit` when that is unset: one directory per
repository, one per task under it, the record of every run beside the task it
belongs to. Nothing is hidden in a database you cannot read — `orbit export`
writes it back out as JSONL.

---

Next: [what you can do](menus.md) · [a run, end to end](run.md)
