package verb

// Asking for a verb, and what it needs to be able to answer.
//
// The declaration next door says what Orbit can be asked for; this is where
// the asking happens, once, for all four ways in. A surface's whole job
// becomes turning what it received — a command line, a JSON body, a tool
// call, a keystroke — into an In, and turning the Out back into its own
// idiom. Which verbs it chooses to offer is its own business; what it may
// not do any more is have its own idea of what a verb means.
//
// What a verb needs of the machine arrives as a World: the store it writes
// to, the board it reads, the engines it can run. They are ports because
// the same verb has to work for a command line that opened a store a moment
// ago, a server that has held one open for hours, and a test that has
// neither.

import (
	"context"
	"errors"
	"fmt"

	"github.com/e1i0r/orbit/internal/words"
)

// In is what a verb was asked with.
type In struct {
	// Task is the id it is about, and empty for the verbs that are about
	// the board instead.
	Task string
	// Repo is the path of the repository the task is against. Empty is a
	// task against none, which is a task all the same.
	Repo string
	// Args is what the reader typed, by the field names the verb Takes.
	Args map[string]string
	// By is who asked, in the words the record shows: "operator" for a
	// person at any of the controls, an engine's name for a model.
	By string
	// Door is where they asked it: the cockpit, the command line, the
	// browser, a tool call. What the supervisor is handed names it, so
	// that an instruction never claims a key was pressed that was not.
	Door string
}

// Arg is one field, and the empty string for one nobody filled in.
func (i In) Arg(name string) string { return i.Args[name] }

// Yes is a yes-or-no field, and false for anything that is not a plain yes.
func (i In) Yes(name string) bool {
	switch i.Args[name] {
	case "true", "yes", "1", "on":
		return true
	}

	return false
}

// Out is what a verb answered.
type Out struct {
	// Said is the sentence a reader is shown. It is written by the verb
	// and not by the surface, so the browser and the terminal report the
	// same act in the same words.
	Said string
	// Of is what the verb acted on, where that is a list worth showing:
	// the libraries an approve accepted, the files a walk touched.
	Of []string
	// Pid is the process a start began, for the caller that watches for
	// it. Zero when this asking started nothing: most verbs leave it
	// alone, and no surface prints it unasked.
	Pid int
	// Saw is what a reading read, in the shape it was read in.
	//
	// Said carries the same answer written out for a terminal, so a
	// surface with nowhere to put a structure still has something to
	// print. A browser or a tool call takes this instead and draws it.
	Saw any
}

// Run asks for one verb by name.
//
// A name nothing answers to is a refusal and not a panic: the name arrives
// from a command line, a URL and a tool call, and all three can carry a
// typo.
func Run(ctx context.Context, w World, name string, in In) (Out, error) {
	v, known := One(name)
	if !known {
		return Out{}, fmt.Errorf("%q is not something Orbit can be asked for", name)
	}

	if err := v.needs(printer(w), in); err != nil {
		return Out{}, err
	}

	out, err := v.do(ctx, w, in)

	return out, err
}

// printer is the reader's language, and English where there is no world to
// ask. Both refusals above happen before the world is reached — a name
// nothing answers to, a verb asked for without what it takes — so a caller
// with nothing to run the verb on can still be told why.
func printer(w World) *words.Printer {
	if w == nil {
		return words.For("")
	}

	return w.Words()
}

// needs refuses a verb that was asked for without what it takes, before
// anything is written down.
//
// Here rather than in each surface, because "a note needs something written
// in it" is a fact about the verb, and four copies of it were four chances
// for one of them to let an empty note through.
func (v Verb) needs(p *words.Printer, in In) error {
	if v.OnTask && in.Task == "" {
		return errors.New(p.T("verb.needs_a_task", "{verb} needs a task",
			words.Arg{Name: "verb", Value: v.Path()}))
	}

	for _, f := range v.Takes {
		if f.Needed && in.Arg(f.Name) == "" {
			return errors.New(p.T("verb.needs_field", "{verb} needs {field}",
				words.Arg{Name: "verb", Value: v.Path()},
				words.Arg{Name: "field", Value: f.Name}))
		}
	}

	return nil
}
