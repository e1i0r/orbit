package tracker

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

var (
	linearPattern = regexp.MustCompile(`(?i)linear\.app/([^/]+)/issue/([A-Za-z0-9]+-[0-9]+)(?:/([^/?#]+))?`)
	jiraPattern   = regexp.MustCompile(`(?i)([a-zA-Z0-9.-]+\.atlassian\.net|jira\.[a-zA-Z0-9.-]+)/browse/([A-Za-z0-9]+-[0-9]+)`)
	githubPattern = regexp.MustCompile(`(?i)github\.com/([^/]+)/([^/]+)/issues/([0-9]+)`)
	gitlabPattern = regexp.MustCompile(`(?i)gitlab\.com/([^/]+)/([^/]+)/-/issues/([0-9]+)`)
)

// LinearProvider handles linear.app URLs.
type LinearProvider struct{}

// Name returns the provider identifier.
func (LinearProvider) Name() string { return "linear" }

// Match reports whether the URL belongs to Linear.
func (LinearProvider) Match(rawURL string) bool {
	return strings.Contains(strings.ToLower(rawURL), "linear.app")
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
func (JiraProvider) Match(rawURL string) bool {
	lower := strings.ToLower(rawURL)
	return strings.Contains(lower, ".atlassian.net") || strings.Contains(lower, "jira.")
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
	return strings.Contains(strings.ToLower(rawURL), "github.com") && strings.Contains(rawURL, "/issues/")
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
	return strings.Contains(strings.ToLower(rawURL), "gitlab.com") && strings.Contains(rawURL, "/issues/")
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
