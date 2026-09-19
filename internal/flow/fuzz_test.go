package flow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuiltinFlowsValidityProperty(t *testing.T) {
	for _, name := range []string{"task", "quick", "careful"} {
		f, err := Resolve(nil, name)
		if err != nil {
			t.Errorf("builtin flow %q failed to resolve: %v", name, err)
		}

		if len(f.Phases) == 0 {
			t.Errorf("builtin flow %q has 0 phases", name)
		}

		if f.Name != name {
			t.Errorf("builtin flow Name = %q, want %q", f.Name, name)
		}
	}
}

// FuzzFlowValidation is a flow file as somebody wrote it.
//
// A flow is JSON a person edits, so it arrives with a phase that names no
// engine, a loop with nothing that says when it is done, or a name with a
// slash in it. What validates is what a run will walk, so a flow that passes
// here has to be one every phase of which can be given to something.
func FuzzFlowValidation(f *testing.F) {
	f.Add([]byte(`{"name":"test","phases":[{"name":"p1","engine":"claude","model":"sonnet"}]}`))
	f.Add([]byte(`{"name":"invalid","phases":[]}`))
	f.Add([]byte(`broken flow format`))
	f.Add([]byte(`{"name":"l","phases":[{"name":"p","loop":{"phases":[{"name":"q","engine":"c"}]}}]}`))
	f.Add([]byte(`{"name":"","phases":[{"name":"p","engine":"c"}]}`))
	f.Add([]byte(`{"name":"a/b","phases":[{"name":"p","engine":"c"}]}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var fl Flow
		if err := json.Unmarshal(data, &fl); err != nil {
			return
		}

		if err := fl.Validate(); err != nil {
			// A refusal says what is wrong with the file, because it is
			// printed at whoever wrote it.
			if err.Error() == "" {
				t.Errorf("a flow was refused with nothing said: %s", data)
			}

			return
		}

		// A flow with no phases walks nothing, and a run that started one
		// would finish before it began.
		if len(fl.Phases) == 0 {
			t.Errorf("a flow with no phases validated: %s", data)
		}

		for i, p := range fl.Phases {
			if p.Name == "" {
				t.Errorf("phase %d of a flow that validated has no name: %s", i, data)
			}

			// A loop names no engine and its phases do, which is the one
			// shape where the phase itself may name none.
			if p.Engine == "" && p.Loop == nil {
				t.Errorf("phase %q of a flow that validated is put to no engine: %s", p.Name, data)
			}

			if p.Loop == nil {
				continue
			}

			if len(p.Loop.Phases) == 0 {
				t.Errorf("the loop at phase %q validated with nothing in it: %s", p.Name, data)
			}

			for _, inner := range p.Loop.Phases {
				if inner.Name == "" || inner.Engine == "" {
					t.Errorf("a phase inside the loop at %q is %q put to %q: %s",
						p.Name, inner.Name, inner.Engine, data)
				}
			}
		}
	})
}

// FuzzAFlowNameIsAFileName.
//
// The name is a filename before it is anything else: Resolve joins it to the
// directory flows live in and reads what is there. A name with a separator
// in it, or one that is a way of saying "up", reads a file the reader never
// chose and never saw.
func FuzzAFlowNameIsAFileName(f *testing.F) {
	for _, seed := range []string{
		"", "task", "quick", "..", ".", "a/b", `a\b`, "../../etc/passwd",
		"a..b", "..a", "a/..", " ", "tdd-fuzz-pr",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, name string) {
		if err := ValidName(name); err != nil {
			return
		}

		if name == "" {
			t.Errorf("a flow with no name at all was accepted")
		}

		if strings.ContainsRune(name, '/') || strings.ContainsRune(name, os.PathSeparator) {
			t.Errorf("the flow name %q holds a path separator and was accepted", name)
		}

		if name == "." || strings.Contains(name, "..") {
			t.Errorf("the flow name %q is a way of saying somewhere else and was accepted", name)
		}

		// What it reads is the name joined to a directory, and that has to
		// stay one entry inside it.
		if joined := filepath.Join("/flows", name+".json"); filepath.Dir(joined) != "/flows" {
			t.Errorf("the flow name %q reads %q, which is not in the flow directory", name, joined)
		}
	})
}
