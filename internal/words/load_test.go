package words

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCatalogOverlaysOnTopOfTheEmbeddedOne(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ORBIT_HOME", home)

	langDir := filepath.Join(home, "lang")
	if err := os.MkdirAll(langDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	body := `{"language": "Español", "keys": {"greeting.hello": {"source": "Hello", "value": "Hola"}}}`
	if err := os.WriteFile(filepath.Join(langDir, "es.json"), []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	p := For("es")

	got := p.T("greeting.hello", "Hello")
	if want := "Hola"; got != want {
		t.Errorf("T() = %q, want %q — the overlay should have won", got, want)
	}
}

func TestLoadCatalogSurvivesACorruptOverlay(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ORBIT_HOME", home)

	langDir := filepath.Join(home, "lang")
	if err := os.MkdirAll(langDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// A trailing comma: invalid JSON.
	body := `{"language": "Español", "keys": {},}`
	if err := os.WriteFile(filepath.Join(langDir, "es.json"), []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	p := For("es")

	got := p.T("anything.at.all", "still shows English")
	if want := "still shows English"; got != want {
		t.Errorf("T() = %q, want %q — a malformed overlay must cost the reader nothing", got, want)
	}
}

func TestLoadCatalogSurvivesAMissingOrbitHome(t *testing.T) {
	t.Setenv("ORBIT_HOME", filepath.Join(t.TempDir(), "does-not-exist"))

	p := For("es")

	got := p.T("anything", "English text")
	if want := "English text"; got != want {
		t.Errorf("T() = %q, want %q", got, want)
	}
}

func TestLoadCatalogRejectsAnUnknownField(t *testing.T) {
	_, err := parseCatalog([]byte(`{"language": "English", "keys": {}, "extra": true}`))
	if err == nil {
		t.Error("parseCatalog accepted an unknown top-level field")
	}
}

func TestTextUnmarshalsAPlainStringAsSingular(t *testing.T) {
	var tx text
	if err := tx.UnmarshalJSON([]byte(`"hi"`)); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	if tx.Single != "hi" || tx.IsPlural {
		t.Errorf("got %+v, want a singular %q", tx, "hi")
	}
}

func TestTextUnmarshalsAnObjectAsPlural(t *testing.T) {
	var tx text
	if err := tx.UnmarshalJSON([]byte(`{"one": "1 task", "other": "{n} tasks"}`)); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	if !tx.IsPlural || tx.One != "1 task" || tx.Other != "{n} tasks" {
		t.Errorf("got %+v", tx)
	}
}

func TestTextUnmarshalRejectsSomethingElse(t *testing.T) {
	var tx text
	if err := tx.UnmarshalJSON([]byte(`42`)); err == nil {
		t.Error("UnmarshalJSON accepted a number")
	}
}

func TestLoadBudgetsReadsOnlyPositiveCells(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ORBIT_HOME", home)

	langDir := filepath.Join(home, "lang")
	if err := os.MkdirAll(langDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	body := `{"language": "English", "keys": {
		"state.todo": {"cells": 5},
		"state.unbounded": {"cells": 0}
	}}`
	if err := os.WriteFile(filepath.Join(langDir, "en.json"), []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	budgets := loadBudgets()
	if budgets["state.todo"] != 5 {
		t.Errorf("budgets[state.todo] = %d, want 5", budgets["state.todo"])
	}

	if _, ok := budgets["state.unbounded"]; ok {
		t.Errorf("a declared cells of 0 should not appear as a budget: %v", budgets)
	}
}
