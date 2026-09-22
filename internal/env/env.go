package env

// Every variable Orbit reads from the environment, named in one place.
//
// They were spelled out where they were read: "ORBIT_HOME" in the store and
// again in the words, "LINEAR_API_KEY" twice in the tracker, a provider's
// key in the quota and its base URL beside it. A name spelled in two files
// is a name that can be misspelled in one of them, and the failure that
// causes is the quiet kind — a key that is set and not found, a home
// directory read from one place and written to another.
//
// This is a list and not a reader. What each variable means is still the
// business of whoever reads it: the store decides what an empty ORBIT_HOME
// falls back to, the quota decides what a missing key says about an
// allowance. What is centralised is the spelling, and the one question
// worth asking of all of them at once — which of these is a secret.
//
// Secrets are here by name and never by value. Orbit prints its settings to
// a terminal, to a screen and into a chat, and the rule that keeps a token
// out of all three is that a token is never a setting: it is read from the
// environment, at the moment it is used, by the package that uses it.

import (
	"os"
	"strings"
)

// Where Orbit keeps its own state and how it speaks.
const (
	// Home is the state root: the record, the settings, the worktrees.
	Home = "ORBIT_HOME"
	// Lang is the language to speak, which overrides the settings file so
	// that one command can be run in another language without changing
	// what the window is in.
	Lang = "ORBIT_LANG"
	// Workspace is the directory a run treats as the workspace it may
	// reach across, for a task that joins more than one repository.
	Workspace = "ORBIT_WORKSPACE"
	// Task is the id of the task a run belongs to, handed down to the
	// engine's own process so that anything it starts can say which task
	// it was doing.
	Task = "ORBIT_TASK"
	// SupervisorEngine is the engine a supervisor is itself running under,
	// so that a task it starts is worked by the same one rather than by
	// whatever the flow names.
	SupervisorEngine = "ORBIT_SUPERVISOR_ENGINE"
)

// The keys and addresses of services Orbit talks to. Every one of these is
// read where it is used and written down nowhere.
const (
	// TelegramToken is the bot token `orbit chat` answers over Telegram
	// with. The account it answers is a setting; the token is not.
	TelegramToken = "ORBIT_TELEGRAM_TOKEN" //nolint:gosec // the name of a variable, not a token
	// LinearKey reads the body of an issue a task was written from.
	LinearKey = "LINEAR_API_KEY" //nolint:gosec // the name of a variable, not a key
	// DecisionKey is the decision engine's key: with it a run that stops
	// can be judged in half a second, without it the supervisor is what it
	// always was. See internal/hunch.
	DecisionKey = "TYPESAFE_API_KEY" //nolint:gosec // the name of a variable, not a key
	// AnthropicKey is read to tell whether claude is billed per token on
	// this machine or runs under a subscription, which decides whether an
	// allowance can be read at all.
	AnthropicKey = "ANTHROPIC_API_KEY" //nolint:gosec // the name of a variable, not a key
	// AnthropicBase and OpenAIBase are where a proxy stands, when one
	// does: the allowance is read from it rather than from the provider.
	AnthropicBase = "ANTHROPIC_BASE_URL"
	OpenAIBase    = "OPENAI_BASE_URL"
	// CodexHome is where codex keeps its own state, which is where its
	// rollouts and transcripts are read from.
	CodexHome = "CODEX_HOME"
)

// Read is one variable with the spaces taken off, which is what every
// caller of these wants: a key pasted into a shell profile arrives with a
// newline as often as not, and "  k  " is not a different key from "k".
func Read(name string) string { return strings.TrimSpace(os.Getenv(name)) }

// Set reports whether a variable has anything in it, which for a key is the
// difference between "this machine cannot" and "this machine failed".
func Set(name string) bool { return Read(name) != "" }
