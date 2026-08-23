package words

// printer_test.go exercises T, P, Cells and For directly. It is safe to
// call p.T and p.P here — collectCallSites skips internal/words entirely —
// so these tests use the real public API rather than its unexported parts.

import "testing"

func TestTFallsBackToEnglishWithNoCatalogue(t *testing.T) {
	p := For("xx") // no lang/xx.json exists, embedded or overlaid
	got := p.T("greeting.hello", "Hello, {name}!", Arg{Name: "name", Value: "World"})
	if want := "Hello, World!"; got != want {
		t.Errorf("T() = %q, want %q", got, want)
	}
}

func TestTPrefersTheCatalogueOverride(t *testing.T) {
	p := &Printer{strs: catalog{keys: map[string]entry{
		"greeting.hello": {Value: text{Single: "Hola, {name}!"}},
	}}}
	got := p.T("greeting.hello", "Hello, {name}!", Arg{Name: "name", Value: "Mundo"})
	if want := "Hola, Mundo!"; got != want {
		t.Errorf("T() = %q, want %q", got, want)
	}
}

func TestTLeavesAnUnknownPlaceholderAlone(t *testing.T) {
	p := &Printer{}
	got := p.T("x", "no placeholders here")
	if want := "no placeholders here"; got != want {
		t.Errorf("T() = %q, want %q", got, want)
	}
}

func TestPSelectsOneVersusOther(t *testing.T) {
	p := &Printer{}
	if got, want := p.P("task.count", 1, "{n} task", "{n} tasks"), "1 task"; got != want {
		t.Errorf("P(1, ...) = %q, want %q", got, want)
	}
	if got, want := p.P("task.count", 3, "{n} task", "{n} tasks"), "3 tasks"; got != want {
		t.Errorf("P(3, ...) = %q, want %q", got, want)
	}
	if got, want := p.P("task.count", 0, "{n} task", "{n} tasks"), "0 tasks"; got != want {
		t.Errorf("P(0, ...) = %q, want %q", got, want)
	}
}

func TestPPrefersTheCatalogueOverridePerForm(t *testing.T) {
	p := &Printer{strs: catalog{keys: map[string]entry{
		"task.count": {Value: text{IsPlural: true, One: "{n} tarea", Other: "{n} tareas"}},
	}}}
	if got, want := p.P("task.count", 1, "{n} task", "{n} tasks"), "1 tarea"; got != want {
		t.Errorf("P(1, ...) = %q, want %q", got, want)
	}
	if got, want := p.P("task.count", 2, "{n} task", "{n} tasks"), "2 tareas"; got != want {
		t.Errorf("P(2, ...) = %q, want %q", got, want)
	}
}

func TestPDoesNotRequireTheCallerToNameN(t *testing.T) {
	p := &Printer{}
	got := p.P("repo.tasks", 4, "{repo} · {n} task", "{repo} · {n} tasks", Arg{Name: "repo", Value: "orbit"})
	if want := "orbit · 4 tasks"; got != want {
		t.Errorf("P() = %q, want %q", got, want)
	}
}

func TestCellsIsZeroWithNoBudgetDeclared(t *testing.T) {
	p := &Printer{}
	if got := p.Cells("anything"); got != 0 {
		t.Errorf("Cells() = %d, want 0", got)
	}
}

func TestCellsReadsTheDeclaredBudget(t *testing.T) {
	p := &Printer{budgets: map[string]int{"state.todo": 9}}
	if got := p.Cells("state.todo"); got != 9 {
		t.Errorf("Cells() = %d, want 9", got)
	}
}

func TestForNeverFailsOnAnUnknownLanguage(t *testing.T) {
	p := For("zz")
	if p == nil {
		t.Fatal("For returned nil")
	}
	if got := p.T("x", "still English"); got != "still English" {
		t.Errorf("T() = %q, want the English fallback", got)
	}
}
