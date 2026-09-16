package repo

// What a checkout already refuses work over.
//
// Every other source of a rule brings a sentence and leaves the command to
// whoever keeps it: you read "everything under money rounds half to even",
// and then you decide whether that deserves a gate and what the gate would
// run. This one is the other way round. The command already exists and is
// already running — if the project's own pull requests go red when `go vet`
// does, then `go vet ./...` is a gate the team has been keeping for years
// and nobody had to write it down.
//
// Which is why it needs no model and costs nothing. A workflow either parses
// or it does not; there is no sentence to verify against a citation and no
// judgement about whether somebody meant it. What a pull request has to pass
// is, by definition, what this project does not let through.
//
// Read by line and not by a YAML parser. The one thing wanted out of these
// files is what each step runs, which is a `run:` with a command after it —
// and a dependency taken on for one field is a dependency carried by every
// build of Orbit forever.

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// atMostGates is how many are offered out of one checkout.
//
// Six, for the reason the cold reading brings five: a tray of forty weak
// offers is a tray somebody stops opening, and they all share it. A project
// whose pull requests run twenty steps has three or four that are really the
// gate and the rest are setup, and the first few are the ones worth asking
// about.
const atMostGates = 6

// A Gate is something the checkout already runs on itself and refuses work
// over.
type Gate struct {
	// Command is what to run, spelled as the repository spells it.
	Command string
	// Where is the file it was read out of, so that whoever is deciding
	// whether to keep it can go and look at the thing that is already
	// running.
	Where string
}

// runs is a step of a workflow with a command on the same line. A `run:`
// that opens a block is a script rather than a check, and a script is not
// something to hand a gate.
var runs = regexp.MustCompile(`^\s*-?\s*run:\s*(\S.*)$`)

// uses is a step that reaches for an action, and byAction the few whose name
// says what they run. A pinned action is the same claim as a command and is
// spelled differently, so the ones worth reading are named here rather than
// guessed at.
var uses = regexp.MustCompile(`^\s*-?\s*uses:\s*([^@\s]+)`)

var byAction = map[string]string{
	"golangci/golangci-lint-action": "golangci-lint run",
	"pre-commit/action":             "pre-commit run --all-files",
	"actions/setup-go":              "",
	"actions/setup-node":            "",
	"actions/checkout":              "",
}

// Gates is what this checkout already refuses work over.
//
// Three readings, strongest first, deduplicated by command. What a pull
// request has to pass is the strongest claim there is — a team that stopped
// meaning it would have a red branch. A hook is next: it runs on every
// commit, on the machines of whoever installed it. A linter's configuration
// is last and is the weakest of the three, because a file saying how a tool
// is configured does not say anything runs it — and when something does, the
// stronger reading already brought that command and this one adds nothing.
func Gates(root string) []Gate {
	var out []Gate

	seen := map[string]bool{}

	for _, g := range append(append(fromWorkflows(root), fromHooks(root)...), fromLinters(root)...) {
		if seen[g.Command] || len(out) == atMostGates {
			continue
		}

		seen[g.Command] = true

		out = append(out, g)
	}

	return out
}

// fromWorkflows is every command a pull request has to pass.
func fromWorkflows(root string) []Gate {
	var out []Gate

	for _, file := range workflows(root) {
		out = append(out, gatesIn(root, file)...)
	}

	return out
}

// byHook is the files that say a repository runs something before a commit
// lands, and what running it means.
//
// .git/hooks is not among them, deliberately. It does not travel: a rule
// written inside the repository about a hook only this machine has is a rule
// that refuses work for everybody who clones the project and has nothing to
// run.
var byHook = []struct{ file, command string }{
	{".pre-commit-config.yaml", "pre-commit run --all-files"},
	{".pre-commit-config.yml", "pre-commit run --all-files"},
	{".husky/pre-commit", "sh .husky/pre-commit"},
	{"lefthook.yml", "lefthook run pre-commit"},
}

