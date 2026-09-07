package engine

// Every model claude answers to.

// claudeModels is what --model takes: the aliases rather than the dated
// slugs, because an alias is what a reader recognises and it follows the
// account onto whichever version claude is shipping that week.
//
// The catalogue lives in a file of its own, as each engine's does, so that
// "which models does this engine have" is one place to look rather than a
// hunt through the adapter that runs it.
var claudeModels = []Choice{
	{ID: "", Label: "default"},
	{ID: "opus", Label: "opus"},
	{ID: "sonnet", Label: "sonnet"},
	{ID: "haiku", Label: "haiku"},
}
