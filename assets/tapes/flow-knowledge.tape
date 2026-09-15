# What Orbit knows: the loop a rule goes round, from a sentence nobody has
# answered to a decision taken with the evidence in front of you.

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
Type "cd ~/code/ledger" Enter
Type "clear" Enter
Show

# What is waiting for an answer: the sentences nobody has decided about, and
# the rule that was sent back to be looked at.
Type "orbit rules" Enter
Sleep 3s

# Keeping one, narrowed to where it is true, and with the command that makes
# it refuse work rather than merely say something.
Type `orbit rules keep -in migrations -check "make migrate-check" 1` Enter
Sleep 3s

Type "clear" Enter
Sleep 500ms

# The rule that has been all the way round: what it stopped, how often you got
# past it, and what you paused it for.
Type "orbit rules review -rule d42b7f60" Enter
Sleep 4s

Type "clear" Enter
Sleep 500ms

# And the decision that was almost always the one wanted: not off — narrower.
Type "orbit rules correct -rule d42b7f60 -in migrations" Enter
Sleep 3s

Type "clear" Enter
Sleep 500ms

# The screen the same rules are read on, walked by how far each one reaches.
Type "orbit top ~/code" Enter
Sleep 4s

Type "K"
Sleep 3s

Down
Sleep 900ms
Down
Sleep 1500ms
Down
Sleep 2s
Down
Sleep 2s

Escape
Sleep 1200ms
Type "q"
Sleep 1s
