package engine

// Whether an engine stopped because its allowance ran out.
//
// A machine that was switched off is an accident. Running out of quota is
// expected, frequent and recoverable — and today they end a run the same
// way, so a reader who comes back to a task that says only "it broke" has to
// open the log to find out which of the two it was. Those two send somebody
// to do completely different things: one is a bug to look at, the other is a
// wait or another engine.
//
// It is read off what the engine printed, because that is the only place it
// exists. exec gives a program one way to say it stopped — a non-zero exit —
// and every one of these CLIs surfaces the provider's own words above it.
//
// An engine that says nothing this recognises answers no, and the run is
// written down as broken, which is what happens today. Nothing gets worse by
// not knowing.

import "strings"

// spentMarks are the words a provider uses when the allowance is gone.
//
// Shared rather than per engine because they do not come from the engines:
// all four are command lines over the same handful of APIs, and what they
// print is the API's own refusal. Anthropic says usage limit, OpenAI says
// insufficient_quota, everybody says 429 eventually.
//
// Hand-written and meant to be argued with. A phrase added here turns a
// broken run into a run that ran out, which is a change in what a reader is
// told — so it is a list somebody reads, not a regular expression somebody
// trusts.
var spentMarks = []string{
	"usage limit",
	"rate limit",
	"rate_limit",
	"insufficient_quota",
	"quota exceeded",
	"out of credits",
	"credit balance is too low",
	"429",
}

// ranOut is whether what an engine printed says its allowance is gone.
//
// Both halves are read: what the program printed and what it said on the way
// out. An engine that streams its refusal puts it in the first, one that
// dies on the request puts it in the second, and which of the two it is is a
// fact about that engine on that day.
//
// A compilation error is not running out of quota, and this is why the marks
// are phrases rather than words: "limit" alone is in every other message a
// linter prints.
func ranOut(out Result, err error) bool {
	said := strings.ToLower(out.Output)
	if err != nil {
		said += "\n" + strings.ToLower(err.Error())
	}

	for _, mark := range spentMarks {
		if strings.Contains(said, mark) {
			return true
		}
	}

	return false
}
