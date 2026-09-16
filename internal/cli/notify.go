package cli

// Interrupting the person at the machine.
//
// A chat reaches a phone; this reaches the desktop somebody is already
// sitting at, which is where they are most of the time. Both carry the same
// five endings, written once in internal/chat, because one switch and two
// wordings is two things to keep in step.
//
// It shells out to whatever the desktop has rather than linking a library.
// A notification is one line of text and a title, every desktop has a
// command that shows one, and a dependency that pulls in a window system to
// say six words is a dependency with nothing to show for itself.

import (
	"context"
	"os/exec"
	"strings"

	"github.com/e1i0r/orbit/internal/logger"
)

// theDesktop is the machine somebody is sitting at, as a place to be told
// things.
//
// It has no Listen: nothing arrives from a notification, and a desktop is
// not a conversation. That is why it is not a chat.Channel — it is handed to
// the desk as somewhere to say things and nothing else.
type theDesktop struct{}

// say shows one notification, and gives up rather than failing.
//
// A desktop with no command to show one is the ordinary case on a server,
// and a chat that refused to run because nobody was sitting at the machine
// would be refusing exactly where it is most useful.
func (theDesktop) say(ctx context.Context, title, text string) {
	name, args := notifier(title, plainly(text))
	if name == "" {
		return
	}

	if err := exec.CommandContext(ctx, name, args...).Run(); err != nil {
		logger.Info("cli/notify", "the desktop did not show a notification: %v", err)
	}
}

// plainly is the message with the chat's markup taken out and its command
// line dropped.
//
// A notification is one line somebody reads on the way past. The command
// that answers it is worth carrying in a chat, where it can be tapped, and
// is noise on a desktop where it cannot.
func plainly(text string) string {
	said, _, _ := strings.Cut(text, "\n")

	return strings.NewReplacer("\x02", "", "\x03", "", "\x04", "", "\x05", "").Replace(said)
}
