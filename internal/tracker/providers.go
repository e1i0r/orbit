package tracker

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// The patterns are anchored to the start of the host.
//
// Unanchored, every one of them matched its tracker's name anywhere in a
// URL, and the Match functions below asked the same question with
// strings.Contains over the whole string. So
// https://evil.example/linear.app/issue/ENG-1/x was a Linear issue: it was
// recognised as one on the paste, parsed as one, and the prompt handed to an
// engine said Linear and carried a link somewhere else entirely. The host is
// the only part of a URL that says who is answering.
var (
	linearPattern = regexp.MustCompile(`(?i)^(?:[a-z0-9-]+\.)*linear\.app/([^/]+)/issue/([A-Za-z0-9]+-[0-9]+)(?:/([^/?#]+))?`)
	jiraPattern   = regexp.MustCompile(`(?i)^([a-zA-Z0-9.-]+\.atlassian\.net|jira\.[a-zA-Z0-9.-]+)/browse/([A-Za-z0-9]+-[0-9]+)`)
	githubPattern = regexp.MustCompile(`(?i)^(?:www\.)?github\.com/([^/]+)/([^/]+)/issues/([0-9]+)`)
	gitlabPattern = regexp.MustCompile(`(?i)^(?:www\.)?gitlab\.com/([^/]+)/([^/]+)/-/issues/([0-9]+)`)
)

// hostUnder reports whether rawURL is served by domain or a subdomain of it.
//
// This is what the Match functions ask instead of searching the whole URL
// for a substring. A path, a query string and a fragment are all attacker
// controlled on a link somebody was sent; the host is not.
func hostUnder(rawURL, domain string) bool {
	u, err := normalizeURL(rawURL)
	if err != nil {
		return false
	}

	host := strings.ToLower(u.Hostname())

	return host == domain || strings.HasSuffix(host, "."+domain)
}

// pathHas reports whether rawURL's path contains seg, which the GitHub and
// GitLab providers use to tell an issue from the rest of a forge.
func pathHas(rawURL, seg string) bool {
	u, err := normalizeURL(rawURL)
	if err != nil {
		return false
	}

	return strings.Contains(u.Path, seg)
}

// LinearProvider handles linear.app URLs.
type LinearProvider struct{}

// Name returns the provider identifier.
func (LinearProvider) Name() string { return "linear" }

// Match reports whether the URL belongs to Linear.
func (LinearProvider) Match(rawURL string) bool {
	return hostUnder(rawURL, "linear.app")
}

// Parse extracts issue details from a Linear URL.
func (LinearProvider) Parse(rawURL string) (Issue, error) {
	u, err := normalizeURL(rawURL)
	if err != nil {
		return Issue{}, err
	}

	m := linearPattern.FindStringSubmatch(u.Host + u.Path)
	if len(m) < 3 {
		return Issue{}, fmt.Errorf("invalid linear url: %s", rawURL)
	}

	title := ""
	if len(m) > 3 && m[3] != "" {
		title = cleanSlug(m[3])
	}

	return Issue{
		Kind:   "linear",
		ID:     strings.ToUpper(m[2]),
		Title:  title,
		Org:    m[1],
		RawURL: rawURL,
	}, nil
}

// FormatPrompt builds the task description for a Linear issue.
func (LinearProvider) FormatPrompt(iss Issue) string {
	var b strings.Builder
	if iss.Title != "" {
		b.WriteString(iss.Title + "\n\n")
	}

	fmt.Fprintf(&b, "Issue Tracker: Linear (%s)\nURL: %s\n\n", iss.ID, iss.RawURL)
	b.WriteString("Please inspect the issue details using Linear MCP tools (e.g. linear_get_issue) or review the requirement and implement the requested changes.")

	return b.String()
}

// JiraProvider handles Atlassian Jira URLs.
type JiraProvider struct{}

// Name returns the provider identifier.
func (JiraProvider) Name() string { return "jira" }

// Match reports whether the URL belongs to Jira.
//
// Jira is the one tracker here with no single domain: a cloud instance sits
// under atlassian.net and a self-hosted one is whatever the company called
// it, so the second half is a host that begins with jira. That is a guess,
// and it is kept narrow — a host, not the string "jira." found anywhere in
// a link.
func (JiraProvider) Match(rawURL string) bool {
	u, err := normalizeURL(rawURL)
	if err != nil {
		return false
	}

	host := strings.ToLower(u.Hostname())

	return hostUnder(rawURL, "atlassian.net") || strings.HasPrefix(host, "jira.")
}

