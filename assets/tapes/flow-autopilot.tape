# Autopilot: the queue works itself, and the brake that keeps it from
# outrunning you.
#
# Real runs. What this shows is a board moving with nobody pressing anything,
# which is the one thing a seeded state root cannot show.

Output "assets/flow-autopilot.gif"
Output "site/flow-autopilot.mp4"
Output "site/flow-autopilot.webm"

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

# The switch, and the bar says what it is now.
Type "A"
Sleep 4s

# It takes the next To Do on its own.
Sleep 45s

# And the phases move without a person at the gates.
Sleep 45s

# The cap is the brake: unread finished tasks stop anything new from starting.
Hide
Type "" Enter
Show
Sleep 3s

Type "A"
Sleep 3s
Type "q"
Sleep 1s
