package ui

import (
	"slices"
	"testing"
)

func TestModelsForEngine(t *testing.T) {
	tests := []struct {
		eng  string
		want []string
	}{
		{"claude", []string{"sonnet", "opus", "haiku"}},
		{"Claude", []string{"sonnet", "opus", "haiku"}}, // case-insensitive
		{"codex", []string{"o3-mini", "o1", "gpt-4o", "gpt-4.5"}},
		{"CODEX", []string{"o3-mini", "o1", "gpt-4o", "gpt-4.5"}},
		{"opencode", []string{"sonnet", "opus", "deepseek-r1", "qwen-2.5-coder", "gemini-2.5-pro"}},
		{"", []string{"sonnet", "opus", "haiku", "o3-mini", "o1", "deepseek-r1", "qwen-2.5-coder"}},
		{"some-other-engine", []string{"sonnet", "opus", "haiku", "o3-mini", "o1", "deepseek-r1", "qwen-2.5-coder"}},
	}
	for _, tt := range tests {
		if got := modelsForEngine(tt.eng); !slices.Equal(got, tt.want) {
			t.Errorf("modelsForEngine(%q) = %v, want %v", tt.eng, got, tt.want)
		}
	}
}

func TestEffortsForEngine(t *testing.T) {
	tests := []struct {
		eng  string
		want []string
	}{
		{"claude", []string{"default", "low", "medium", "high", "xhigh", "max"}},
		{"CLAUDE", []string{"default", "low", "medium", "high", "xhigh", "max"}},
		{"codex", []string{"default", "low", "medium", "high"}},
		{"Codex", []string{"default", "low", "medium", "high"}},
		{"opencode", []string{"default", "low", "medium", "high", "xhigh"}},
		{"", []string{"default", "low", "medium", "high", "xhigh"}},
		{"some-other-engine", []string{"default", "low", "medium", "high", "xhigh"}},
	}
	for _, tt := range tests {
		if got := effortsForEngine(tt.eng); !slices.Equal(got, tt.want) {
			t.Errorf("effortsForEngine(%q) = %v, want %v", tt.eng, got, tt.want)
		}
	}
}
