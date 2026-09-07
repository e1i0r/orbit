# What you can do: the menu on a row, and the menu inside a task — including
# the verbs it will not let you press, with the reason.

Output "assets/flow-menus.gif"
Output "site/flow-menus.mp4"
Output "site/flow-menus.webm"

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

# On a task that is waiting for a person: everything that can be done to it,
# and everything that cannot.
Type "m"
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
Sleep 1500ms

# Inside a task the same key carries both blocks: the panes, and the verbs.
Type "G"
Sleep 600ms
Enter
Sleep 800ms
Type "G"
Sleep 600ms
Enter
Sleep 2500ms

Type "m"
Sleep 4s
Down
Sleep 600ms
Down
Sleep 600ms
Down
Sleep 600ms
Down
Sleep 600ms
Down
Sleep 2500ms
Escape
Sleep 1200ms

# And ? is the whole sheet.
Type "?"
Sleep 4s
Escape
Sleep 800ms
Escape
Sleep 800ms
Type "q"
Sleep 1s
