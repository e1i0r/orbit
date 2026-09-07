# A run, end to end — take A: the dialog, the flow, and the row going live.
#
# The two takes are cut together with ffmpeg (see real.md). A phase takes
# minutes and the section is forty seconds, so what is shot is the two ends of
# it: the decision that starts a run, and the gate it stops at.

Output "site/flow-run-a.mp4"

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
Sleep 4s

# The task nobody has run yet: G is the last row, which is the Done band's
# own head; up from it is To Do, and opening that band puts its two rows
# under the cursor.
Type "G"
Sleep 700ms
Up
Sleep 700ms
Enter
Sleep 900ms
Down
Sleep 600ms
Down
Sleep 1500ms

# n is the dialog that decides what the run will be: the flow, and the dials.
Type "n"
Sleep 4s
Type "f"
Sleep 1800ms
Type "f"
Sleep 1800ms
Type "f"
Sleep 2s
Enter
Sleep 8s

# It is in Running now, and the row says which phase it is in.
Sleep 12s
Type "q"
Sleep 1s
