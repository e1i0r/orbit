# Orbit

A cockpit for supervising coding agents.

Open it on a folder, and every repository underneath is in scope. Tasks run in parallel,
each through a flow you can see and change before it starts. The agents are other
people's programs — opencode, codex, claude code. Orbit owns the orchestration, the
record, and the window; it owns no model, no editor, and no agent loop.

A body in orbit spends no fuel. You burn the motor only to change course, then coast
again.

**Status:** the spine works. `orbit repos`, `new`, `run`, `list` and `show` write a task
down, run it through a flow in a worktree of its own, and record what happened in an
append-only log. The window — the part that makes it a cockpit — is not built yet, and
this README is still the design-only one: install instructions, usage and a demo come
with it.