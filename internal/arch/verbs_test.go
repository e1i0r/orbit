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

// chatOffers is what a chat can be asked for.
//
// Built from the declaration like the command line and the tool calls: the
// list is verb.Every() and nothing in that package writes a second one. The
// handful it turns down are read off its own list rather than assumed from
// notThere, so that the two have to agree.
func chatOffers(t *testing.T) map[string]bool {
	t.Helper()

	body := read(t, "internal/chat")
	if !strings.Contains(body, "verb.Every()") {
		t.Error("internal/chat no longer builds its commands from the declaration")

		return map[string]bool{}
	}

	offered := all()
	for _, name := range namesIn(body, "var cannot = map[string]string{") {
		delete(offered, name)
	}

	return offered
}

// TestEveryVerbIsOfferedByEveryWayIn.
func TestEveryVerbIsOfferedByEveryWayIn(t *testing.T) {
	offers := map[string]map[string]bool{
		"cli":    cliOffers(t),
		"web":    webOffers(t),
		"mcp":    mcpOffers(t),
		"window": windowOffers(t),
		"chat":   chatOffers(t),
	}

	for wayIn, offered := range offers {
		for _, v := range verb.Every() {
			name := v.Path()
			key := wayIn + ":" + name
			why, excused := notThere[key]

			switch {
			case offered[name] && excused:
				t.Errorf("%s offers %q, and %q says it does not — drop the line", wayIn, name, key)
			case !offered[name] && !excused:
				t.Errorf("%s does not offer %q; give it a way in, or write down why not in notThere",
					wayIn, name)
			case !offered[name] && why == "":
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
		!strings.Contains(body, `POST /api/do/{under}/{verb}`) ||
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
		"board":             "boardEntries",
		"board new":         "key.compose",
		"board list":        "screenList",
		"task start":        "key.start",
		"task pause":        "key.pause",
		"task resume":       "key.resume",
		"task continue":     "key.hand",
		"task skip":         "key.skip",
		"task cancel":       "key.cancel",
		"task requeue":      "key.requeue",
		"task read":         "key.read",
		"task delete":       "key.delete_task",
		"task take":         "key.take",
		"supervisor say":    "key.supervisor",
		"supervisor thread": "key.supervisor",
		"knowledge":         "key.knowledge",
		"engines":           "key.engines",
		"quota":             "key.quota",
		"flows":             "key.flows",
		"repos":             "key.repos",
		"settings":          "screenSettings",
		"task show":         "key.open",
		"task history":      "tab.history",
		"task tree":         "tab.map",
		"task compare":      "compare.running",
		"task flow":         "tab.flow",
		"task diff":         "tab.diff",
		"task prompt":       "tab.prompt",
		"task impact":       "tab.impact",
		"task direct":       `{name: "task", child: "direct", says: true}`,
		"task note":         `{name: "task", child: "note", says: true}`,
		"task approve":      "approve",
		"task permit":       "permit",
		"task critical":     "critical",
		// The tray on the knowledge screen: the sentences waiting, and the
		// two answers to one of them. The marks are the sentences the screen
		// says while offering them, because that is the offer. The tray is
		// the band they are listed under.
		"rules":      "knowledge.band_waiting",
		"rules keep": "knowledge.kept",
		"rules drop": "knowledge.left_said",
		// The review, opened on one rule with everything it has put you
		// through under it. The marks are the sentences it answers with,
		// because that is the decision landing.
		"rules review":   "knowledge.sec_friction",
		"rules enforced": "knowledge.gates_read",
		"rules correct":  "knowledge.field_where",
		"rules off":      "knowledge.switched_off",
		"rules pause":    "knowledge.pause_needs_why",
		"rules resume":   "knowledge.applies_again",
		// The window asks for these through the parent, which is what
		// carries the streaming bodies: the toolbar watches `pr` run.
		"pr merge":       `"MERGE PR"`,
		"pr close":       `"CLOSE PR"`,
		"pr update":      "updatePRBranch",
		"pr checks":      "fixChecks",
		"pr tests":       "addMoreTests",
		"pr resolve":     "resolveComments",
		"pr review":      "reviewPR",
		"settings set":   "screenSettings",
		"settings clear": "settings.back_to",
	})
}

// all is every verb, offered — what a surface derived from the declaration
// answers.
func all() map[string]bool {
	out := map[string]bool{}
	for _, v := range verb.Every() {
		out[v.Path()] = true
	}

	return out
}

// sees is which verbs a body of source mentions: by the mark given for it,
// or by its own name where no mark was.
func sees(body string, marks map[string]string) map[string]bool {
	out := map[string]bool{}

	for _, v := range verb.Every() {
		mark, given := marks[v.Path()]
		if !given {
			mark = `"` + v.Path() + `"`
		}

		out[v.Path()] = strings.Contains(body, mark)
	}

	return out
}

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

// read is every Go file of one package, joined.
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
