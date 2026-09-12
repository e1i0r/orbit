package verb

import "testing"

// TestEveryVerbIsSpelledOnce. The path is what the four ways in share —
// a child is both its words — so a duplicate is two verbs that would each
// claim the same command, route and tool. The bare name is not: two
// families may each have a show, and each answers through its parent.
func TestEveryVerbIsSpelledOnce(t *testing.T) {
	seen := map[string]bool{}

	for _, v := range Every() {
		if seen[v.Path()] {
			t.Errorf("%q is declared twice", v.Path())
		}

		seen[v.Path()] = true
	}
}

// TestEveryVerbSaysWhatItIs. A verb with no sentence is one every surface
// has to invent words for, which is where four vocabularies came from.
func TestEveryVerbSaysWhatItIs(t *testing.T) {
	for _, v := range Every() {
		if v.Name == "" {
			t.Error("a verb has no name")
		}

		if v.About == nil {
			t.Errorf("%q says nothing about what it is", v.Name)
		}

		for _, f := range v.Takes {
			if f.Name == "" || f.About == nil {
				t.Errorf("%q takes a field that does not say what it is", v.Name)
			}
		}
	}
}

// TestWhatSpendsAndWhatLeavesIsDeclared, because both are things every
// surface has to say out loud before it asks, and a surface deciding for
// itself which is which is how one of them forgets.
func TestWhatSpendsAndWhatLeavesIsDeclared(t *testing.T) {
	spends := map[string]bool{
		"run": true, "continue": true,
		"pr resolve": true, "pr update": true, "pr checks": true, "pr tests": true, "pr review": true,
	}
	outward := map[string]bool{
		"pr": true, "pr merge": true, "pr close": true,
		"pr update": true, "pr checks": true, "pr tests": true, "pr resolve": true, "pr review": true,
	}

	for _, v := range Every() {
		if v.Spends != spends[v.Path()] {
			t.Errorf("%q says it spends %v", v.Path(), v.Spends)
		}

		if v.Outward != outward[v.Path()] {
			t.Errorf("%q says it leaves this machine %v", v.Path(), v.Outward)
		}
	}
}

// TestWhatOnlyReadsIsDeclared. A surface has to be able to tell them apart:
// a reading is a GET and an action is a POST behind a guard, and a model may
// be given every reading while being trusted with only some of the actions.
func TestWhatOnlyReadsIsDeclared(t *testing.T) {
	reads := map[string]bool{
		"list": true, "show": true, "flow": true, "diff": true, "impact": true,
		"knowledge": true, "flows": true, "engines": true, "repos": true,
		"thread": true, "history": true, "quota": true, "pr show": true,
		"tree": true, "rules": true, "settings": true,
	}

	for _, v := range Every() {
		if v.Reads != reads[v.Path()] {
			t.Errorf("%q says it only reads: %v", v.Path(), v.Reads)
		}

		if v.Reads && (v.Spends || v.Outward) {
			t.Errorf("%q only reads, and also spends or leaves the machine", v.Path())
		}
	}
}

// TestOneFindsThemAndRefusesTheRest.
func TestOneFindsThemAndRefusesTheRest(t *testing.T) {
	if _, ok := One("pr merge"); !ok {
		t.Error("pr merge is not findable by name")
	}

	if _, ok := One("mergre"); ok {
		t.Error("a name nothing answers to was found anyway")
	}

	if _, ok := One("merge"); ok {
		t.Error("an old name still finds its verb")
	}
}
