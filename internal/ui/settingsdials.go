package ui

import "strings"

func modelsForEngine(eng string) []string {
	switch strings.ToLower(eng) {
	case "claude":
		return []string{"sonnet", "opus", "haiku"}
	case "codex":
		return []string{"o3-mini", "o1", "gpt-4o", "gpt-4.5"}
	case "opencode":
		return []string{"sonnet", "opus", "deepseek-r1", "qwen-2.5-coder", "gemini-2.5-pro"}
	default:
		return []string{"sonnet", "opus", "haiku", "o3-mini", "o1", "deepseek-r1", "qwen-2.5-coder"}
	}
}

func effortsForEngine(eng string) []string {
	switch strings.ToLower(eng) {
	case "claude":
		return []string{"default", "low", "medium", "high", "xhigh", "max"}
	case "codex":
		return []string{"default", "low", "medium", "high"}
	case "opencode":
		return []string{"default", "low", "medium", "high", "xhigh"}
	default:
		return []string{"default", "low", "medium", "high", "xhigh"}
	}
}
