package arch

// Every verb, offered by every way in.
//
// Orbit has four: the command line, the window, the browser and the MCP
// server. internal/verb is where the vocabulary is declared; this is where
// the four are held to it.
//
// The rule is not that every verb must be everywhere. It is that a way in
// that does not offer one has to say why, here, in a sentence somebody can
// argue with — because the differences that were there before this test
// existed were not decisions. Nobody chose that the MCP server could start a
// stopped task and the browser could not; it is what four lists drifting
// looks like.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e1i0r/orbit/internal/verb"
)

// notThere is a verb one way in does not offer, and the reason it does not.
//
// The key is "<way in>:<verb>". A verb missing from a surface with no line
// here fails, and a line here for a verb that is offered fails too: a
// stale excuse reads as a decision somebody made.
var notThere = map[string]string{
	// The window is a terminal, and a terminal is where a person already
	// is: what it cannot do is what needs a second program in front of it.
	"window:learn": "the knowledge screen writes facts through its own port, not as a task verb",
	"window:reconcile": "opening the window reconciles every task in the state root, " +
		"so there is nothing left for a gesture to ask for",
	"window:export": "it writes the record into a directory the reader names, " +
		"and the window has nowhere to type a path that is not a task's",
	"window:retract": "it points at a line by its number in a listing, and the window " +
		"draws the thread as a conversation rather than a numbered list — a number " +
		"typed against a screen that does not number its lines takes back whatever is there",
	"window:join": "it names the task with -task because the caller it was written for is " +
		"an engine inside a run, where the id is already in the environment; the menu " +
		"passes a task positionally, the way every other verb about one takes it",

	// The MCP server is spoken to by a model, and these are the four a
	// model has no business asking for on its own — plus the one it could
	// not do if it wanted to.
	"mcp:pr":       "opening a pull request is a person's decision, not a model's",
	"mcp:merge":    "merging is a person's decision, not a model's",
	"mcp:close-pr": "closing a pull request is a person's decision, not a model's",
	"mcp:approve":  "accepting a library a task reached for is the question the gate asked a person",
	"mcp:take":     "this hands a terminal to an engine, and a tool call has no terminal to hand over",
}

// TestEveryVerbIsOfferedByEveryWayIn.
func TestEveryVerbIsOfferedByEveryWayIn(t *testing.T) {
	offers := map[string]map[string]bool{
		"cli":    cliOffers(t),
		"web":    webOffers(t),
		"mcp":    mcpOffers(t),
		"window": windowOffers(t),
	}

	for wayIn, offered := range offers {
		for _, v := range verb.Every() {
			key := wayIn + ":" + v.Name
			why, excused := notThere[key]

			switch {
			case offered[v.Name] && excused:
				t.Errorf("%s offers %q, and %q says it does not — drop the line", wayIn, v.Name, key)
			case !offered[v.Name] && !excused:
				t.Errorf("%s does not offer %q; give it a way in, or write down why not in notThere",
					wayIn, v.Name)
			case !offered[v.Name] && why == "":
				t.Errorf("%q is excused with no reason", key)
			}
		}
	}
}

// cliOffers is what the command line answers to.
//
// It builds a command for every declared verb it has not written one for by
// hand, so it offers the whole vocabulary by construction — which is a
// stronger thing than this test could check by reading. What is checked is
// that it still does that.
func cliOffers(t *testing.T) map[string]bool {
	t.Helper()

	if !strings.Contains(read(t, "internal/cli"), "withVerbs(") {
		t.Error("internal/cli no longer builds its commands from the declaration")

		return map[string]bool{}
	}

	return all()
}

// webOffers is what the browser asks for.
//
// One route takes any verb by name and one takes any reading, so the browser
// offers the whole vocabulary by construction the same way the command line
// does. The screens with a shape of their own still have routes of their
// own; those are renderings, not a second vocabulary.
func webOffers(t *testing.T) map[string]bool {
	t.Helper()

	body := read(t, "internal/web")
	if !strings.Contains(body, `POST /api/tasks/{id}/{verb}`) ||
		!strings.Contains(body, `GET /api/read/{verb}`) {
		t.Error("internal/web no longer routes every verb by name")

		return map[string]bool{}
	}

	return all()
}

