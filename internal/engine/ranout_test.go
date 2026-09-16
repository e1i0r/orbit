package engine

// Whether an engine stopped because its allowance ran out.

import (
	"errors"
	"testing"
)

// TestWhatAProviderSaysWhenTheAllowanceIsGone.
//
// The words come from the APIs and not from the engines: all four are
// command lines over the same handful of providers, and what they print is
// the provider's own refusal.
func TestWhatAProviderSaysWhenTheAllowanceIsGone(t *testing.T) {
	for _, said := range []string{
		"Claude AI usage limit reached, resets at 3pm",
		"Error: 429 Too Many Requests",
		"rate limit exceeded for requests",
		`{"error":{"type":"insufficient_quota"}}`,
		"Your credit balance is too low to run this request",
	} {
		if !ranOut(Result{Output: said}, nil) {
			t.Errorf("this was not read as an allowance running out: %q", said)
		}
	}
}

// TestAnEngineThatBrokeIsNotAnEngineThatRanOut.
//
// The two send somebody to do completely different things, which is the
// whole reason for telling them apart — and a compilation error that
// happened to contain the word "limit" would send them to wait for a quota
// that is not the problem.
func TestAnEngineThatBrokeIsNotAnEngineThatRanOut(t *testing.T) {
	for _, said := range []string{
		"./main.go:12:2: declared and not used: n",
		"panic: runtime error: index out of range [3] with length 2",
		"exit status 1",
		"limit: 20",
		"",
	} {
		if ranOut(Result{Output: said}, nil) {
			t.Errorf("this was read as an allowance running out: %q", said)
		}
	}
}

// TestItIsReadOffBothHalves.
//
// An engine that streams its refusal puts it in what it printed; one that
// dies on the request puts it in what it said on the way out. Which of the
// two it is is a fact about that engine on that day, so both are read.
func TestItIsReadOffBothHalves(t *testing.T) {
	if !ranOut(Result{}, errors.New("anthropic: usage limit reached")) {
		t.Error("an engine that said it on the way out was read as broken")
	}

	if !ranOut(Result{Output: "RATE LIMIT"}, nil) {
		t.Error("it is read whatever case the provider shouted it in")
	}
}

// TestEveryEngineAnswers, because the compiler is the reviewer for a new
// one: an engine added without an answer here does not build.
func TestEveryEngineAnswers(t *testing.T) {
	spent := Result{Output: "usage limit reached"}

	for name, e := range All() {
		if !e.RanOut(spent, nil) {
			t.Errorf("%s does not read a provider saying the allowance is gone", name)
		}

		if e.RanOut(Result{Output: "exit status 2"}, nil) {
			t.Errorf("%s reads a broken run as one that ran out", name)
		}
	}
}
