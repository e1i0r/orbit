# Reading what it did: the story, the report, the diff, and what the change
# reaches that the diff cannot show.

Output "assets/flow-reading.gif"
Output "site/flow-reading.mp4"
Output "site/flow-reading.webm"

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

# G is the last row, which is the Done band's own head; opening it opens the
# band, and G again is the newest task in it.
Type "G"
Sleep 700ms
Enter
Sleep 900ms
Type "G"
Sleep 700ms
Enter
Sleep 4s

# The story: how this prompt became this diff, each field the reason for the
# one under it.
Down
Sleep 600ms
Down
Sleep 600ms
Down
Sleep 2s

# 7 is what the engine wrote about it.
Type "7"
Sleep 4s
Down
Sleep 600ms
Down
Sleep 600ms
Down
Sleep 2s

# 0 is what it actually wrote, a card per file with the reason beside it.
Type "0"
Sleep 4s
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 2500ms

# i is the one nobody else answers: what usually comes along with these files
# and did not this time.
Type "i"
Sleep 4500ms
Down
Sleep 700ms
Down
Sleep 700ms
Down
Sleep 700ms
Down
Sleep 2500ms

Escape
Sleep 1s
Type "q"
Sleep 1s