// fromHooks is what the repository runs before a commit lands.
func fromHooks(root string) []Gate {
	var out []Gate

	for _, hook := range byHook {
		if _, err := os.Stat(filepath.Join(root, hook.file)); err != nil {
			continue
		}

		out = append(out, Gate{Command: hook.command, Where: hook.file})
	}

	return out
}

// byLinter is the configuration files that name a linter, and the command
// that runs it.
//
// The file is the claim. A project does not carry a .golangci.yml by
// accident: somebody configured that tool for this code, and the rule is
// that the code passes it. What the file says inside — which checks are on —
// is the tool's business and not Orbit's.
var byLinter = []struct{ file, command string }{
	{".golangci.yml", "golangci-lint run"},
	{".golangci.yaml", "golangci-lint run"},
	{".golangci.toml", "golangci-lint run"},
	{"ruff.toml", "ruff check ."},
	{".ruff.toml", "ruff check ."},
	{".rubocop.yml", "rubocop"},
	{"biome.json", "npx biome check ."},
	{"eslint.config.js", "npx eslint ."},
	{"eslint.config.mjs", "npx eslint ."},
	{".eslintrc.json", "npx eslint ."},
	{".eslintrc.js", "npx eslint ."},
	{".eslintrc.yml", "npx eslint ."},
}

// fromLinters is what the repository is configured to lint with.
func fromLinters(root string) []Gate {
	var out []Gate

	for _, one := range byLinter {
		if _, err := os.Stat(filepath.Join(root, one.file)); err != nil {
			continue
		}

		out = append(out, Gate{Command: one.command, Where: one.file})
	}

	return out
}

// workflows are the files that say what a pull request has to pass, in the
// order a reader would find them.
func workflows(root string) []string {
	entries, err := os.ReadDir(filepath.Join(root, ".github", "workflows"))
	if err != nil {
		return nil
	}

	var out []string

	for _, one := range entries {
		name := one.Name()
		if one.IsDir() || (!strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml")) {
			continue
		}

		out = append(out, filepath.Join(".github", "workflows", name))
	}

	sort.Strings(out)

	return out
}

// gatesIn is what one workflow runs, and nothing at all when that workflow
// does not stand between a change and the branch.
//
// A release builds what was already agreed to and a schedule runs on nobody's
// change: neither is something a pull request has to pass, and a rule taken
// out of one would refuse work over a command that has never refused any.
func gatesIn(root, file string) []Gate {
	f, err := os.Open(filepath.Join(root, file))
	if err != nil {
		return nil
	}

	defer func() { _ = f.Close() }() //nolint:errcheck // nothing was written

	var (
		out    []Gate
		body   []string
		guards bool
	)

	lines := bufio.NewScanner(f)
	for lines.Scan() {
		body = append(body, lines.Text())

		if strings.Contains(lines.Text(), "pull_request") || strings.Contains(lines.Text(), "push:") {
			guards = true
		}
	}

	if !guards {
		return nil
	}

	for _, line := range body {
		if m := runs.FindStringSubmatch(line); m != nil {
			if one := oneCommand(m[1]); one != "" {
				out = append(out, Gate{Command: one, Where: file})
			}

			continue
		}

		if m := uses.FindStringSubmatch(line); m != nil {
			if one := byAction[m[1]]; one != "" {
				out = append(out, Gate{Command: one, Where: file})
			}
		}
	}

	return out
}

// oneCommand is the command a step runs, and nothing when the step is a
// script rather than a command.
//
// A block opened with | or > is somebody's shell program: several commands,
// a conditional, a heredoc. Handing one of those to a gate would be handing
// it something nobody wrote to be a gate, and the reader who kept it would
// find out on the next run.
func oneCommand(after string) string {
	one := strings.TrimSpace(after)
	if one == "|" || one == ">" || strings.HasPrefix(one, "|") || strings.HasPrefix(one, ">") {
		return ""
	}

	// Quoted the way YAML quotes a command with a colon in it.
	if len(one) > 1 && (one[0] == '\'' || one[0] == '"') && one[len(one)-1] == one[0] {
		one = one[1 : len(one)-1]
	}

	return strings.TrimSpace(one)
}
