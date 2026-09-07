# What Orbit knows: the facts that reach a run before it works.

Output "assets/flow-knowledge.gif"
Output "site/flow-knowledge.mp4"
Output "site/flow-knowledge.webm"

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

# K is what Orbit knows. The count is in the header before it is opened.
Type "K"
Sleep 3s

Down
Sleep 900ms
Down
Sleep 900ms
Down
Sleep 1500ms
Down
Sleep 2s
Down
Sleep 2s
Down
Sleep 2500ms

Escape
Sleep 1500ms
Type "q"
Sleep 1s
