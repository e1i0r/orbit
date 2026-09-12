package verb

// The supervisor thread, and what is said in it.
//
// Saying something without opening the cockpit, reading it back, and
// taking a line back. The parent keeps its own body: `orbit supervisor`
// reads the thread.
import "github.com/e1i0r/orbit/internal/words"

// supervisorFamily is the family, parent first.
func supervisorFamily() []Verb {
	return []Verb{
		{
			Name: "supervisor", Reads: true,
			About: func(p *words.Printer) string {
				return p.T("verb.supervisor", "everything said in the supervisor thread")
			},
		},
		{
			Under: "supervisor",
			Name:  "say",
			About: func(p *words.Printer) string {
				return p.T("verb.say", "say something in the supervisor thread")
			},
			Takes: []Field{{Name: "text", Kind: Words, Needed: true, About: func(p *words.Printer) string {
				return p.T("verb.say.text", "what to say")
			}}},
		},
		{
			Under: "supervisor",
			Name:  "thread", Reads: true, About: func(p *words.Printer) string {
				return p.T("verb.thread", "everything said in the supervisor thread")
			},
		},
		{
			Under: "supervisor",
			Name:  "retract",
			About: func(p *words.Printer) string {
				return p.T("verb.retract", "take back a line of the supervisor thread")
			},
			Takes: []Field{{Name: "line", Kind: Named, Needed: true, About: func(p *words.Printer) string {
				return p.T("verb.retract.line", "which line, as `orbit thread` numbers them")
			}}},
		},
	}
}
