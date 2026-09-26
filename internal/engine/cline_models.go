package engine

// Every model cline's own subscription answers to.

// clineModels is the cline-pass catalogue cline ships with, and the labels
// are those ids without the provider in front, as opencode's are.
//
// Only cline-pass, and that is a choice with a reason. cline answers to
// whatever provider it was signed into — Anthropic, OpenRouter, a local
// endpoint — and each of those has its own ids, which cline reads live. The
// one catalogue that is cline's own is its subscription's, and default is
// always there: it is whatever model the reader last chose in cline itself.
// Refresh it from @cline/llms's bundled models.
var clineModels = []Choice{
	{ID: "", Label: "default"},
	{ID: "cline-pass/deepseek-v4-flash", Label: "deepseek-v4-flash"},
	{ID: "cline-pass/deepseek-v4-pro", Label: "deepseek-v4-pro"},
	{ID: "cline-pass/deepseek-v4.1-flash", Label: "deepseek-v4.1-flash"},
	{ID: "cline-pass/glm-5.2", Label: "glm-5.2"},
	{ID: "cline-pass/glm-5.3", Label: "glm-5.3"},
	{ID: "cline-pass/glm-5.3-flash", Label: "glm-5.3-flash"},
	{ID: "cline-pass/kimi-k2.6", Label: "kimi-k2.6"},
	{ID: "cline-pass/kimi-k2.7-code", Label: "kimi-k2.7-code"},
	{ID: "cline-pass/kimi-k3", Label: "kimi-k3"},
	{ID: "cline-pass/mimo-v2.5", Label: "mimo-v2.5"},
	{ID: "cline-pass/mimo-v2.5-pro", Label: "mimo-v2.5-pro"},
	{ID: "cline-pass/minimax-m3", Label: "minimax-m3"},
	{ID: "cline-pass/qwen3.7-max", Label: "qwen3.7-max"},
	{ID: "cline-pass/qwen3.7-plus", Label: "qwen3.7-plus"},
	{ID: "cline-pass/qwen3.8-max", Label: "qwen3.8-max"},
}
