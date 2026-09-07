# A run, end to end — take B: the gate it stopped at, and letting it go.
#
# Shot after the run in take A has reached its review gate. What is on screen
# is a real record: the phases that ran, the gates they passed, and the one
# waiting for a person.

Output "site/flow-run-b.mp4"

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

# It is in Needs You, and the row says what it is waiting on.
Enter
Sleep 3s

# The tree: what ran, what it was given, and where it stopped.
Type "2"
Sleep 5s
Down
Sleep 800ms
Down
Sleep 800ms
Enter
Sleep 4s

# The gates it passed on the way.
Type "3"
Sleep 5s

# And r lets it through the one it is waiting at.
Escape
Sleep 1500ms
Type "r"
Sleep 5s
Type "q"
Sleep 1s