// mcpOffers is what the MCP server names as a tool.
//
// It builds a tool for every declared verb its hand-written ones do not
// already carry, so it offers the whole vocabulary by construction. The
// hand-written names stay: a model that has learned orbit_retry_task should
// not have to learn it again for a rename that changes nothing. Which verb
// each of them is is written down in mcp's own spelledAs, beside the tools.
func mcpOffers(t *testing.T) map[string]bool {
	t.Helper()

	body := read(t, "internal/mcp")
	if !strings.Contains(body, "verbTools()") {
		t.Error("internal/mcp no longer builds its tools from the declaration")

		return map[string]bool{}
	}

	// Everything, less the handful it refuses on purpose. Those are read
	// off its own list rather than assumed from notThere below, so that
	// the two have to agree: a verb quietly dropped there fails here.
	offered := all()
	for _, name := range namesIn(body, "var cannot = map[string]string{") {
		delete(offered, name)
	}

	return offered
}

// namesIn is the keys of a map literal in a package's source, from the line
// that opens it to the brace that closes it.
func namesIn(body, opens string) []string {
	at := strings.Index(body, opens)
	if at < 0 {
		return nil
	}

	block, _, _ := strings.Cut(body[at+len(opens):], "\n}")

	var out []string

	for _, line := range strings.Split(block, "\n") {
		key, _, found := strings.Cut(strings.TrimSpace(line), ":")
		if !found || !strings.HasPrefix(key, `"`) {
			continue
		}

		out = append(out, strings.Trim(key, `"`))
	}

	return out
}

// windowOffers is what the cockpit has a key, a tab or a menu row for.
func windowOffers(t *testing.T) map[string]bool {
	t.Helper()

	return sees(read(t, "internal/ui"), map[string]string{
		"new":       "key.compose",
		"run":       "key.start",
		"read":      "key.read",
		"delete":    "key.delete_task",
		"take":      "key.take",
		"continue":  "key.hand",
		"say":       "key.supervisor",
		"thread":    "key.supervisor",
		"knowledge": "key.knowledge",
		"engines":   "key.engines",
		"quota":     "key.quota",
		"flows":     "key.flows",
		"repos":     "key.repos",
		"requeue":   "key.requeue",
		"settings":  "screenSettings",
		"list":      "screenList",
		"show":      "key.open",
		"history":   "tab.history",
		"tree":      "tab.map",
		"compare":   "compare.running",
		"flow":      "tab.flow",
		"diff":      "tab.diff",
		"impact":    "tab.impact",
		"direct":    `{name: "direct", says: true}`,
		"note":      `{name: "note", says: true}`,
	})
}

// all is every verb, offered — what a surface derived from the declaration
// answers.
func all() map[string]bool {
	out := map[string]bool{}
	for _, v := range verb.Every() {
		out[v.Name] = true
	}

	return out
}

// sees is which verbs a body of source mentions: by the mark given for it,
// or by its own name where no mark was.
func sees(body string, marks map[string]string) map[string]bool {
	out := map[string]bool{}

	for _, v := range verb.Every() {
		mark, given := marks[v.Name]
		if !given {
			mark = `"` + v.Name + `"`
		}

		out[v.Name] = strings.Contains(body, mark)
	}

	return out
}

// read is every Go file of one package, joined.
// readAll is every file of one suffix under a directory, joined.
func readAll(t *testing.T, dir, suffix string) string {
	t.Helper()

	var b strings.Builder

	where := filepath.Join(root(t), dir)

	err := filepath.WalkDir(where, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, suffix) {
			return err
		}

		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		b.Write(raw)
		b.WriteString("\n")

		return nil
	})
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}

	return b.String()
}

func read(t *testing.T, dir string) string {
	t.Helper()

	var b strings.Builder

	where := filepath.Join(root(t), dir)

	err := filepath.WalkDir(where, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}

		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		b.Write(raw)

		return nil
	})
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}

	return b.String()
}
