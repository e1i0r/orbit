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

func (sn Session) boardSummary() CallToolResult {
	sb, err := sn.readBoard()
	if err != nil {
		return refuse(err)
	}

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
// the repository path a follow-up call needs to name it unambiguously.
func row(t view.Task) map[string]any {
	return map[string]any{
		"id":             t.ID,
		"title":          t.Title,
		"repo":           t.Repo,
		"repo_path":      t.RepoPath,
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

// reposOf names every repository on the board with the path a caller needs
// to pass back.
func reposOf(b board.Board) []map[string]string {
	repos := make([]map[string]string, 0, len(b.RepoList))
	for _, r := range b.RepoList {
		repos = append(repos, map[string]string{"name": r.Name, "path": r.Path})
	}

	return repos
}
