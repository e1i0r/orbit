# The CLI, both ways

Orbit does not replace the CLI you already use. It hands it the terminal and
takes it back, and it talks to it over MCP while you are away.

<img src="../assets/flow-cli.gif" alt="pressing c on a task, the CLI opening in that task's worktree with orbit's context and MCP server, quitting, and orbit offering to write the session down as a task" width="900">

## Hand the terminal over

`c` on a task. Your CLI, whether Claude Code, Codex, OpenCode, Antigravity or Cline, opens **in that
task's worktree**, carrying the task's own context and Orbit's MCP server. You
are in a normal session, in the right directory, with everything the run knows
already in front of it.

Quit it and you land back on the board. Orbit asks whether to write the
session down as a task. Press `y` and it is on the queue with what you just did in
it.

That is the whole point of the key. The work you do by hand and the work an
agent does are the same work, and neither should have to be re-explained to
the other.

| Key | What |
| :--- | :--- |
| `c` | open your CLI in this task's worktree |
| `t` | take the keyboard from a run that is going |
| `h` | hand it back and let the run carry on |

## Talk to Orbit from anywhere

```bash
orbit mcp install     # register in every client found
orbit mcp             # run the server on stdin/stdout
```

Twenty-two tools, so the flow goes both ways. Your CLI can:

| | |
| --- | --- |
| **write and run** | `orbit_create_task`, `orbit_retry_task`, `orbit_direct_task`, `orbit_pause_task`, `orbit_cancel_task`, `orbit_requeue_task` |
| **read** | `orbit_get_board_summary`, `orbit_list_tasks`, `orbit_inspect_task`, `orbit_list_repos`, `orbit_inspect_repo` |
| **steer** | `orbit_add_note`, `orbit_supervisor_say`, `orbit_supervisor_history` |
| **manage** | `orbit_list_flows`, `orbit_get_flow`, `orbit_save_flow`, `orbit_delete_flow`, `orbit_add_repo`, `orbit_forget_repo` |
| **teach** | `orbit_learn`, `orbit_knowledge`, see [what Orbit knows](knowledge.md) |

So the loop is: in your CLI you investigate, validate and build the plan.
From there it writes the tasks and hands them to Orbit, you open the cockpit
and let them run, and when you go back to your CLI it can read everything that
happened while you were away.

Writing a task and paying to run it stay two separate decisions.

## Orbit by chat

```bash
orbit chat
```

A fifth way in, for a screen too small to draw a board on. You type a command
and it answers:

```
/board
  ACME-3  needs you   claude ran out: implement · back in 1h
  ACME-7  running     implement · $0.42

/task start ACME-3 -engine codex
  started ACME-3 on codex

/board new -id ACME-12 "the webhook retries on 5xx"
  ACME-12 written down against payments
```

**The words are the ones you already know.** `/task show` and `orbit task
show` are the same verb reaching the same body: the list of commands is built
from the one declaration every way in is built from, so a verb added to Orbit
can be asked for from a chat the same day. `/help` prints it.

A few are not offered, and say so rather than failing somewhere deeper: a
terminal cannot be handed to an engine through a chat, and a diff, a tree and
an impact are read in columns that a phone would turn into a scroll bar.

**What cannot be taken back asks twice.** `/pr merge`, `/pr close`, `/task
delete` answer *send /yes to go ahead*, and changing the subject clears it,
so a confirmation cannot fire later against something nobody was talking
about any more.

Prose is not a command and not a mistake either. A line without a slash is
answered with nothing, because a channel that replies to every stray sentence
is a channel that gets muted.

Today the only chat it speaks is the terminal itself. That is on purpose: it
is the same loop, the same gate and the same confirmations a service adapter
will run, so an adapter that behaves differently from `orbit chat` is a bug
in the adapter.

## The other direction

The agents Orbit runs report back through the same protocol. They do not have
their stdout scraped and guessed at. They say what they did, in structured
events, which is why the record is complete enough to be worth reading in six
months.

---

Next: [the supervisor](supervisor.md) · [what Orbit knows](knowledge.md)
