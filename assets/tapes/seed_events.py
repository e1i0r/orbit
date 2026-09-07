#!/usr/bin/env python3
"""The record behind the demo board.

One JSONL file per task, in the layout an older Orbit wrote. The migration
that runs before every command copies them into the record, so nothing here
reaches past a door the product already has.
"""

import datetime
import json
import pathlib
import sys

D = pathlib.Path(sys.argv[1])
CODE = pathlib.Path(sys.argv[2])
TASKS = D / "home" / "tasks"

NOW = datetime.datetime.now(datetime.timezone.utc).replace(microsecond=0)


def ago(minutes):
    return (NOW - datetime.timedelta(minutes=minutes)).isoformat().replace("+00:00", "Z")


def ev(minutes, kind, phase=None, text=None, **data):
    e = {"at": ago(minutes), "kind": kind}
    if phase:
        e["phase"] = phase
    if text:
        e["text"] = text
    if data:
        e["data"] = {k: str(v) for k, v in data.items()}
    return e


def write(task, events):
    p = TASKS / task / "events.jsonl"
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text("\n".join(json.dumps(e) for e in events) + "\n")


# ---------------------------------------------------------------- LED-4
# Done and unread: the task the reading video is shot on. Two attempts, a
# gate that refused the first, every pane with something in it.

REPORT = """## What I did

`Charge` posted to the processor on every call, so a webhook the processor
retried charged the order twice. It now keeps the orders it has posted and
returns early on the second call for the same one.

The key is the order and not the request id: the processor retries with a new
request id, so keying on that would have kept both charges.

## What I did not do

The map is per process. Two instances behind a load balancer will still
double-charge, and the durable fix is a unique index on `(order_id)` in the
charges table — which is a migration, and this task did not ask for one.
"""

led4 = [
    ev(320, "task.created", text="Charge() is not idempotent\n\nA retried webhook charges twice. The processor retries on any 5xx, and our handler posts every time."),
    ev(300, "task.started", attempt=1),
    ev(300, "phase.started", "implement", engine="claude", model="opus", attempt=1),
    ev(299, "phase.thought", "implement", text="Reading the handler first. Deliver calls Charge with no guard, and the processor's docs say it retries on any 5xx with a new request id — so the request id is not the key."),
    ev(298, "phase.tool_call", "implement", text='{"file_path":"webhook.go"}', tool="Read"),
    ev(297, "phase.tool_call", "implement", text='{"file_path":"ledger.go"}', tool="Read"),
    ev(296, "phase.tool_call", "implement", text='{"file_path":"ledger.go"}', tool="Edit"),
    ev(295, "phase.refused", "implement", text="git push origin main\nthe branch belongs to the operator or the runner, and nothing in a run may publish one", tool="Bash"),
    ev(294, "phase.tool_call", "implement", text="go test ./...", tool="Bash"),
    ev(293, "gate.failed", "implement", text="go test ./...", gate="tests", cause="TestARetriedWebhookChargesOnce is not there: the change has no test that would fail without it"),
    ev(292, "phase.retried", "implement", gate="tests", cost="0.31", attempt=2),
    ev(291, "task.started", attempt=2),
    ev(291, "phase.started", "implement", engine="claude", model="opus", attempt=2),
    ev(290, "phase.thought", "implement", text="The gate is right. Writing the test first this time: a charge, then the same order again, and the processor should see one post."),
    ev(289, "phase.tool_call", "implement", text='{"file_path":"ledger_test.go"}', tool="Edit"),
    ev(288, "phase.tool_call", "implement", text="go test ./...", tool="Bash"),
    ev(287, "gate.passed", "implement", text="go test ./...", gate="tests"),
    ev(286, "gate.passed", "implement", text="go build ./...", gate="build"),
    ev(285, "phase.finished", "implement", text=REPORT, cost="0.58", session="8f2c31",
       output_bytes="41920", attempt=2),
    ev(284, "phase.started", "review", engine="claude", model="sonnet", attempt=2),
    ev(280, "phase.finished", "review", text="The guard is in the right place and the test fails without it. Flagging the per-process scope, which the report already says.", cost="0.11", attempt=2),
    ev(279, "task.story", entry="POST /charge, from the processor's webhook",
       purpose="move the money for an order exactly once",
       symptom="a retried webhook charged the order twice",
       cause="Charge posted on every call; nothing said the order had been posted",
       fix="keep the orders already posted and return early on the second call"),
    ev(279, "task.delta",
       needs="callers must treat Charge as safe to call twice for the same order",
       guarantees="one post to the processor per order id, per process",
       assumes="the processor retries with a new request id, so the order id is the only stable key",
       instead="keying on the request id, which the processor changes on every retry\na unique index on (order_id), which is durable and is a migration this task did not ask for"),
    ev(278, "task.noted", text="Good catch on the per-process scope. Open a follow-up for the index.", attempt=2),
    ev(276, "task.dialogue", text="opened a session in the worktree and read the processor's retry docs\nthey do retry with a new request id", by="cli"),
    ev(275, "deliver.asked", verb="PR", by="supervisor"),
    ev(273, "deliver.answered", verb="PR", text="opened #482"),
    ev(272, "task.finished", text="one post per order, with the test that would have caught it", cost="0.69"),
]

# ---------------------------------------------------------------- LED-7
# Needs you: stopped at the review gate, waiting for a person.

