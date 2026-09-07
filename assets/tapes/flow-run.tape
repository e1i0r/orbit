# A run, end to end: a real one. The engine is whatever the dial is set to,
# and the phases are the flow's own.
#
# This is one of the three that cannot be seeded: what it shows is movement.
# The state root it runs against is thrown away afterwards, and the task is
# small on purpose — the video is about the phases and the gate, not about
# the change.

Output "assets/flow-run.gif"
Output "site/flow-run.mp4"
Output "site/flow-run.webm"

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

Type "orbit top ~/code" Enter
Sleep 3s

# LED-11 has never run. n opens the dialog that decides what the run will be.
Type "G"
Sleep 600ms
Type "G"
Sleep 600ms
Up
Sleep 800ms
Type "n"
Sleep 3s

# The flow it runs under, and the dials it runs on.
Type "f"
Sleep 1500ms
Type "f"
Sleep 1500ms
Enter
Sleep 4s

# It is in Running now, and the phase it is in is on the row.
Sleep 30s

# The flow tree, moving.
Enter
Sleep 2s
Type "2"
Sleep 25s

# And what it is thinking while it does it.
Type "w"
Sleep 20s
Type "6"
Sleep 20s

Escape
Sleep 2s
Type "q"
Sleep 1s
