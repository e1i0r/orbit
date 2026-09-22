# Orbit by chat

Orbit only exists where the terminal is. You go to lunch and a task waits on a
three-second decision until you come back.

The board on a phone is not the answer. Fourteen panes do not fit on one.
What fits is typing a command and being answered.

```
/board
  ACME-3  needs you   claude ran out: implement · back in 1h
  ACME-7  running     implement · $0.42

/task start ACME-3 -engine codex
  started ACME-3 on codex
```

## Set it up

1. **Make a bot.** In Telegram, write to **@BotFather** and send `/newbot`. It
   asks for a name and a username ending in `bot`, and answers with a token.
   Whoever holds that token can read and write as the bot.

2. **Put the token in your shell**, never in a file. It is a secret, and the
   settings file is not where a secret goes.

   ```bash
   export ORBIT_TELEGRAM_TOKEN=8123456789:AAH...
   ```

3. **Start it and write to it.** With nobody allowed yet, Orbit answers nobody
   and tells the first person who writes what their own id is.

   ```bash
   orbit chat -on telegram
   ```

   ```
   nobody is allowed to command this Orbit yet. Your chat id is 7912204269,
   run `orbit settings set chat-id 7912204269` and start me again.
   ```

4. **Say who may command it**, and start it again.

   ```bash
   orbit settings set chat-id 7912204269
   orbit chat -on telegram
   ```

   Or on the settings screen in the cockpit, `:` then `settings`, where
   `chat-id` is a row like any other.

There is no server to run and no address to have. Orbit calls Telegram and
waits, and Telegram does the push to your phone.

## What you can ask for

Everything the terminal offers. Type `/` and the list appears with a line
under each. It is built from the same declaration the command line, the
window, the browser and the MCP server are built from, so a verb added to
Orbit can be asked for from a phone the same day.

| | |
| --- | --- |
| `/help` | the list, as a message |
| `/board`, `/task show ACME-3` | read |
| `/task start ACME-3 -engine codex` | act |
| `/board new -id ACME-12 "the webhook retries on 5xx"` | write a task down |

**The words are the ones you already know.** `/task show` and `orbit task
show` are the same verb reaching the same body, with the same flags in the
same order. Quotes hold a sentence together, because a chat has no shell in
front of it to do that.

A few are not offered, and say so rather than failing somewhere deeper: a
terminal cannot be handed to an engine through a chat, and a diff, a tree and
an impact are read in columns a phone would turn into a scroll bar.

**What cannot be taken back asks twice.** `/pr merge`, `/pr close` and
`/task delete` answer *send /yes to go ahead*. Changing the subject clears it,
so a confirmation cannot fire later against something nobody was talking about
any more.

## Talking to it

A line without a slash is a sentence said to the supervisor, and it answers.

```
you:    cómo va?
orbit:  Tranquilo — no se movió nada. Hay una tarea, CHAT-1, y sigue en to do.
```

It is the same thread `S` opens in the window, so something said from a bus
reaches the next run's prompt exactly as it would have from the desk. It runs
on the engine and model your settings name, and it costs a model run per
sentence, which is why a build with no engine reachable writes the line into
the thread and says so rather than dropping it.

## Who can command it

One account, named in `chat-id`. A message from anybody else does nothing and
is written to the log.

That is the whole of the gate, and it is one and not a list on purpose: a list
is a thing that grows by accident, and a channel anybody can join is a channel
anybody can cancel a run from.

## Another service

The channel is a port, the way an engine is one. Telegram is one file,
`internal/chat/telegram.go`, and a second service is another file, not a
migration.

What a new one implements is small:

| | |
| --- | --- |
| `Listen` | hand over every message until the context is done. Never end the loop over a service that is merely unreachable |
| `Say` | answer into one conversation. A channel that is down must not be able to stop a run |
| `Name` | what the log calls it |

And two it may implement, asked for by type rather than declared, because not
every service has them:

| | |
| --- | --- |
| `Working` | show that an answer is coming. A model run is forty seconds, and forty seconds of silence on a phone is indistinguishable from a bot that is not running |
| `Announcing` | publish the command menu, so the list appears as the reader types |

Everything else belongs to the desk and is inherited: reading a message as
a verb, the gate, the confirmations, the wording. An adapter that behaves
differently from `orbit chat` at a terminal is a bug in the adapter.

**Markup is the adapter's.** The answer arrives with this package's own
marks, a fence around a listing and a pair of control characters around the
name a row is filed under, and each service turns them into what it has.
Telegram turns them into `<pre>` and `<b>`. The terminal takes them out.

Two kinds of answer arrive, and they want opposite treatment: a verb's output
is columns padded for a terminal, full of punctuation nobody meant as
formatting, and the supervisor's is prose a model wrote to be read. Prose
marks itself, and no adapter escapes what is inside the mark.

> **Not MarkdownV2**, and the reason is worth knowing before trying it again:
> it reserves eighteen characters and rejects, rather than mangles, any
> message with one of them unescaped. Two are the full stop and the hyphen. So
> Orbit's own output cannot go unescaped (`unread-cap` loses the message) and
> the supervisor's prose cannot go escaped (its formatting is deleted) and
> cannot go unescaped either (a sentence ends in a full stop). HTML reserves
> three.

## Being told

```bash
orbit settings set notify on
```

Or the `notify` row on the cockpit's settings screen, which is the same
switch.

One switch, off by default: a program that starts interrupting somebody the
day they install it has made a decision that was theirs. On, it reaches the
desktop, and the chat as well when one is set up. Where it reaches is a fact
about what you have configured, not a second question.

The line is **what changes where a task is**, and the list is closed:

| | |
| --- | --- |
| it began | the run started, or changed engine halfway |
| it stopped | at a gate, out of attempts, out of allowance, over budget, over the diff, on a new dependency, against a decision, or broken |
| it ended | finished, a pull request opened, merged, cancelled, timed out, abandoned |

A task's whole arc is a handful of these, and somebody away from their desk
can follow it.

What is left out is everything **inside** a phase: a phase starting, a tool
call, a thought, a cost going up. That is a run working rather than a run
moving, and there are hundreds of them: **a channel that tells you everything
is a channel you mute**, which is worse than none, because you believed you
would be told.

Every message carries what to do about it:

```
ACME-3 — claude ran out in implement; codex, opencode could take it on

/task start ACME-3 -engine codex
```

A message you cannot answer is a phone opened for nothing, and it is how a
reader learns to ignore the next one. The one ending with nothing to offer,
where no engine has anything left, offers nothing, because a command that cannot
work is worse than none.

It is a poll of the record and not a hook, and that is deliberate: a run is
its own process and can be killed, so a notification that depended on the
dying process to send it is the one that never arrives for the task that
most needed it. It starts from wherever the record is when the chat opens,
a chat started this afternoon telling you about Tuesday is a chat you scroll
past.

---

Next: [the CLI, both ways](cli.md) · [the supervisor](supervisor.md)
