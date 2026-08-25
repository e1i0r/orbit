package ui

import (
	"testing"

	"github.com/e1i0r/orbit/internal/words"
)

func TestPlainRenderingComprehensive(t *testing.T) {
	p := words.For("en")
	opts := Options{
		Root:   "/workspace/orbit",
		Words:  p,
		Width:  100,
		Height: 30,
	}

	output, err := Plain(opts)
	if err != nil {
		t.Fatalf("Plain failed: %v", err)
	}
	if output == "" {
		t.Fatal("Plain returned empty string")
	}
}
