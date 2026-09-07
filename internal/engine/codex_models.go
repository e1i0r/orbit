package engine

// Every model codex answers to.

// codexModels is what `codex exec --model` takes.
//
// The five names this replaced were o3-mini, o3, o1, gpt-4o and
// gpt-4.5-preview — a list from before codex shipped. internal/task checks a
// phase's model against this one before it runs anything, so the only models
// a phase could name on codex were five the account answers "not supported"
// to. It was then cut to nothing at all, which was the opposite mistake: the
// dial had one position and codex has four.
//
// These four are what `codex exec --model` was actually run with, one at a
// time, on codex-cli 0.150.1. The other slugs in the binary — gpt-5.6-sol,
// gpt-5.6-pro, gpt-5.4, gpt-5.3-codex, gpt-5.2, gpt-5.2-codex — are the
// legacy names its own picker points at config.toml for, and every one of
// them came back "not supported". Refresh this the same way, by running
// them; codex has no verb that lists them.
var codexModels = []Choice{
	{ID: "", Label: "default"},
	{ID: "gpt-5.6-terra", Label: "gpt-5.6-terra"},
	{ID: "gpt-5.6-luna", Label: "gpt-5.6-luna"},
	{ID: "gpt-5.5", Label: "gpt-5.5"},
	{ID: "gpt-5.4-mini", Label: "gpt-5.4-mini"},
}
