package mcp

import (
	"fmt"
	"strings"

	"github.com/e1i0r/orbit/internal/board"
	"github.com/e1i0r/orbit/internal/store"
	"github.com/e1i0r/orbit/internal/view"
)

// Call runs one tool and answers with the result the protocol carries back.
//
// A tool that refuses is not a JSON-RPC error. The distinction the protocol
// draws is between the call failing to happen — which is an error object the
// client handles — and the call happening and the answer being "no", which
// is a result with isError set that the model reads and can act on. A task
// id nobody recognises is the second kind, so it comes back as text the
// model can correct rather than as a transport fault it cannot see.
func (sn Session) Call(name string, args map[string]any) CallToolResult {
	return guard(name, args, func() CallToolResult { return sn.answer(name, args) })
}

// answer is the tool itself, without the two lines the log takes. It is
// split out so that the switch below stays the whole of what this server
// does, and so that no tool can be added without being written down.
func (sn Session) answer(name string, args map[string]any) CallToolResult {
	switch name {
	case "orbit_get_board_summary":
		return sn.boardSummary()
	case "orbit_list_tasks":
		return sn.listTasks(args)
	case "orbit_inspect_task":
		return sn.inspectTask(args)
	case "orbit_create_task":
		return sn.createTask(args)
	case "orbit_retry_task":
		return sn.retryTask(args)
	case "orbit_add_note":
		return sn.addNote(args)
	case "orbit_pause_task":
		return sn.control(args, "pause", "paused")
	case "orbit_direct_task":
		return sn.directTask(args)
	case "orbit_cancel_task":
		return sn.cancelTask(args)
	case "orbit_requeue_task":
		return sn.requeueTask(args)
	case "orbit_list_flows":
		return sn.listFlows()
	case "orbit_get_flow":
		return sn.getFlow(args)
	case "orbit_save_flow":
		return sn.saveFlow(args)
	case "orbit_delete_flow":
		return sn.deleteFlow(args)
	case "orbit_list_repos":
		return sn.listRepos()
	case "orbit_inspect_repo":
		return sn.inspectRepo(args)
	case "orbit_add_repo":
		return sn.addRepo(args)
	case "orbit_forget_repo":
		return sn.forgetRepo(args)
	case "orbit_learn":
		return sn.learn(args)
	case "orbit_knowledge":
		return sn.knowledgeOf(args)
	case "orbit_supervisor_say":
		return sn.supervisorSay(args)
	case "orbit_supervisor_history":
		return sn.supervisorHistory(args)
	default:
		return refuse(fmt.Errorf("no tool is called %q; the tools are %s", name, strings.Join(toolNames(), ", ")))
	}
}

// readBoard is the opening move of every tool: the state root and one folded
// board, or the sentence saying why there is neither.
func (sn Session) readBoard() (*storeAndBoard, error) {
	s, err := sn.open()
	if err != nil {
		return nil, err
	}

	b, roots, err := sn.board(s)
	if err != nil {
		// Nothing is handed back on this path, so nothing else can close
		// what was opened a moment ago.
		_ = s.Close() //nolint:errcheck // the failure being reported is the worse one

		return nil, fmt.Errorf("read the board: %w", err)
	}

	return &storeAndBoard{store: s, board: b, roots: roots}, nil
}

// storeAndBoard is what a tool needs to answer anything: somewhere to write
// and the folded record it is writing against.
type storeAndBoard struct {
	store *store.Store
	board board.Board
	// roots is where the board above was folded from. It is carried rather
	// than asked for a second time: the version this replaces reopened the
	// state root to answer "where did you look?", and answered nil — an
	// empty list, indistinguishable from having looked nowhere — when that
	// second attempt failed.
	roots []string
}

// close gives the record back.
//
// Every tool call opens the state root for itself, and opening it opens the
// record: one SQLite connection, which is the database file, its write-ahead
// log and its shared-memory index. A server answers for as long as the
// client that spawned it runs, so a call that does not close leaves those
// open for the life of the process — and enough calls take the machine's own
// file table with them, after which nothing on it can open a file: not the
// next tool call, not the window's run markers, not git.
func (sb *storeAndBoard) close() {
	if sb == nil || sb.store == nil {
		return
	}

	_ = sb.store.Close() //nolint:errcheck // the answer is already made and there is nobody left to tell
}

func (sn Session) boardSummary() CallToolResult {
	sb, err := sn.readBoard()
	if err != nil {
		return refuse(err)
	}

	defer sb.close()

	b := sb.board

	counts := map[string]int{}
	for _, band := range view.Bands() {
		counts[bandSlug(band)] = b.Counts[band]
	}

	spend := 0.0
	for _, t := range b.Tasks {
		spend += t.Cost
	}

	return reply(map[string]any{
		"repos":       reposOf(b),
		"repos_count": len(b.RepoList),
		"counts":      counts,
		"tasks_total": len(b.Tasks),
		"unread":      board.Unread(b),
		"spend":       spend,
		"roots":       sb.roots,
	})
}

func (sn Session) listTasks(args map[string]any) CallToolResult {
	bandFilter := stringArg(args, "band")

	var want view.Band

	if bandFilter != "" {
		parsed, err := parseBand(bandFilter)
		if err != nil {
			return refuse(err)
		}

		want = parsed
	}

	sb, err := sn.readBoard()
	if err != nil {
		return refuse(err)
	}

	defer sb.close()

	repoFilter := stringArg(args, "repo")

	rows := make([]map[string]any, 0, len(sb.board.Tasks))
	for _, t := range sb.board.Tasks {
		if bandFilter != "" && t.Band != want {
			continue
		}

		if repoFilter != "" && !sameRepo(sb.board, t, repoFilter) {
			continue
		}

		rows = append(rows, row(t))
	}

	return reply(map[string]any{"tasks": rows, "count": len(rows)})
}

// row is one task as a tool reports it: the columns the window draws, and
// where the work went.
//
// Where it went is repos and not repo. A task reaches into as many
// checkouts as the work needed, and a report that named one of them would be
// telling a supervisor that the other three do not exist — it would read the
// row, act on the one repository it was shown, and account for a change it
// never saw. repo and repo_path stay beside it as the home checkout: the one
// the worktree and the diff are opened from, and the first the task joined.
func row(t view.Task) map[string]any {
	return map[string]any{
		"id":             t.ID,
		"title":          t.Title,
		"repo":           t.Repo,
		"repo_path":      t.RepoPath,
		"repos":          reposWorked(t),
		"band":           bandSlug(t.Band),
		"flow":           t.Flow,
		"phase":          t.Phase,
		"engine":         t.Engine,
		"model":          t.Model,
		"attempt":        t.Attempt,
		"live":           t.Live.String(),
		"read":           t.Read,
		"cost":           t.Cost,
		"reason":         t.Reason.Key,
		"current_action": t.CurrentAction,
	}
}

// reposWorked is every repository the task is worked in, by name and oldest
// join first, and never nil: a tool that answers `"repos": null` for a task
// leaves a reader to decide whether that means none or unknown, and a task
// on the board is worked in at least one.
func reposWorked(t view.Task) []string {
	if len(t.Repos) > 0 {
		return t.Repos
	}

	if t.Repo != "" {
		return []string{t.Repo}
	}

	return []string{}
}

// reposOf names every repository on the board with the path a caller needs
// to pass back.
func reposOf(b board.Board) []map[string]string {
	repos := make([]map[string]string, 0, len(b.RepoList))
	for _, r := range b.RepoList {
		repos = append(repos, map[string]string{"name": r.Name, "path": r.Path})
	}

	return repos
}
