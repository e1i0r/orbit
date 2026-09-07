package supervisor

// The conversations of the supervisor thread: which one is open, what there
// is to open, and starting or removing one.
//
// The thread was one list with no ends — every line anybody had ever written
// to the supervisor, growing. Cutting it into conversations is safe because
// nothing permanent lives in them: the moment something was worth keeping,
// /rule or /aware sent it to what Orbit knows. A conversation is meant to be
// thrown away.
//
// The rule about which lines belong where is the record's: what reaches this
// screen is already folded, with the removed conversations gone and the
// retractions marked. What is here is presentation and the two gestures.

import (
	"strings"
	"time"
	"unicode"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

// conversation is one of them, as the list shows it.
type conversation struct {
	id    string
	title string
	last  time.Time
	turns int
}

// linesIn is one conversation's lines.
func linesIn(all []view.SupervisorLine, id string) []view.SupervisorLine {
	out := make([]view.SupervisorLine, 0, len(all))

	for _, l := range all {
		if l.Conversation == id {
			out = append(out, l)
		}
	}

	return out
}

// conversationsOf is what there is to open, most recently spoken in first.
//
// A line that was taken back neither titles a conversation nor counts in it:
// the reader decided it was not said, and a list that titled one with it
// would put it back on the screen it was taken off.
func conversationsOf(all []view.SupervisorLine) []conversation {
	var (
		out   []conversation
		where = map[string]int{}
	)

	for _, l := range all {
		if l.Retracted || strings.TrimSpace(l.Text) == "" {
			continue
		}

		at, held := where[l.Conversation]
		if !held {
			at = len(out)
			where[l.Conversation] = at
			out = append(out, conversation{id: l.Conversation, title: convTitle(l.Text)})
		}

		out[at].last = l.At
		out[at].turns++
	}

	// Newest last spoken in first, and a stable order between two that were
	// spoken in at the same moment.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].last.After(out[j-1].last); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}

	return out
}

// titleCut is how much of the first sentence a title carries: enough to
// recognise it by, short enough that the list stays a list.
const titleCut = 60

// convTitle is the first line of what was said, cut to fit a row.
func convTitle(text string) string {
	line := strings.TrimSpace(text)
	if cut, _, held := strings.Cut(line, "\n"); held {
		line = strings.TrimSpace(cut)
	}

	runes := []rune(line)
	if len(runes) <= titleCut {
		return line
	}

	return strings.TrimSpace(string(runes[:titleCut])) + "…"
}

// openConversation reads one, and puts the thread at its end.
func (s State) openConversation(id string, e Env) State {
	s.conversation = id
	s.list = false
	s.picking = false
	s.follow = true
	s.offset = 0

	return s.Sync(e)
}

// startConversation opens an empty one.
//
// It exists the moment something is said in it and not before: the id is
// carried by the first line written, so a reader who changes their mind
// leaves nothing behind.
func (s State) startConversation(e Env) (State, Out) {
	if e.NewID == nil {
		return s, said(e.Words.T("supervisor.no_new", "this build cannot start a conversation"))
	}

	next := s.openConversation(e.NewID(), e)
	next.input = ""

	return next, said(e.Words.T("supervisor.started",
		"new conversation — it takes its name from what you say first"))
}

// removeConversation takes the one under the cursor off the list.
func (s State) removeConversation(e Env) (State, Out) {
	convs := conversationsOf(s.all)
	if e.Forget == nil || s.listSel >= len(convs) {
		return s, Out{}
	}

	gone := convs[s.listSel]
	if err := e.Forget(gone.id); err != nil {
		return s, said(err.Error())
	}

	next := s.Sync(e)
	next.listSel = min(next.listSel, max(len(conversationsOf(next.all))-1, 0))

	// The one that was open is gone with it, so the screen lands on
	// whatever is left rather than on an empty thread it cannot explain.
	if gone.id == next.conversation {
		next = next.openLatest(e)
		next.list = true
	}

	return next, said(e.Words.T("supervisor.removed",
		"removed from the list — every line of it is still in the record"))
}

// openLatest opens the conversation last spoken in, and a new one when there
// is nothing to carry on.
func (s State) openLatest(e Env) State {
	if convs := conversationsOf(s.all); len(convs) > 0 {
		return s.openConversation(convs[0].id, e)
	}

	if e.NewID != nil {
		return s.openConversation(e.NewID(), e)
	}

	return s.openConversation("", e)
}

// openConversationList puts the list up, on the conversation being read.
func (s State) openConversationList() State {
	s.list = true
	s.picking = false
	s.listSel = 0

	for i, c := range conversationsOf(s.all) {
		if c.id == s.conversation {
			s.listSel = i
			break
		}
	}

	return s
}

// conversationKey is every key while the list is up.
func (s State) conversationKey(msg tea.KeyPressMsg, e Env) (State, Out) {
	convs := conversationsOf(s.all)

	switch {
	case msg.Code == tea.KeyEscape || key.Matches(msg, e.Keys.Back):
		s.list = false
		return s, Out{}
	case msg.Code == tea.KeyUp:
		s.listSel = max(s.listSel-1, 0)
		return s, Out{}
	case msg.Code == tea.KeyDown:
		s.listSel = min(s.listSel+1, max(len(convs)-1, 0))
		return s, Out{}
	case msg.Code == tea.KeyEnter || key.Matches(msg, e.Keys.Open):
		if s.listSel < len(convs) {
			return s.openConversation(convs[s.listSel].id, e), Out{}
		}

		return s, Out{}
	case msg.Code == 'd' || msg.Code == 'D':
		return s.removeConversation(e)
	case (msg.Code == 'n' || msg.Code == 'N') && msg.Mod&tea.ModCtrl != 0:
		return s.startConversation(e)
	}

	return s, Out{}
}

// clearedLine takes the gesture out of the line it was typed into, which
// every other gesture does through the thread being redrawn.
func (s State) clearedLine() State {
	s.input = ""

	return s
}

// ctrlLetter is whether a key is control and this letter, in the shapes a
// terminal may deliver it in: the letter with the modifier set, either case,
// or the control code itself — 0x0C for ^L — which is what a terminal
// speaking no modifier protocol sends and where the modifier is not set at
// all.
func ctrlLetter(msg tea.KeyPressMsg, letter rune) bool {
	code := msg.Code

	if code == letter-'a'+1 {
		return true
	}

	if msg.Mod&tea.ModCtrl == 0 {
		return false
	}

	return unicode.ToLower(code) == letter
}

// briefQuestion is what /brief asks, in the reader's own language.
//
// It is a question typed for somebody rather than a report Orbit writes: the
// answer is the supervisor's, at the length the question deserves, and it is
// asked in the language the screen is in because that is the language the
// answer has to come back in. Whatever was written after the word narrows
// it — "/brief the coverage" is still this question, about that.
func (s State) briefQuestion(narrower string, e Env) string {
	p := e.Words

	asked := p.T("supervisor.brief_ask",
		"What happened while I was away? Which tasks ran, how did each one end, "+
			"which checks passed and which failed? Say plainly what is verified and what is your reading.")

	if narrower = strings.TrimSpace(narrower); narrower != "" {
		asked += " " + p.T("supervisor.brief_about", "In particular: {about}", about("about", narrower))
	}

	return asked
}
