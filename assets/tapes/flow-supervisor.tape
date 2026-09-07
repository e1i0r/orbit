# The supervisor: the thread you steer it in, and what it learns from it.

Output "assets/flow-supervisor.gif"
Output "site/flow-supervisor.mp4"
Output "site/flow-supervisor.webm"

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

Type "S"
Sleep 4s

# It persists: this is a conversation from last week, still here.
Down
Sleep 1s
Down
Sleep 1s
Down
Sleep 2500ms

# And what you say in it is what the runs carry.
Type "the charges table is not to be touched without me"
Sleep 2500ms
Escape
Sleep 1500ms
Escape
Sleep 1s
Type "q"
Sleep 1s
