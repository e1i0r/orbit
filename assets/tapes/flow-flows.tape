# Flows you write yourself: the list of them, what each phase is given, and
# the designer that writes a new one.
#
# Seeded — this is all screen, and the demo board carries three flows of its
# own beside the five that ship, so the list reads like a machine somebody
# actually works on.

Output "assets/flow-flows.gif"
Output "site/flow-flows.mp4"
Output "site/flow-flows.webm"

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

# F is every flow this machine can run: the five that ship, and the ones
# written here. Each phase says what it runs on and whether it stops.
Type "F"
Sleep 5s

Down
Sleep 1200ms
Down
Sleep 1200ms
Down
Sleep 2s
Down
Sleep 2s
Down
Sleep 2500ms

# One of them opened: its phases, their engines, their gates.
Enter
Sleep 5s
Down
Sleep 900ms
Down
Sleep 2500ms
Escape
Sleep 1500ms

# And n writes a new one. A flow is a list of phases, and the phase names are
# yours — a release flow is as legitimate as a coding one.
Type "n"
Sleep 5s

Escape
Sleep 1500ms
Escape
Sleep 1s
Type "q"
Sleep 1s
