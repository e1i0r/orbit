#!/bin/bash
# Seeds a demo board for the landing recordings: two repositories, tasks in
# every band, and a record behind each one. Nothing here calls an engine —
# the events are written as JSONL and the migration that runs before every
# command copies them into the record, which is the same door a state root
# written by an older Orbit comes in through.
set -euo pipefail

# Taken before anything cds anywhere: the script walks into each repository it
# builds, and a relative $0 stops meaning this file the moment it does.
HERE="$(cd "$(dirname "$0")" && pwd)"

# The state root is disposable and lives wherever the caller says. The
# repositories are not: they are in shot, and a path with a job id in it reads
# as somebody's scratch directory rather than as somebody's code.
D="${1:?usage: seed.sh <state dir>}"
CODE="${DEMO_CODE:-$HOME/code}"
ORBIT="${ORBIT_BIN:?set ORBIT_BIN}"
export ORBIT_HOME="$D/home"

# The two repositories are wiped and rebuilt on every take, because the board
# state decides where the arrows land. A directory that is not one of ours is
# not ours to delete: the marker is what says a previous take made it.
for name in ledger checkout; do
  if [ -e "$CODE/$name" ] && [ ! -e "$CODE/$name/.orbit-demo" ]; then
    echo "$CODE/$name is not a demo repository this script made; move it or set DEMO_CODE" >&2
    exit 1
  fi
done

rm -rf "$D" "$CODE/ledger" "$CODE/checkout"
mkdir -p "$D/home" "$CODE"

git_init() {
  local name=$1
  mkdir -p "$CODE/$name"
  cd "$CODE/$name"
  git init -q -b main
  git config user.email demo@orbit
  git config user.name "Demo"
  : > .orbit-demo
}

# ---- the ledger: a payments service
git_init ledger
cat > ledger.go <<'EOF'
package ledger

import "errors"

// Charge moves money once, for one order.
func Charge(orderID string, cents int) error {
	if cents <= 0 {
		return errors.New("nothing to charge")
	}

	return post(orderID, cents)
}
EOF
cat > webhook.go <<'EOF'
package ledger

// Deliver is the endpoint the processor calls back on.
func Deliver(orderID string, cents int) error { return Charge(orderID, cents) }
EOF
cat > invoice.go <<'EOF'
package ledger

// Issue writes the invoice for a charge that went through.
func Issue(orderID string, cents int) error { return nil }
EOF
# The reconciliation job the real-run take is shot against. The bug in it is
# real and small: a refund is its own row and is also subtracted from the
# charge, so it lands on the total twice. A task the agent cannot actually do
# makes a recording of an agent explaining that there is nothing to do.
cat > reconcile.go <<'EOF'
package ledger

// A Row is one line of the day's ledger.
type Row struct {
	OrderID string
	Cents   int
	Refund  int
}

// Total is what the day came to, for the reconciliation job.
func Total(rows []Row) int {
	var out int

	for _, r := range rows {
		out += r.Cents - r.Refund
		if r.Refund > 0 {
			out -= r.Refund
		}
	}

	return out
}
EOF
cat > ledger_test.go <<'EOF'
package ledger

import "testing"

func TestChargeRejectsNegativeAmounts(t *testing.T) {}
func TestAnInvoiceFollowsEveryCharge(t *testing.T)  {}
EOF
printf 'module ledger\n\ngo 1.26\n' > go.mod
printf 'func post(orderID string, cents int) error { return nil }\n' >> ledger.go
git add -A && git commit -qm "the ledger, the webhook and the invoice"

# A history the impact reading can be read off: invoice.go has come along with
# ledger.go on almost every commit, which is the fact the pane reports.
for i in 1 2 3 4 5 6 7; do
  printf '\n// rev %d\n' "$i" >> ledger.go
  printf '\n// rev %d\n' "$i" >> invoice.go
  git add -A && git commit -qm "charge and invoice, rev $i"
done
printf '\n// webhook only\n' >> webhook.go
git add -A && git commit -qm "the webhook alone"

# ---- the checkout front end, so the board is over more than one repository
git_init checkout
printf 'export function pay(order){ return fetch("/charge", {method:"POST"}); }\n' > pay.js
printf '{"name":"checkout"}\n' > package.json
git add -A && git commit -qm "the checkout page"