led7 = [
    ev(140, "task.created", text="drop the legacy amount_cents column\n\nEverything reads amount_minor now. The old column is written and never read."),
    ev(120, "task.started", attempt=1),
    ev(120, "phase.started", "implement", engine="claude", model="opus", attempt=1),
    ev(112, "phase.tool_call", "implement", text='{"file_path":"migrations/0031_drop_amount_cents.sql"}', tool="Write"),
    ev(110, "gate.passed", "implement", text="go build ./...", gate="build"),
    ev(109, "phase.finished", "implement", text="Wrote the migration and took the column out of the model. It is a one-way change, so the review gate is where it should stop.", cost="0.24", attempt=1),
    ev(108, "phase.waiting", "review", cause="dropping a column is not reversible; somebody has to say yes"),
]

# ---------------------------------------------------------------- LED-9
# Needs you: the run broke, and the record says on what.

led9 = [
    ev(95, "task.created", text="rate-limit the webhook endpoint to 30/s per merchant"),
    ev(80, "task.started", attempt=1),
    ev(80, "phase.started", "implement", engine="codex", model="gpt-5.6-terra", attempt=1),
    ev(76, "phase.tool_call", "implement", text='{"file_path":"webhook.go"}', tool="Edit"),
    ev(74, "phase.tool_call", "implement", text="go test ./...", tool="Bash"),
    ev(73, "phase.failed", "implement", cause="the limiter needs a store per merchant and this service has none", exit="1", cost="0.18", attempt=1),
]

# ---------------------------------------------------------------- CHK-2
# Done and read: the band has something in it that nobody has to look at.

chk2 = [
    ev(600, "task.created", text="the pay button stays enabled while the request is out\n\nA double click pays twice."),
    ev(560, "task.started", attempt=1),
    ev(560, "phase.started", "implement", engine="opencode", model="claude-sonnet-5", attempt=1),
    ev(552, "gate.passed", "implement", text="npm test", gate="tests"),
    ev(551, "phase.finished", "implement", text="Disabled the button while the request is out and restored it in a finally.", cost="0.07", attempt=1),
    ev(550, "task.finished", text="one payment per click", cost="0.07"),
    ev(400, "task.read"),
]

# ---------------------------------------------------------------- to do
# Nothing but the line that wrote them down.

led11 = [ev(60, "task.created", text="the reconciliation job double-counts refunds\n\nA refund lands as its own row and is also subtracted from the charge.")]
chk3 = [ev(45, "task.created", text="read the card brand from the token, not the PAN\n\nThe PAN is not ours to keep and the token already carries the brand.")]

for task, events in [("LED-4", led4), ("LED-7", led7), ("LED-9", led9),
                     ("CHK-2", chk2), ("LED-11", led11), ("CHK-3", chk3)]:
    write(task, events)

# ---------------------------------------------------------------- knowledge
# What Orbit knows about this code, in the store's own file format: a header
# of key: value lines and the sentence as the body, which is the shape a
# person reads in a pull request.

def fact(path, scope, source, phrase, ref=None, at=None, check=None,
         stops=False, off=False, used=0, p=None, symbol=None, lang=None):
    head = ["---", "scope: " + scope, "source: " + source]
    if ref:    head.append("ref: " + ref)
    if p:      head.append("path: " + p)
    if symbol: head.append("symbol: " + symbol)
    if lang:   head.append("lang: " + lang)
    head.append("at: " + (at or ago(400)).replace("Z", "Z"))
    if stops:  head.append("action: stop")
    if check:  head.append("check: " + check)
    if off:    head.append("off: true")
    if used:   head.append("used: " + str(used))
    head.append("---")
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("\n".join(head) + "\n\n" + phrase + "\n")


STATE = D / "home" / "knowledge"
LEDGER = CODE / "ledger" / ".orbit" / "knowledge"

fact(STATE / "general" / "prs-are-written-in-english.md", "general", "human",
     "Pull requests and commit messages are written in English, however the task was written.", used=14)
fact(STATE / "lang" / "go" / "never-discard-an-error.md", "lang", "human",
     "In Go, never discard an error with `_`. If it cannot be handled, wrap it and return it.",
     lang="go", used=31)
fact(LEDGER / "the-ledger-only-appends.md", "repo", "code",
     "The ledger only appends. `Write` inserts and nothing updates a posted row, so a correction is a new row and never an edit.",
     used=9)
fact(LEDGER / "migrations-stop-for-a-person.md", "repo", "human",
     "Anything that drops a column or writes a migration stops for a person, autopilot or not.",
     stops=True, check="git diff --name-only | grep -q '^migrations/'", used=2)
fact(LEDGER / "money" / "round-half-to-even.md", "dir", "record",
     "Everything under money/ is in minor units and rounds half to even. A float in this directory is a bug.",
     p="money", ref="LED-2", used=6)
fact(LEDGER / "charge-must-stay-idempotent.md", "symbol", "record",
     "Charge() is called from the webhook, which the processor retries with a new request id. It must stay idempotent on the order id.",
     p="ledger.go", symbol="Charge", ref="LED-4", used=1)
fact(LEDGER / "the-test-suite-needs-postgres.md", "repo", "record",
     "The test suite needs a Postgres on :5432. Without it every test fails on connection refused, which is not the change's fault.",
     ref="LED-9", off=True, used=3)

print("wrote 7 facts")
