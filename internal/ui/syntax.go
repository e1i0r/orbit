package ui

import "strings"

// Colour inside a code well.
//
// A fenced block was one flat colour on a darker paper, which says "this is
// code" and nothing else. What a reader is looking for in a block somebody
// else's model wrote is the shape of it — where the strings are, which line
// is a comment, what the words the language reserved are — and that shape is
// carried by colour in every editor they have ever used.
//
// The arrangement is Monokai's, because that is the one every terminal reader
// recognises: the keyword loud, the string warm, the comment gone quiet. The
// hues are not Monokai's. They are whichever theme is up, asked for by role,
// so a block inside a nord pane is nord — a well that kept Monokai's pink
// would be the one thing on the screen from somebody else's window.
//
// This is a highlighter and not a parser. It reads one line at a time and
// knows nothing of the line above it, so a string opened on one line and
// closed on the next is two strings to it. That is the trade a fenced block
// in a pane is worth: a parser per language is a dependency per language.

// codePart is what a run of a line of code is, as far as colour cares.
type codePart int

const (
	codePlain   codePart = iota
	codeComment          // to the end of the line
	codeString           // between quotes, escapes respected
	codeNumber           // a literal, decimal or hex
	codeKeyword          // what the language reserved: control flow, declarations
	codeType             // what it is built out of: the types and the builtins
	codeConst            // the named literals: nil, true, null, undefined
)

// codeRole is the role a part is painted in, and whether it is painted at
// all: plain code is the well's own ink, which is the brightest thing on it
// and the right weight for the half of a line that is neither a keyword nor
// a string.
func codeRole(p codePart) (Role, bool) {
	switch p {
	case codeComment:
		return Dim, true
	case codeString:
		return Warn, true
	case codeNumber, codeConst:
		return Accent, true
	case codeKeyword:
		return Bad, true
	case codeType:
		return Live, true
	case codePlain:
		return Dim, false
	}

	return Dim, false
}

// codeFamily is the syntax a fence's language is read with. The name on a
// fence is whatever the model typed, so the aliases are the ones models type.
func codeFamily(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "go", "golang":
		return "go"
	case "sh", "bash", "zsh", "shell", "console", "terminal":
		return "shell"
	case "json", "jsonc", "jsonl", "yaml", "yml", "toml":
		return "data"
	case "py", "python":
		return "python"
	case "js", "javascript", "ts", "typescript", "tsx", "jsx":
		return "js"
	case "sql":
		return "sql"
	}

	// A fence with no language, or one nothing here knows: the quotes and the
	// numbers still read, and nothing is called a keyword that might not be.
	return ""
}

// codeVocabulary is what one family calls its own: what it reserved, what it
// is built out of, and the literals it gave names to.
//
// They are three sets and not one because they are read at three volumes.
// Monokai paints the keyword loud, the type cool and the constant apart, and
// a block that shouts every one of them equally is a block with no shape.
//
// None of them is the whole standard library. The words here are the ones a
// reader skims a block for; a list that also held every function a language
// ships would paint most of a line loud and say nothing by it.
type codeVocabulary struct {
	words  map[string]bool
	types  map[string]bool
	consts map[string]bool
}

var codeVocab = map[string]codeVocabulary{
	"go": {
		words: wordSet(`break case chan const continue default defer else fallthrough for func go
			goto if import interface map package range return select struct switch type var`),
		types: wordSet(`string int int8 int16 int32 int64 uint uint8 uint16 uint32 uint64 byte
			rune float32 float64 bool error any make new len cap append copy delete panic recover`),
		consts: wordSet(`nil true false iota`),
	},
	"shell": {
		words: wordSet(`if then else elif fi for while until do done case esac function return
			in select time coproc local export readonly declare unset shift source`),
		types:  wordSet(`echo cd set trap exit read printf`),
		consts: wordSet(`true false`),
	},
	"data": {consts: wordSet(`true false null`)},
	"python": {
		words: wordSet(`and as assert async await break class continue def del elif else except
			finally for from global if import in is lambda nonlocal not or pass raise
			return try while with yield`),
		types:  wordSet(`str int float bool list dict set tuple bytes print len range open`),
		consts: wordSet(`none true false self`),
	},
	"js": {
		words: wordSet(`async await break case catch class const continue debugger default delete
			do else export extends finally for function if import in instanceof let new
			return super switch this throw try typeof var void while with yield`),
		types:  wordSet(`string number boolean object symbol bigint interface type enum implements readonly`),
		consts: wordSet(`null true false undefined nan`),
	},
	"sql": {
		words: wordSet(`select from where group by order having join left right inner outer full
			on insert into values update set delete create table alter drop index view
			and or not as distinct limit offset union all case when then else end`),
		types:  wordSet(`int integer text varchar char boolean date timestamp numeric decimal`),
		consts: wordSet(`null true false`),
	},
}

