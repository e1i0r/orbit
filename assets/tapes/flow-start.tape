# Getting started: register the MCP server, point the cockpit at the code,
# write the first task.
#
# It is shot under a HOME of its own, so `orbit mcp install` writes into a
# scratch directory and not into the machine's own clients — the seeding
# script sets ORBIT_DEMO_FRESH to one with two repositories under code/ and
# no state root at all.

Output "assets/flow-start.gif"
Output "site/flow-start.mp4"
Output "site/flow-start.webm"

Set Shell bash
Set FontSize 22
Set Width 1880
Set Height 700
Set Padding 20
Set Framerate 24

Hide
Type `export HOME="$ORBIT_DEMO_FRESH" ORBIT_HOME="$ORBIT_DEMO_FRESH/.orbit" PATH="$ORBIT_DEMO_BIN:$PATH"` Enter
Type `cd "$HOME"` Enter
Type "clear" Enter
Show

Type "orbit mcp install" Sleep 800ms Enter
Sleep 4s

Type "orbit new -repo ~/code/ledger -id LED-3 " Sleep 500ms
Type `"the refund endpoint does not check the idempotency key"` Sleep 500ms Enter
Sleep 3s

Type "orbit top ~/code" Sleep 600ms Enter
Sleep 5s

# The task is on the board, in To Do, with nothing spent on it yet.
Down
Sleep 2s
Type "m"
Sleep 3500ms
Escape
Sleep 1500ms
Type "q"
Sleep 1s
