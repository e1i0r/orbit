package arch

// The doors of every package, which is also the index of the repository:
// reading a line here tells you what that package is asked for, and where to
// look, without opening a file.
//
// A package with one door is a package with one subject. A package with a
// dozen is one nobody has taken apart yet, and saying so here is the first
// step: the list is what makes the next export a decision instead of a
// habit, because a name that appears anywhere else fails the build.

var doors = map[string][]string{
	// The window. Twelve of its files export, and the rest are the Model's
	// own methods, which are lowercase and stay that way. The screens are
	// leaving one at a time — each one that goes takes names off this line.
	"internal/ui": {
		"bytes.go", "engines.go", "mouse.go", "plain.go", "port.go", "portinfo.go",
		"portread.go", "screen.go", "target.go", "ui.go", "update.go", "watch.go",
	},
	// The window's own packages: each is one subject, entered by an action
	// or by the vocabulary its actions share.
	"internal/ui/cells": {"cells.go", "marks.go"},
	// The flow designer: one door per thing the window asks of it, and the
	// vocabulary they share. Everything else in there is the workings of
	// one of them.
	"internal/ui/flows": {
		"flows.go", "state.go", "asked.go", "draw.go", "form_key.go", "mouse.go", "say.go",
		"template.go", "builder_rows.go", "detail_view.go", "template_builtin.go",
	},
	// The sheet that says what every key does: one door, one subject.
	"internal/ui/cheat": {"cheat.go"},
	"internal/ui/clip":  {"clip.go"},
	// The form a task is written into: one door per thing it is asked for.
	// compose.go is the form and its keyboard, click.go is the pointer,
	// submit.go is the task going out, read.go is the issue coming in, and
	// view.go is it drawn. The rest of the directory — the boxes, the
	// pills, the layout, the caret's arithmetic — is the workings of one of
	// those.
	"internal/ui/compose": {"compose.go", "click.go", "submit.go", "read.go", "view.go"},
	// The engine and model knobs: two doors. engines.go is the screen —
	// State, Env, Out, the dials it holds and what it draws — and key.go is
	// every gesture that turns one. draw.go, rows.go, fold.go and filter.go
	// are how the list is built, set out, folded and cut down.
	"internal/ui/engines": {"engines.go", "key.go"},
	// A piece of knowledge as the window names it: how far it reaches, and
	// what a repository is called.
	"internal/ui/fact": {"fact.go"},
	// How a block of markdown is set. Render is the door; Inline, Well and
	// Plain are the pieces of it a pane needs on its own.
	"internal/ui/markdown": {"markdown.go", "inline.go"},
	"internal/ui/keymap":   {"affordance.go", "keys.go", "why.go"},
	// What Orbit knows, read whole. One door: the list is one subject, and
	// draw.go is how it is set out.
	"internal/ui/known":  {"known.go"},
	"internal/ui/layout": {"columns.go", "frame.go", "repocell.go"},
	// What can be done to the thing under the pointer: three doors. menu.go
	// is the screen — State, Env, Out, and what it is a menu of — view.go is
	// it drawn and what a click lands on, and key.go is every gesture that
	// moves the cursor or chooses a row. entries.go and draw.go are how the
	// list is built and one row of it set out.
	"internal/ui/menu":  {"menu.go", "view.go", "key.go"},
	"internal/ui/patch": {"patch.go", "rationale.go"},
	// The body of the task screen: one door per pane, named after the pane
	// it draws, plus panes.go for the world they are all handed. A pane's
	// own workings — the rows of a gate, the blocks of the thinking — stay
	// in its file.
	"internal/ui/panes": {
		"panes.go", "cost.go", "gates.go", "notes.go", "refused.go", "report.go", "thinking.go",
		"timeline.go",
	},
	"internal/ui/point": {"point.go"},
	// How a block of text is set, and the shapes a screen is built out of:
	// prose.go is the typography and parts.go the assemblies.
	"internal/ui/prose":  {"prose.go", "parts.go"},
	"internal/ui/prompt": {"deliver.go", "flowdraft.go", "phase.go"},
	// The engines and their quota, as the ports answer: one file, because
	// it is a vocabulary and not an action.
	"internal/ui/roster":   {"roster.go"},
	"internal/ui/settings": {"apply.go", "key.go", "rows.go", "settings.go", "view.go"},
	"internal/ui/spoken":   {"spoken.go"},
	// The supervisor's screen: two doors. supervisor.go is the screen
	// itself — State, Env, Out, opening it, reading the record, drawing it
	// — and keys.go is everything that puts a line in the thread: a
	// keystroke, the wheel, and the sentence a delivery key sends. What is
	// left in the directory — the conversations, the offers over a
	// half-typed word, the column of what Orbit knows — is a satellite of
	// one of them.
	"internal/ui/supervisor": {"supervisor.go", "keys.go"},
	"internal/ui/theme":      {"badge.go", "syntax.go", "theme.go", "tokens.go"},
	"internal/ui/typing":     {"field.go", "paint.go", "select.go", "wrap.go"},
	"internal/ui/upgrade":    {"upgrade.go"},

	// The command line: eight doors under forty-seven files, because a
	// command is a function in a table and the table is commands.go.
	"internal/cli": {
		"cli.go", "commands.go", "critical.go", "engines.go", "set.go", "settings.go",
		"top.go", "version.go",
	},

	// What a task is and what running one does. Twenty-one doors is a
	// package that has grown a lot of verbs; it is the next one to take
	// apart, and until then this line is the list of them.
	"internal/task": {
		"alive.go", "cancel.go", "control.go", "critical.go", "decision.go", "delete.go",
		"deliver.go", "dependency.go", "dialogue.go", "direct.go", "gate.go", "join.go",
		"note.go", "read.go", "reconcile.go", "requeue.go", "review.go", "run.go",
		"start.go", "story.go", "task.go",
	},
	// One file per engine, plus the stream and transcript each one answers
	// in. The shape is the subject: adding an engine is adding three files
	// and a line here.
	"internal/engine": {
		"agy.go", "agystream.go", "claude.go", "claudetranscript.go", "codex.go",
		"codexstream.go", "codextranscript.go", "engine.go", "fake.go", "opencode.go",
		"opencodestream.go", "opencodetranscript.go", "permission.go", "prompt.go",
		"stream.go", "transcript.go",
	},

	// The record and what reads it.
	"internal/record": {
		"append.go", "conversation.go", "event.go", "kind.go", "read.go", "retract.go",
		"tail.go", "write.go",
	},
	"internal/db": {"append.go", "check.go", "db.go", "follow.go", "message.go", "read.go", "repo.go"},
	"internal/store": {
		"atomic.go", "control.go", "create.go", "flatten.go", "record.go", "repos.go",
		"run.go", "settings.go", "store.go", "tasks.go",
	},
	"internal/view": {
		"delta.go", "digest.go", "entrykind.go", "fields.go", "file.go", "fold.go", "log.go",
		"reason.go", "stuck.go", "supervisor.go", "task.go", "walk.go",
	},

	// The rest: each of these is one subject already.
	"internal/board":      {"board.go", "files.go", "filetext.go", "health.go", "log.go", "refresh.go", "supervisor.go"},
	"internal/export":     {"export.go"},
	"internal/flow":       {"draft.go", "flow.go", "load.go", "loop.go", "resolve.go", "save.go"},
	"internal/knowledge":  {"fact.go", "scope.go", "store.go"},
	"internal/logger":     {"logger.go", "openfiles.go"},
	"internal/mcp":        {"clients.go", "handlers.go", "install.go", "launch.go", "server.go", "session.go", "tools.go", "types.go"},
	"internal/migrate":    {"migrate.go"},
	"internal/quota":      {"billing.go", "codex.go", "quota.go", "source.go"},
	"internal/repo":       {"cochange.go", "compare.go", "discover.go", "impact.go", "repo.go", "review.go", "workspace.go", "worktree.go", "worktree_deliver.go", "worktree_diff.go"},
	"internal/supervisor": {"conversation.go", "happened.go", "supervise.go", "thread.go"},
	"internal/tracker":    {"linear.go", "provider.go", "providers.go", "read.go", "tracker.go"},
	"internal/words":      {"load.go", "locale.go", "words.go"},
}