// wordSet reads a block of words into the set a lexer asks.
func wordSet(words string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.Fields(words) {
		out[w] = true
	}

	return out
}

// codeWordPart is what a family calls one identifier, and whether it calls it
// anything at all.
func codeWordPart(family, word string) (codePart, bool) {
	vocab, known := codeVocab[family]
	if !known {
		return codePlain, false
	}

	lower := strings.ToLower(word)

	switch {
	case vocab.words[lower]:
		return codeKeyword, true
	case vocab.types[lower]:
		return codeType, true
	case vocab.consts[lower]:
		return codeConst, true
	}

	return codePlain, false
}

// codeComments is what opens a comment in each family. A family with none is
// a family where nothing on a line is dropped to the quiet colour.
var codeComments = map[string][]string{
	"go":     {"//"},
	"shell":  {"#"},
	"data":   {"#", "//"},
	"python": {"#"},
	"js":     {"//"},
	"sql":    {"--"},
}

// codeToken is one run of a line and what it is.
type codeToken struct {
	text string
	part codePart
}

// lexCode reads one line into the runs colour is decided on.
//
// It walks runes rather than bytes: a comment in a report is prose, and prose
// in these panes is not measured in bytes anywhere else either.
func lexCode(line, family string) []codeToken {
	var (
		out  []codeToken
		flat []rune
	)

	keep := func(part codePart, text string) {
		if len(flat) > 0 {
			out = append(out, codeToken{text: string(flat), part: codePlain})
			flat = flat[:0]
		}

		out = append(out, codeToken{text: text, part: part})
	}

	runes := []rune(line)

	for i := 0; i < len(runes); {
		rest := string(runes[i:])

		if open := commentAt(rest, family); open {
			keep(codeComment, rest)

			return out
		}

		if q := runes[i]; q == '"' || q == '\'' || q == '`' {
			text := quotedAt(runes[i:])
			keep(codeString, text)
			i += len([]rune(text))

			continue
		}

		// A digit is only ever reached at the start of a token: an
		// identifier is taken whole, and the digits inside one go with it.
		if isDigit(runes[i]) {
			text := codeNumberAt(runes[i:])
			keep(codeNumber, text)
			i += len([]rune(text))

			continue
		}

		if isWordRune(runes[i]) {
			text := wordAt(runes[i:])
			if part, named := codeWordPart(family, text); named {
				keep(part, text)
			} else {
				flat = append(flat, []rune(text)...)
			}

			i += len([]rune(text))

			continue
		}

		flat = append(flat, runes[i])
		i++
	}

	if len(flat) > 0 {
		out = append(out, codeToken{text: string(flat), part: codePlain})
	}

	return out
}

// commentAt says whether a comment opens here.
func commentAt(rest, family string) bool {
	for _, open := range codeComments[family] {
		if strings.HasPrefix(rest, open) {
			return true
		}
	}

	return false
}

// quotedAt is the literal that opens at the first rune, closing quote and
// all. A quote never closed runs to the end of the line, which is what a
// highlighter that reads one line at a time can honestly say about it.
func quotedAt(runes []rune) string {
	quote := runes[0]

	for i := 1; i < len(runes); i++ {
		if runes[i] == '\\' && quote != '`' {
			i++

			continue
		}

		if runes[i] == quote {
			return string(runes[:i+1])
		}
	}

	return string(runes)
}

// codeNumberAt is the literal that opens at the first rune: a hex body counts,
// because 0xFF in a stack trace is a number and reading it as two things is
// worse than reading it as one.
func codeNumberAt(runes []rune) string {
	for i, r := range runes {
		if isDigit(r) || r == '.' || r == '_' ||
			(i > 0 && (r == 'x' || r == 'X' || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F'))) {
			continue
		}

		return string(runes[:i])
	}

	return string(runes)
}

// wordAt is the identifier that opens at the first rune.
func wordAt(runes []rune) string {
	for i, r := range runes {
		if !isWordRune(r) {
			return string(runes[:i])
		}
	}

	return string(runes)
}

func isDigit(r rune) bool { return r >= '0' && r <= '9' }

// isWordRune is what an identifier is made of. Letters outside ASCII count:
// a language that allows them is a language whose identifiers this should not
// cut in half.
func isWordRune(r rune) bool {
	return r == '_' || isDigit(r) ||
		(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r > 127
}
