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
Sleep 5s

# Two tasks in To Do and nothing running. A is the switch, and the bar says
# what it did.
Type "A"
Sleep 5s

# It takes the next one on its own, and the row goes live.
Sleep 30s

# And the one after it, without a person at either gate.
Sleep 30s

# The brake is the unread cap, and the bar says when it is what stopped the
# queue rather than the queue being empty.
Sleep 20s

# Off again: it runs what you start and hands the rest back.
Type "A"
Sleep 4s
Type "q"
Sleep 1s
