package ui

// What the engine is told when somebody asks for a flow in words.

import "strings"

// flowDraftPrompt is that instruction: the shape to answer in, one whole
// example of it, and the handful of rules internal/flow will refuse a draft
// over.
//
// The example is the specification. A list of field names and their types is
// something a model reads and then improvises around; one complete document
// of the right shape is something it copies. The rules under it are only the
// ones a reader cannot fix by editing a field afterwards.
//
// The engine names are listed rather than left to the model, because a phase
// naming an engine this machine does not have is a flow that cannot run and
// a reader who has to work out why.
func flowDraftPrompt(said string, engines []string) string {
	var b strings.Builder

	b.WriteString("Write one Orbit flow. The whole of your answer must be one JSON object: ")
	b.WriteString("no prose before it, no explanation after it, no markdown fences, nothing but the object. ")
	b.WriteString("Keep to the shape below exactly — the field names, the nesting, and no field that is not in it.\n\n")
	b.WriteString("A flow is phases run in order. Each phase is one coding agent given instructions. ")
	b.WriteString("A phase can instead be a loop, which repeats until every one of its checks exits zero.\n\n")

	b.WriteString("Answer in exactly this shape. This is a whole example, not a fragment:\n\n")
	b.WriteString(draftExample(firstEngine(engines)))

	b.WriteString("\nThe rules a wrong answer is refused over:\n")
	b.WriteString("- engine is one of: " + strings.Join(engines, ", ") + ", and the same one throughout unless asked otherwise.\n")
	b.WriteString("- a phase has an engine or a loop, never both. The loop's own phases carry the engine.\n")
	b.WriteString("- a loop needs max and at least one check in until. A check is a shell command, and its exit code is what says the work is done — never a model saying so.\n")
	b.WriteString("- permissions are read, repo or network. Ask for the least that will do.\n")
	b.WriteString("- wait: true stops the flow for a person, and belongs on a review phase if there is one.\n")
	b.WriteString("- leave model and effort out unless the request names one.\n")
	b.WriteString("- write the prompts in the language the request below is written in.\n")
	b.WriteString("- valid JSON: no comments, no trailing commas, a line break inside a string is \\n, a quote inside a string is \\\". ")
	b.WriteString("Commands need no quotes around them: write go test ./... bare.\n\n")

	b.WriteString("The request:\n")
	b.WriteString(said)
	b.WriteString("\n")

	return b.String()
}

// draftExample is the whole document, with a loop in it, on the engine this
// build would run.
func draftExample(engine string) string {
	return `{
  "name": "safe-refactor",
  "description": "plan it, apply it, then go round until the tests pass",
  "attempts": 2,
  "phases": [
    {
      "name": "1-plan",
      "engine": "` + engine + `",
      "permissions": ["read"],
      "prompt": "Study the code and write the plan. Change nothing."
    },
    {
      "name": "2-until-it-passes",
      "loop": {
        "max": 3,
        "until": [
          {"name": "tests", "command": "go test ./..."},
          {"name": "lint", "command": "golangci-lint run"}
        ],
        "phases": [
          {
            "name": "fix",
            "engine": "` + engine + `",
            "feed_output": true,
            "permissions": ["repo"],
            "prompt": "Read what the checks said above and fix what failed."
          }
        ]
      }
    },
    {
      "name": "3-review",
      "engine": "` + engine + `",
      "wait": true,
      "feed_output": true,
      "permissions": ["repo"],
      "prompt": "Review the whole diff: what it does, what it broke, what it left."
    }
  ]
}
`
}

// firstEngine is the name the example is written on, so the model copies an
// engine this build actually has.
func firstEngine(engines []string) string {
	if len(engines) == 0 {
		return "claude"
	}

	return engines[0]
}

// mendDraftPrompt is the second ask: what came back, what the decoder said
// about it, and nothing else. The engine wrote it, so the engine is the one
// thing that knows what it meant to say.
func mendDraftPrompt(out string, err error) string {
	var b strings.Builder

	b.WriteString("That was not valid JSON. The decoder said:\n")
	b.WriteString(err.Error())
	b.WriteString("\n\nHere is what you wrote:\n")
	b.WriteString(out)
	b.WriteString("\n\nAnswer with the corrected JSON alone — no prose, no fences. ")
	b.WriteString("Escape every double quote inside a string as \\\", and every line break as \\n.\n")

	return b.String()
}
