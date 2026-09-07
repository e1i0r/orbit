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
Sleep 4s

# A task that has actually run, so there is a worktree to open the session
# in: the shot is the CLI standing in the task's own checkout.
Type "G"
Sleep 700ms
Up
Sleep 700ms
Enter
Sleep 900ms
Down
Sleep 600ms
Enter
Sleep 4s

# c hands the terminal over. What opens is your own CLI, in this task's
# worktree, carrying the task's context and Orbit's MCP server.
Type "c"
Sleep 25s

# The engine asks a person before it touches a directory, which is the point
# of the key and is not something a tape can answer for them.
Down
Sleep 1500ms
Enter
Sleep 15s

Type "which orbit task am I in, and what is in this worktree?" Sleep 1s Enter
Sleep 75s

# Quit it and you land back on the board, with the session written down.
Type "/exit" Enter
Sleep 10s

Sleep 4s
Type "q"
Sleep 2s
