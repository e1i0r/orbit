package supervisor

// What the supervisor's line offers while it is being typed.
//
// The four gestures are worth nothing if nobody knows they exist. A slash is
// where somebody finds out, and a mention is where they stop having to
// remember an id. Both are the same rule the key bar is drawn under: a thing
// the window can do and never shows is a thing nobody does.
//
// It offers only while a word is unfinished. A conversation is not a command
// line, and a list popping up over somebody's sentence is the window
// interrupting them.

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/spoken"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// A completion is one thing on offer: what it puts in the line, and what it
// does. The second is not decoration — "/rule" and "/aware" are two words
// for two powers, and which is which is the whole point of showing them.
type completion struct {
	Text string
	What string
}

// completions is what the line is offering right now, and nothing at all
// when the word being typed is an ordinary one.
func (s State) completions(e Env) []completion {
	typed := s.input
	if strings.ContainsAny(typed, " \n") || typed == "" {
		// Past the first word: whatever is being written is a sentence.
		return nil
	}

	switch {
	case strings.HasPrefix(typed, "/"):
		return matching(s.gestures(e), typed)
	case strings.HasPrefix(typed, spoken.AtWord):
		return matching(s.tasksOnOffer(e), typed)
	default:
		return nil
	}
}

// gestures is the four things a line can be, with what each one does.
func (s State) gestures(e Env) []completion {
	p := e.Words

	return []completion{
		{spoken.RuleWord, p.T("complete.rule", "a rule with a check: the gate sends the work back")},
		{spoken.AwareWord, p.T("complete.aware", "something to keep in mind: it reaches the prompt")},
		{spoken.RuleWord + " " + spoken.GeneralFlag, p.T("complete.rule_general", "a rule for every repository")},
		{spoken.AwareWord + " " + spoken.GeneralFlag, p.T("complete.aware_general", "for every repository")},
		{spoken.AwareWord + " " + spoken.LangFlag, p.T("complete.aware_lang", "for one language: {flag} go", about("flag", spoken.LangFlag))},
		{spoken.BriefWord, p.T("complete.brief", "what happened while you were away, from the record")},
		{spoken.ChatsWord, p.T("complete.chats", "the conversations, by what started each one")},
		{spoken.NewWord, p.T("complete.new", "start a conversation of its own")},
	}
}

// tasksOnOffer is every task on the board, by id, with its title beside it —
// an id alone is not a thing anybody remembers.
func (s State) tasksOnOffer(e Env) []completion {
	out := make([]completion, 0, len(e.Tasks))
	for _, t := range e.Tasks {
		out = append(out, completion{spoken.AtWord + t.ID, t.Title})
	}

	return out
}

// matching keeps what starts with the word being typed, case aside.
//
// A prefix and not a substring: somebody typing "/aw" is finishing a word
// from its front, and a list that also offered everything containing "aw"
// would put the answer somewhere down the middle.
func matching(all []completion, typed string) []completion {
	kept := make([]completion, 0, len(all))

	for _, c := range all {
		if strings.HasPrefix(strings.ToLower(c.Text), strings.ToLower(typed)) {
			kept = append(kept, c)
		}
	}

	if len(kept) == 0 {
		return nil
	}

	return kept
}

// takeCompletion puts the chosen offer in the line, with the space that
// follows it: what comes next is the sentence, and it should not arrive
// stuck to the gesture.
func (s State) takeCompletion(e Env) State {
	offers := s.completions(e)
	if len(offers) == 0 {
		return s
	}

	s.input = offers[min(s.pick, len(offers)-1)].Text + " "
	s.pick = 0

	return s
}

// moveCompletion walks the list, and stops at either end rather than
// wrapping: a list that comes back round has no bottom, and somebody holding
// a key down never finds out they reached it.
func (s State) moveCompletion(d int, e Env) State {
	s.pick = min(max(s.pick+d, 0), max(len(s.completions(e))-1, 0))

	return s
}

// offering is whether a key belongs to the list rather than to the line.
// Every other key types, which is what keeps the list out of the way of
// somebody who is not looking at it.
func offering(msg tea.KeyPressMsg) bool {
	switch msg.Code {
	case tea.KeyTab, tea.KeyEnter, tea.KeyUp, tea.KeyDown:
		return true
	default:
		return false
	}
}

// completionKey is the list's own keyboard.
func (s State) completionKey(msg tea.KeyPressMsg, e Env) State {
	switch msg.Code {
	case tea.KeyUp:
		return s.moveCompletion(-1, e)
	case tea.KeyDown:
		return s.moveCompletion(1, e)
	default:
		return s.takeCompletion(e)
	}
}

// completionRows is how many of the offers are drawn at once. Enough that the
// four gestures are all on screen together — which is where somebody finds
// out they exist — and few enough that a board of forty tasks does not take
// the conversation off the screen behind it.
const completionRows = 6

// drawCompletions is the list, above the line being typed.
//
// It is drawn where a reader is already looking: their eyes are on the
// cursor, and a list anywhere else is a list they have to find. The chosen
// row carries the mark, and what each offer does sits beside it in dim —
// the descriptions are the reason this exists, not decoration on it.
func (s State) drawCompletions(cw int, e Env) []string {
	offers := s.completions(e)
	if len(offers) == 0 {
		return nil
	}

	pick := min(s.pick, len(offers)-1)

	from := 0
	if pick >= completionRows {
		from = pick - completionRows + 1
	}

	to := min(from+completionRows, len(offers))
	rows := make([]string, 0, to-from+1)

	for i := from; i < to; i++ {
		rows = append(rows, offerRow(offers[i], i == pick, cw))
	}

	if len(offers) > to-from {
		rows = append(rows, theme.Paint(theme.Dim).Render(
			e.Words.T("complete.more", "{n} more", about("n", strconv.Itoa(len(offers)-(to-from))))))
	}

	return append(rows, "")
}

// offerRow is one offer: the mark, what it puts in the line, and what it
// does.
func offerRow(c completion, chosen bool, cw int) string {
	mark, text := "  ", theme.Paint(theme.Dim).Render(c.Text)
	if chosen {
		mark, text = theme.Paint(theme.Accent).Bold(true).Render("▸ "), theme.Paint(theme.Accent).Bold(true).Render(c.Text)
	}

	return cells.Fit(mark+text+"  "+theme.Paint(theme.Dim).Render(c.What), cw)
}
