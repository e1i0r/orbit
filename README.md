# Orbit

A cockpit for supervising coding agents.

Open it on a folder, and every repository underneath is in scope. Tasks run in parallel,
each through a flow you can see and change before it starts. The agents are other
people's programs — opencode, codex, claude code. Orbit owns the orchestration, the
record, and the window; it owns no model, no editor, and no agent loop.

A body in orbit spends no fuel. You burn the motor only to change course, then coast
again.

**Status:** the spine works and the window opens. `orbit repos`, `new`, `run`, `list` and
`show` write a task down, run it through a flow in a worktree of its own, and record what
happened in an append-only log. `orbit top` is one window over all of it. This README is
still the design-only one: install instructions and a demo come with the release.

## The window

```
orbit top [dir] [-once] [-lang <code>]
```

`dir` is the folder the header names and the one the empty state points at; the tasks
themselves come from the state root, so every repository you have ever written a task
against is in the window whether or not it sits under that folder. Without it, the folder
you are standing in.

`q` closes it and `?` lists every key. Opening it writes nothing except the one sweep that
closes the records of runs whose processes are gone — the same sweep `orbit reconcile`
runs, over every repository at once.

`-once` draws a single frame as plain text and exits. It is the same board through the
same view functions with the styling taken off and every band open, which is what a pipe,
a log, a CI job and `TERM=dumb` get:

```
orbit top ~/work > board.txt
```

You do not have to ask for it. A window whose output is not a terminal draws that frame
rather than seizing a screen nobody is looking at.

`-lang` is the language of this window only. Without it, `$ORBIT_LANG`, then the language
`orbit set language es` saved, then the `$LANG` the terminal was started in, then English.