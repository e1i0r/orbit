// Package tracker turns an issue tracker URL a reader pasted into a task
// description, through a registry of providers — Linear, Jira, GitHub,
// GitLab — that each recognise their own URLs and write their own prompt.
//
// The sentence above is a doc comment because it sits above the package
// clause. It used to sit below it, here and in tracker.go both, which makes
// it an ordinary comment attached to nothing: `go doc` showed this package
// with no description at all, twice over.
package tracker

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Provider defines the extensible contract for any issue tracking platform.
type Provider interface {
	Name() string
	Match(rawURL string) bool
	Parse(rawURL string) (Issue, error)
	FormatPrompt(issue Issue) string
}

// Issue represents a normalized issue tracker reference.
type Issue struct {
	Kind        string // "linear", "jira", "github", "gitlab", etc.
	ID          string // issue identifier (e.g. "ENG-123", "PROJ-456", "GH-42")
	Title       string // extracted title or clean slug
	Description string // issue description if available
	Org         string // workspace or organization name
	Repo        string // repository name if applicable
	RawURL      string // original URL provided by the user
}

// registryMu guards registry.
//
// Register appends to a package-level slice and every reader below ranges
// over it. Nothing serialised the two, so a provider registered while a
// paste was being parsed is a data race — the kind the race detector calls
// and the kind that, unluckily, hands a reader of the slice a header that no
// longer describes the array behind it.
var registryMu sync.RWMutex

var registry = []Provider{
	LinearProvider{},
	JiraProvider{},
	GitHubProvider{},
	GitLabProvider{},
}

// Register registers a new tracker provider.
func Register(p Provider) {
	registryMu.Lock()
	defer registryMu.Unlock()

	registry = append(registry, p)
}

// Providers returns a copy of all registered tracker providers.
func Providers() []Provider {
	registryMu.RLock()
	defer registryMu.RUnlock()

	out := make([]Provider, len(registry))
	copy(out, registry)

	return out
}

// Parse finds the matching provider for rawURL and parses the issue.
func Parse(rawURL string) (Issue, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return Issue{}, errors.New("empty url")
	}

	for _, p := range Providers() {
		if p.Match(trimmed) {
			return p.Parse(trimmed)
		}
	}

	return Issue{}, fmt.Errorf("no tracker provider matched url: %s", rawURL)
}

// FormatPrompt generates the task description prompt with MCP hints.
func FormatPrompt(issue Issue) string {
	for _, p := range Providers() {
		if strings.EqualFold(p.Name(), issue.Kind) {
			return p.FormatPrompt(issue)
		}
	}

	return defaultPrompt(issue)
}

func defaultPrompt(iss Issue) string {
	var b strings.Builder
	if iss.Title != "" {
		b.WriteString(iss.Title)
		b.WriteString("\n\n")
	}

	b.WriteString("Issue Tracker: ")
	b.WriteString(strings.ToUpper(iss.Kind))
	b.WriteString(" (")
	b.WriteString(iss.ID)
	b.WriteString(")\nURL: ")
	b.WriteString(iss.RawURL)
	b.WriteString("\n\nPlease inspect the issue details using available MCP tools and implement the requested changes.")

	return b.String()
}
