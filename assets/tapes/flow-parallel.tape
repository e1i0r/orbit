# Many tasks at once: three runs, three worktrees, one window.
#
# Real runs, started from the terminal so that all three are already going
# when the cockpit opens — the shot is the board with three live rows on it,
# not the three keystrokes that started them.

Output "assets/flow-parallel.gif"
Output "site/flow-parallel.mp4"
Output "site/flow-parallel.webm"

Set Shell bash
Set FontSize 22
Set Width 1880
Set Height 700
Set Padding 20
Set Framerate 24

Hide
Type `export ORBIT_HOME="$ORBIT_DEMO_HOME" PATH="$ORBIT_DEMO_BIN:$PATH"` Enter
Type "clear" Enter
Show

Type "orbit run -repo ~/code/ledger LED-11 &" Sleep 400ms Enter
Sleep 2s
Type "orbit run -repo ~/code/checkout CHK-3 &" Sleep 400ms Enter
Sleep 2s
Type "orbit run -repo ~/code/ledger LED-9 &" Sleep 400ms Enter
Sleep 3s

Type "clear" Enter
Type "orbit top ~/code" Enter
Sleep 10s

# Three rows in Running, each in its own phase, each in its own worktree, and
# the header counting them.
Sleep 25s

# One of them opened, while the other two carry on behind it.
Down
Sleep 1500ms
Enter
Sleep 6s
Type "2"
Sleep 8s
Escape
Sleep 8s
Type "q"
Sleep 1s

# The worktrees they are running in: one checkout each, on a branch of its own.
Type "clear" Enter
Type `git -C ~/code/ledger worktree list` Sleep 500ms Enter
Sleep 5s
