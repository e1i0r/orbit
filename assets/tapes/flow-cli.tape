# The CLI, both ways: c hands your CLI the terminal in that task's worktree,
# and quitting hands it back with the session written down.
#
# The beat this tape cannot script is the engine's own approval prompt, which
# is a person's answer by design. Record generously and cut it in post:
#
#   ffmpeg -i take.mp4 -ss 0 -to 12 -c copy a.mp4
#   ffmpeg -i take.mp4 -ss 40 -c copy b.mp4
#   ffmpeg -f concat -i parts.txt -c copy site/flow-cli.mp4

Output "assets/flow-cli.gif"
Output "site/flow-cli.mp4"
Output "site/flow-cli.webm"

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

# The task that is waiting for a person is the one worth opening a session on.
Enter
Sleep 3s

# c hands the terminal over, in this task's worktree, with its context.
Type "c"
Sleep 12s

Type "pwd" Sleep 400ms Enter
Sleep 4s
Type "what is this task about, and what did the last run stop on?" Sleep 800ms Enter
Sleep 45s

# Quit it, and Orbit asks whether the session was worth writing down.
Type "/exit" Enter
Sleep 8s
Type "y"
Sleep 6s

Escape
Sleep 2s
Type "q"
Sleep 1s