cd "$D"

# ---- the tasks
new() { "$ORBIT" new -repo "$CODE/$1" -id "$2" "$3" >/dev/null; }

new ledger   LED-4  "Charge() is not idempotent: a retried webhook charges twice"
new ledger   LED-7  "drop the legacy amount_cents column"
new ledger   LED-9  "rate-limit the webhook endpoint to 30/s per merchant"
new ledger   LED-11 "the reconciliation job double-counts refunds"
new checkout CHK-2  "the pay button stays enabled while the request is out"
new checkout CHK-3  "read the card brand from the token, not the PAN"

# ---- the record behind them
python3 "$HERE/seed_events.py" "$D" "$CODE"

# ---- a worktree for the task that is read on camera, so the diff pane has a
# diff to draw and the impact reading has a change to weigh.
# The key is a hash of the repository's own path, and there are two
# repositories here: picking the first directory picked the wrong one, and the
# impact pane answered "the history could not be read" against a worktree
# nobody had made.
KEY=""
for d in "$D"/home/repos/*/; do
  if grep -qx "path: $CODE/ledger" "$d/repo" 2>/dev/null; then
    KEY=$(basename "$d")
  fi
done
[ -n "$KEY" ] || { echo "no state root entry for $CODE/ledger" >&2; exit 1; }
WT="$D/home/worktrees/$KEY/LED-4"
mkdir -p "$(dirname "$WT")"
cd "$CODE/ledger"
git worktree add -q -b orbit/LED-4 "$WT" >/dev/null
cd "$WT"
python3 - "$WT" <<'PY'
import sys, pathlib
wt = pathlib.Path(sys.argv[1])
p = wt / "ledger.go"
s = p.read_text()
s = s.replace('''import "errors"''', '''import (
	"errors"
	"sync"
)

// charged is every order this process has already posted, so a webhook the
// processor sends twice is posted once. The key is the order and not the
// request: the processor retries with a new request id.
var charged sync.Map''')
s = s.replace('''	return post(orderID, cents)''', '''	if _, seen := charged.LoadOrStore(orderID, true); seen {
		return nil
	}

	return post(orderID, cents)''')
p.write_text(s)
(wt / "ledger_test.go").write_text((wt / "ledger_test.go").read_text().replace(
    "func TestAnInvoiceFollowsEveryCharge(t *testing.T)  {}",
    "func TestAnInvoiceFollowsEveryCharge(t *testing.T)  {}\nfunc TestARetriedWebhookChargesOnce(t *testing.T)   {}"))
PY

# ---- the supervisor thread, written through the door that writes it: -by
# names the author and no engine is called.
say() { "$ORBIT" supervisor -by "$1" "$2" >/dev/null; }
say operator   "PRs are written in English, however the task was written."
say supervisor "Noted. I will write them in English."
say operator   "Anything that drops a column or writes a migration stops for me, autopilot or not."
say supervisor "Understood. I have written that down against the ledger, so a run reads it before it starts."
say supervisor "LED-4 came back with one post per order and a test that fails without the guard. The report says the map is per process, which is true: two instances still double-charge. I opened #482 and left a note asking for a follow-up on the unique index."
say operator   "Good. Do not touch the charges table without me."

# ---- the dials the recordings are shot on
"$ORBIT" settings engine claude >/dev/null
"$ORBIT" settings model opus >/dev/null
# Written down rather than left to the default, so a take is the same take
# whatever a later build decides the default is.
"$ORBIT" settings theme frauddi >/dev/null

# ---- and a home with nothing in it, for the getting-started take: the state
# root does not exist yet, and HOME is this directory so `orbit mcp install`
# writes into it rather than into the machine's own clients.
# Short, because it is in shot: `orbit mcp install` prints the path of every
# client it registered, and a state root nested four directories inside a
# checkout makes those lines wrap.
FRESH="${DEMO_FRESH:-$HOME/.orbit-demo}"
rm -rf "$FRESH"
mkdir -p "$FRESH/code"
cp -R "$CODE/ledger" "$FRESH/code/ledger"
cp -R "$CODE/checkout" "$FRESH/code/checkout"

cd "$D"
echo "seeded $D"
