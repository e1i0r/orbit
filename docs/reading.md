# Reading what it did

The diff says what was written. These say whether it was right.

<img src="../assets/flow-reading.gif" alt="the overview of a finished task, then the report, the diff card by card, and the impact tab: what usually comes along and did not" width="900">

## Overview — the answer in one screen

`1`. What the task was, in the words it was written in. What it spent, how
long it took, what flow it ran. The story: how this prompt became this diff,
as a chain — the way in, what it is for, what went wrong, why, what was done —
each field the reason for the one under it, and under that every file the run
changed, in the order it got there.

Then the deliver toolbar: `p` opens the pull request, `C` fixes the failing
checks, `R` brings the review comments back for the next run.

## Report — what it says it did

`7`. What the engine wrote about the change, per phase, seamed where one
attempt ends and the next begins. When the record kept less than the engine
printed, the pane says so and by how much: a summary that quietly lost half
its bytes is worse than no summary.

## Diff — what it actually wrote

`0`. One card per file, with the sentence from the record saying why that file
was touched. `Z` collapses them all, `Enter` opens one, `o` opens the file
under the cursor in `$EDITOR` at the right line.

Long lines scroll rather than wrap — a generated file arrives as one line of
several thousand cells, and wrapping it would push the rest of the hunk off
the screen. `e` sets them whole when you want them whole.

## Impact — what the diff cannot show

`i`. This is the one nobody else answers.

**What usually comes along.** Read from this repository's own history: files
that have been committed together with the ones this task changed, and that it
did not touch this time. `invoice.py · 87% of the time (34/39) · not touched`
is not a rule and the pane says so — a file left out on purpose is the normal
case — but it is the question a reviewer cannot ask a diff.

**What those tests say they hold.** The names of the tests in those files,
read as sentences. `rejects_negative_amounts` is a specification somebody
already wrote, and it is the cheapest one any repository has.

**What the checks say, both sides.** `r` runs the flow's own checks on the
branch this work was cut from and on the work itself, at the same time. An
exit code decided every line of it — nothing there is anybody's reading. It is
a test suite twice on your machine, so the pane names the cost before you pay
it.

**What the agent says it did.** Last, and marked: what the change now asks of
its callers, what it promises them, what it took for granted, and what it
considered and did not do. Nobody verified it — no command can, for "assumes
UTC timestamps" — and the discarded alternatives exist nowhere else. They die
with the run, and the next person to touch that code pays again to find out
why the obvious approach was not taken.

## The rest

| | |
| --- | --- |
| `3` gates | every check, whether it passed, what it ran, and why it stopped |
| `4` cost | what each phase spent, and what it adds up to |
| `5` refused | tool calls the sandbox denied, and the rules it denied them by |
| `6` timeline | the run event by event, seamed between attempts |
| `8` artifacts | every file the run left, opened where you want to read one |
| `9` notes | your notes, the sessions beside the run, what it stopped to ask |
| `w` thinking | the reasoning it showed its work in |

`e` opens every row of any of them at once; `v` shows what was written down
rather than what was made of it.

## Six months later

```bash
orbit show -repo ~/code/api fix-auth   # every phase, its gates, its refusals, its cost
orbit export ~/somewhere               # the whole record back out as JSONL
```

---

Next: [the CLI, both ways](cli.md) · [the supervisor](supervisor.md)
