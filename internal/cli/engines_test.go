package cli

// The one table of engines, and the two questions asked of it.

import (
	"slices"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/engine"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/words"
)

// TestTheWindowOffersEveryEngineThisBuildCanRun is the promise the table
// makes. If the list the settings screen draws were names written out again
// beside the map, an engine added to the build would be an engine the screen
// does not offer — startable from the command line and invisible on screen.
func TestTheWindowOffersEveryEngineThisBuildCanRun(t *testing.T) {
	engines := newEngines()
	if len(engines) == 0 {
		t.Fatal("this build can run no engine at all")
	}

	var offered []string
	for _, info := range enginesPort(engines)() {
		offered = append(offered, info.Name)
	}

	want := engineNames(engines)
	if !slices.Equal(offered, want) {
		t.Errorf("the settings screen offers %v, and this build can run %v", offered, want)
	}
}

// TestTheListIsInTheSameOrderEveryTime. It comes off a map, which has none,
// and a screen whose rows move between two openings is a screen nobody can
// learn the shape of.
func TestTheListIsInTheSameOrderEveryTime(t *testing.T) {
	first := engineNames(newEngines())

	if !slices.IsSorted(first) {
		t.Errorf("the engines are listed %v, which is not an order anything decided", first)
	}

	for range 20 {
		if got := engineNames(newEngines()); !slices.Equal(got, first) {
			t.Fatalf("the engines were listed %v and then %v", first, got)
		}
	}
}

// TestTheSupervisorAnswersOnTheEngineItWasAskedFor. Falling back to claude
// for a name the table does not have is the one substitution a reader cannot
// detect: the window labels the reply with the engine their dial names, so an
// answer written by claude comes back under the name of the engine they chose
// — on their claude quota, in claude's voice, and reported as codex.
func TestTheSupervisorAnswersOnTheEngineItWasAskedFor(t *testing.T) {
	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	engines := newEngines()

	_, err = askSupervisorPort(s, engines)("an-engine-nobody-has", "how is it going?")
	if err == nil {
		t.Fatal("the supervisor answered on an engine that was not asked for")
	}

	if !strings.Contains(err.Error(), "an-engine-nobody-has") {
		t.Errorf("the refusal does not name the engine: %v", err)
	}

	_, err = autoSupervisePort(s, engines)("an-engine-nobody-has", []string{"ACME-1"})
	if err == nil {
		t.Fatal("autopilot supervised on an engine that was not asked for")
	}

	if !strings.Contains(err.Error(), "an-engine-nobody-has") {
		t.Errorf("the refusal does not name the engine: %v", err)
	}
}

// TestAnEngineTheTableHasNoRunnerFor is the other half of the same question:
// a name that is in the table with nothing behind it is not an engine, and
// calling Supervise on the nil would take the window down with it.
func TestAnEngineTheTableHasNoRunnerFor(t *testing.T) {
	if _, err := engineNamed(map[string]engine.Engine{"hollow": nil}, "hollow"); err == nil {
		t.Error("a name with no engine behind it was answered as an engine")
	}
}

// TestARefusalReadsAsASentenceWithOrWithoutATask. The same record is put in
// the activity band from two places — a task whose record names an engine
// this build dropped, and a supervisor dial naming one — and only one of
// them has a task to name.
func TestARefusalReadsAsASentenceWithOrWithoutATask(t *testing.T) {
	withTask := (&unknownEngineError{Name: "codex", ID: "ACME-1"}).Error()
	if !strings.Contains(withTask, "ACME-1") || !strings.Contains(withTask, "codex") {
		t.Errorf("a task's refusal names neither the task nor the engine: %q", withTask)
	}

	alone := (&unknownEngineError{Name: "codex"}).Error()
	if strings.Contains(alone, "task ") {
		t.Errorf("a refusal with no task behind it still talks about one: %q", alone)
	}

	if !strings.Contains(alone, "codex") {
		t.Errorf("the refusal does not name the engine: %q", alone)
	}
}

// TestTheSetupStepsAreInTheReadersLanguage. The engines screen puts three
// steps under an engine this machine cannot run yet — and its title, its
// notice and the way back out of it were already translated, so the steps
// were the only English left on a screen a reader is expected to follow.
func TestTheSetupStepsAreInTheReadersLanguage(t *testing.T) {
	for _, name := range engineNames(newEngines()) {
		guide := setupGuide(name)
		if guide == nil {
			t.Errorf("%s is an engine this build offers and nobody wrote the steps for", name)
			continue
		}

		english, spanish := guide(words.For("en")), guide(words.For("es"))
		if len(spanish) != len(english) {
			t.Errorf("%s has %d steps in English and %d in Spanish", name, len(english), len(spanish))
			continue
		}

		for i := range english {
			if english[i] == spanish[i] {
				t.Errorf("%s step %d is the same sentence in both languages: %q", name, i+1, english[i])
			}
		}
	}
}

// TestTheCommandToTypeIsNotTranslated is the other half of it. The steps are
// read; the command in them is typed, and a translated command does not run.
func TestTheCommandToTypeIsNotTranslated(t *testing.T) {
	for name, command := range map[string]string{
		"claude": "npm install -g @anthropic-ai/claude-code",
		"codex":  "npm install -g @openai/codex",
	} {
		var found bool

		for _, step := range setupGuide(name)(words.For("es")) {
			found = found || strings.Contains(step, command)
		}

		if !found {
			t.Errorf("the Spanish steps for %s do not carry %q for the reader to type", name, command)
		}
	}
}
