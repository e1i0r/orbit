package verb

// What Orbit has been told, and writing more of it down.
//
// Reading everything known about the code, and learning one more true
// thing. The parent keeps its own body: `orbit knowledge` reads.
import "github.com/e1i0r/orbit/internal/words"

// knowledgeFamily is the family, parent first.
func knowledgeFamily() []Verb {
	return []Verb{
		{
			Name: "knowledge", Reads: true, About: func(p *words.Printer) string {
				return p.T("verb.knowledge", "everything Orbit has been told about this code")
			},
		},
		{
			Under: "knowledge",
			Name:  "learn",
			About: func(p *words.Printer) string {
				return p.T("verb.learn", "write down something true about this code")
			},
			Takes: []Field{
				{Name: "text", Kind: Words, Needed: true, About: func(p *words.Printer) string {
					return p.T("verb.learn.text", "the fact, in a sentence")
				}},
				{Name: "repo", Kind: Named, About: func(p *words.Printer) string {
					return p.T("verb.learn.repo", "which checkout it is about; the default is the one you are in")
				}},
			},
		},
	}
}