// Parse extracts issue details from a Jira URL.
func (JiraProvider) Parse(rawURL string) (Issue, error) {
	u, err := normalizeURL(rawURL)
	if err != nil {
		return Issue{}, err
	}

	m := jiraPattern.FindStringSubmatch(u.Host + u.Path)
	if len(m) < 3 {
		return Issue{}, fmt.Errorf("invalid jira url: %s", rawURL)
	}

	return Issue{
		Kind:   "jira",
		ID:     strings.ToUpper(m[2]),
		Org:    m[1],
		RawURL: rawURL,
	}, nil
}

// FormatPrompt builds the task description for a Jira issue.
func (JiraProvider) FormatPrompt(iss Issue) string {
	var b strings.Builder
	if iss.Title != "" {
		b.WriteString(iss.Title + "\n\n")
	}

	fmt.Fprintf(&b, "Issue Tracker: Jira (%s)\nURL: %s\n\n", iss.ID, iss.RawURL)
	b.WriteString("Please inspect the issue details using Jira MCP tools or review the requirement and implement the requested changes.")

	return b.String()
}

// GitHubProvider handles github.com issue URLs.
type GitHubProvider struct{}

// Name returns the provider identifier.
func (GitHubProvider) Name() string { return "github" }

// Match reports whether the URL belongs to GitHub Issues.
func (GitHubProvider) Match(rawURL string) bool {
	return hostUnder(rawURL, "github.com") && pathHas(rawURL, "/issues/")
}

// Parse extracts issue details from a GitHub Issue URL.
func (GitHubProvider) Parse(rawURL string) (Issue, error) {
	u, err := normalizeURL(rawURL)
	if err != nil {
		return Issue{}, err
	}

	m := githubPattern.FindStringSubmatch(u.Host + u.Path)
	if len(m) < 4 {
		return Issue{}, fmt.Errorf("invalid github issue url: %s", rawURL)
	}

	return Issue{
		Kind:   "github",
		ID:     "GH-" + m[3],
		Org:    m[1],
		Repo:   m[2],
		RawURL: rawURL,
	}, nil
}

// FormatPrompt builds the task description for a GitHub issue.
func (GitHubProvider) FormatPrompt(iss Issue) string {
	var b strings.Builder
	if iss.Title != "" {
		b.WriteString(iss.Title + "\n\n")
	}

	fmt.Fprintf(&b, "Issue Tracker: GitHub (#%s)\nURL: %s\n\n", strings.TrimPrefix(iss.ID, "GH-"), iss.RawURL)
	b.WriteString("Please inspect the issue details using GitHub MCP tools or `gh issue view` and implement the requested changes.")

	return b.String()
}

// GitLabProvider handles gitlab.com issue URLs.
type GitLabProvider struct{}

// Name returns the provider identifier.
func (GitLabProvider) Name() string { return "gitlab" }

// Match reports whether the URL belongs to GitLab Issues.
func (GitLabProvider) Match(rawURL string) bool {
	return hostUnder(rawURL, "gitlab.com") && pathHas(rawURL, "/issues/")
}

// Parse extracts issue details from a GitLab Issue URL.
func (GitLabProvider) Parse(rawURL string) (Issue, error) {
	u, err := normalizeURL(rawURL)
	if err != nil {
		return Issue{}, err
	}

	m := gitlabPattern.FindStringSubmatch(u.Host + u.Path)
	if len(m) < 4 {
		return Issue{}, fmt.Errorf("invalid gitlab issue url: %s", rawURL)
	}

	return Issue{
		Kind:   "gitlab",
		ID:     "GL-" + m[3],
		Org:    m[1],
		Repo:   m[2],
		RawURL: rawURL,
	}, nil
}

// FormatPrompt builds the task description for a GitLab issue.
func (GitLabProvider) FormatPrompt(iss Issue) string {
	var b strings.Builder
	if iss.Title != "" {
		b.WriteString(iss.Title + "\n\n")
	}

	fmt.Fprintf(&b, "Issue Tracker: GitLab (#%s)\nURL: %s\n\n", strings.TrimPrefix(iss.ID, "GL-"), iss.RawURL)
	b.WriteString("Please inspect the issue details using GitLab tools and implement the requested changes.")

	return b.String()
}

func normalizeURL(raw string) (*url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
		trimmed = "https://" + trimmed
	}

	return url.Parse(trimmed)
}

func cleanSlug(slug string) string {
	parts := strings.Split(slug, "-")

	var words []string

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			words = append(words, p)
		}
	}

	if len(words) == 0 {
		return ""
	}

	res := strings.Join(words, " ")

	runes := []rune(res)
	if len(runes) > 0 {
		runes[0] = unicode.ToUpper(runes[0])
	}

	return string(runes)
}
