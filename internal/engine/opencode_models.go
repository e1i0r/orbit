package engine

// Every model opencode answers to, which is a list long enough to need a
// file of its own.

// opencodeModels is what `opencode models` prints, in the order it prints
// them, and the labels are those ids without the provider in front.
//
// It is the whole catalogue rather than a chosen dozen. The dozen this
// replaced was somebody's shortlist, and a reader whose work is on
// gpt-5.4-pro or kimi-k3 could not name it here at all — the dial is the
// only place a phase's model is chosen from, so a model missing from this
// list is a model Orbit cannot run.
//
// A written-down copy of a catalogue that moves is a cost taken
// deliberately, for the reason Models() gives: reading it at run time means
// shelling out to opencode before a dial can be drawn, on a machine where
// opencode may not be installed at all. Refresh it with `opencode models`.
var opencodeModels = []Choice{
	{ID: "", Label: "default"},
	{ID: "opencode/big-pickle", Label: "big-pickle"},
	{ID: "opencode/claude-fable-5", Label: "claude-fable-5"},
	{ID: "opencode/claude-fable-5-1", Label: "claude-fable-5-1"},
	{ID: "opencode/claude-haiku-4-5", Label: "claude-haiku-4-5"},
	{ID: "opencode/claude-opus-4-5", Label: "claude-opus-4-5"},
	{ID: "opencode/claude-opus-4-6", Label: "claude-opus-4-6"},
	{ID: "opencode/claude-opus-4-7", Label: "claude-opus-4-7"},
	{ID: "opencode/claude-opus-4-8", Label: "claude-opus-4-8"},
	{ID: "opencode/claude-opus-5", Label: "claude-opus-5"},
	{ID: "opencode/claude-sonnet-4", Label: "claude-sonnet-4"},
	{ID: "opencode/claude-sonnet-4-5", Label: "claude-sonnet-4-5"},
	{ID: "opencode/claude-sonnet-4-6", Label: "claude-sonnet-4-6"},
	{ID: "opencode/claude-sonnet-5", Label: "claude-sonnet-5"},
	{ID: "opencode/deepseek-v4-flash", Label: "deepseek-v4-flash"},
	{ID: "opencode/deepseek-v4-pro", Label: "deepseek-v4-pro"},
	{ID: "opencode/gemini-3-flash", Label: "gemini-3-flash"},
	{ID: "opencode/gemini-3.1-pro", Label: "gemini-3.1-pro"},
	{ID: "opencode/gemini-3.5-flash", Label: "gemini-3.5-flash"},
	{ID: "opencode/gemini-3.5-flash-lite", Label: "gemini-3.5-flash-lite"},
	{ID: "opencode/gemini-3.6-flash", Label: "gemini-3.6-flash"},
	{ID: "opencode/gemini-3.7-flash", Label: "gemini-3.7-flash"},
	{ID: "opencode/gemini-3.8-flash", Label: "gemini-3.8-flash"},
	{ID: "opencode/glm-5", Label: "glm-5"},
	{ID: "opencode/glm-5.1", Label: "glm-5.1"},
	{ID: "opencode/glm-5.2", Label: "glm-5.2"},
	{ID: "opencode/gpt-5", Label: "gpt-5"},
	{ID: "opencode/gpt-5-codex", Label: "gpt-5-codex"},
	{ID: "opencode/gpt-5-nano", Label: "gpt-5-nano"},
	{ID: "opencode/gpt-5.1", Label: "gpt-5.1"},
	{ID: "opencode/gpt-5.1-codex", Label: "gpt-5.1-codex"},
	{ID: "opencode/gpt-5.1-codex-max", Label: "gpt-5.1-codex-max"},
	{ID: "opencode/gpt-5.1-codex-mini", Label: "gpt-5.1-codex-mini"},
	{ID: "opencode/gpt-5.2", Label: "gpt-5.2"},
	{ID: "opencode/gpt-5.2-codex", Label: "gpt-5.2-codex"},
	{ID: "opencode/gpt-5.3-codex", Label: "gpt-5.3-codex"},
	{ID: "opencode/gpt-5.3-codex-spark", Label: "gpt-5.3-codex-spark"},
	{ID: "opencode/gpt-5.4", Label: "gpt-5.4"},
	{ID: "opencode/gpt-5.4-mini", Label: "gpt-5.4-mini"},
	{ID: "opencode/gpt-5.4-nano", Label: "gpt-5.4-nano"},
	{ID: "opencode/gpt-5.4-pro", Label: "gpt-5.4-pro"},
	{ID: "opencode/gpt-5.5", Label: "gpt-5.5"},
	{ID: "opencode/gpt-5.5-pro", Label: "gpt-5.5-pro"},
	{ID: "opencode/gpt-5.6-luna", Label: "gpt-5.6-luna"},
	{ID: "opencode/gpt-5.6-sol", Label: "gpt-5.6-sol"},
	{ID: "opencode/gpt-5.6-terra", Label: "gpt-5.6-terra"},
	{ID: "opencode/grok-4.5", Label: "grok-4.5"},
	{ID: "opencode/grok-4.6", Label: "grok-4.6"},
	{ID: "opencode/grok-build-0.1", Label: "grok-build-0.1"},
	{ID: "opencode/kimi-k2.5", Label: "kimi-k2.5"},
	{ID: "opencode/kimi-k2.6", Label: "kimi-k2.6"},
	{ID: "opencode/kimi-k2.7-code", Label: "kimi-k2.7-code"},
	{ID: "opencode/kimi-k3", Label: "kimi-k3"},
	{ID: "opencode/ling-3.0-flash-fin-free", Label: "ling-3.0-flash-fin-free"},
	{ID: "opencode/mimo-v2.5-free", Label: "mimo-v2.5-free"},
	{ID: "opencode/minimax-m2.5", Label: "minimax-m2.5"},
	{ID: "opencode/minimax-m2.7", Label: "minimax-m2.7"},
	{ID: "opencode/minimax-m3", Label: "minimax-m3"},
	{ID: "opencode/muse-spark-1.2", Label: "muse-spark-1.2"},
	{ID: "opencode/muse-spark-1.2-contributor-free", Label: "muse-spark-1.2-contributor-free"},
	{ID: "opencode/muse-spark-1.3-contributor-free", Label: "muse-spark-1.3-contributor-free"},
	{ID: "opencode/nemotron-3-ultra-free", Label: "nemotron-3-ultra-free"},
	{ID: "opencode/nemotron-3.5-lightning-free", Label: "nemotron-3.5-lightning-free"},
	{ID: "opencode/qwen3.5-plus", Label: "qwen3.5-plus"},
	{ID: "opencode/qwen3.6-plus", Label: "qwen3.6-plus"},
}
