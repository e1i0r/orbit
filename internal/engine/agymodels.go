package engine

// Every model agy answers to.

// agyModels is what `agy models` prints and what --model takes.
//
// It is what `agy models` prints, IDs on the left and its own labels on the
// right, and it is a written-down copy of a catalogue that moves for the
// reason opencode's is: reading it at run time means shelling out to agy
// before a dial can be drawn, on a machine where agy may not be installed.
// Refresh it with `agy models`.
//
// Gemini names its reasoning inside the model — a family appears three
// times, high, medium and low — while --effort says the same thing beside
// it. Both are offered as they are printed rather than folded together,
// because what a dial shows and what the binary is told have to be the same
// string.
var agyModels = []Choice{
	{ID: "", Label: "default"},
	{ID: "gemini-3.8-flash-high", Label: "Gemini 3.8 Flash (High)"},
	{ID: "gemini-3.8-flash-medium", Label: "Gemini 3.8 Flash (Medium)"},
	{ID: "gemini-3.8-flash-low", Label: "Gemini 3.8 Flash (Low)"},
	{ID: "gemini-3.7-flash-high", Label: "Gemini 3.7 Flash (High)"},
	{ID: "gemini-3.7-flash-medium", Label: "Gemini 3.7 Flash (Medium)"},
	{ID: "gemini-3.7-flash-low", Label: "Gemini 3.7 Flash (Low)"},
	{ID: "gemini-3.6-flash-high", Label: "Gemini 3.6 Flash (High)"},
	{ID: "gemini-3.6-flash-medium", Label: "Gemini 3.6 Flash (Medium)"},
	{ID: "gemini-3.6-flash-low", Label: "Gemini 3.6 Flash (Low)"},
	{ID: "gemini-3.1-pro-high", Label: "Gemini 3.1 Pro (High)"},
	{ID: "gemini-3.1-pro-low", Label: "Gemini 3.1 Pro (Low)"},
	{ID: "claude-sonnet-4-6", Label: "Claude Sonnet 4.6 (Thinking)"},
	{ID: "claude-opus-4-6-thinking", Label: "Claude Opus 4.6 (Thinking)"},
	{ID: "gpt-oss-120b-medium", Label: "GPT-OSS 120B (Medium)"},
}
