package ui

// The repository list screen: every repository the board found, with its
// name, path relative to the root, and task counts by band. Choosing one
// filters the board to that repository.

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/e1i0r/orbit/internal/view"
)

type repolistState struct {
	sel int
}

type repoItem struct {
	name   string
	path   string
	counts [4]int
	total  int
}

func (m Model) openRepos() Model {
	m.screen = screenRepos
	m.repolist = repolistState{}

	return m
}

func (m Model) abandonRepos() Model {
	m.repolist = repolistState{}
	m.screen = screenList

	return m
}

func (m Model) collectRepos() []repoItem {
	var list []repoItem

	seen := make(map[string]int)

	if len(m.board.RepoList) > 0 {
		for _, r := range m.board.RepoList {
			idx := len(list)
			seen[strings.ToLower(r.Name)] = idx
			list = append(list, repoItem{
				name: r.Name,
				path: r.Path,
			})
		}
	}

	for _, t := range m.board.Tasks {
		idx, ok := seen[strings.ToLower(t.Repo)]
		if !ok {
			idx = len(list)
			seen[strings.ToLower(t.Repo)] = idx
			list = append(list, repoItem{
				name: t.Repo,
				path: t.RepoPath,
			})
		}

		band := view.BandOf(t)
		if band >= 0 && int(band) < len(list[idx].counts) {
			list[idx].counts[band]++
			list[idx].total++
		}
	}

	return list
}

func (m Model) repolistKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	repos := m.collectRepos()
	if len(repos) == 0 {
		if key.Matches(msg, m.keys.Back) || key.Matches(msg, m.keys.Quit) {
			return m.abandonRepos(), nil
		}

		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Back) || key.Matches(msg, m.keys.Quit):
		return m.abandonRepos(), nil
	case key.Matches(msg, m.keys.Up):
		m.repolist.sel--
		if m.repolist.sel < 0 {
			m.repolist.sel = len(repos) - 1
		}

		return m, nil
	case key.Matches(msg, m.keys.Down):
		m.repolist.sel++
		if m.repolist.sel >= len(repos) {
			m.repolist.sel = 0
		}

		return m, nil
	case key.Matches(msg, m.keys.Open), msg.Text == " ":
		p := m.opts.Words

		selected := repos[m.repolist.sel]
		if strings.EqualFold(m.repoFilter, selected.name) {
			m.repoFilter = ""
			return m.abandonRepos().say(p.T("repos.filter_cleared", "showing all repositories")), nil
		}

		m.repoFilter = selected.name

		return m.abandonRepos().say(p.T("repos.filtered", "filtered to {repo}", about("repo", selected.name))), nil
	}

	return m, nil
}

func (m Model) repolistRows(h, w int) []string {
	if h <= 0 {
		return nil
	}

	p := m.opts.Words
	out := []string{
		"",
		"  " + Paint(Accent).Render(p.T("repos.title", "Repositories")),
		"  " + Paint(Dim).Render(p.T("repos.subtitle", "choose a repository to filter the board")),
		"",
	}

	repos := m.collectRepos()
	if len(repos) == 0 {
		out = append(out, "  "+Paint(Dim).Render(p.T("repos.none", "no repositories found")))
	}

	for i, r := range repos {
		mark := strings.Repeat(" ", gutter)
		if i == m.repolist.sel {
			mark = markGlyph + strings.Repeat(" ", gutter-1)
		}

		nameRendered := Paint(Accent).Render(r.name)
		if strings.EqualFold(m.repoFilter, r.name) {
			nameRendered += " " + Paint(OK).Render(p.T("repos.active", "[filtered]"))
		}

		countsStr := fmt.Sprintf("%d todo · %d in flight · %d needs you · %d done",
			r.counts[0], r.counts[1], r.counts[2], r.counts[3])

		line := fmt.Sprintf("%s%-24s  %s  (%s)",
			mark,
			nameRendered,
			Paint(Dim).Render(r.path),
			Paint(Dim).Render(countsStr),
		)
		out = append(out, fit(line, w))
	}

	waysOut := p.T("repos.ways_out", "{open} filter · {up_down} move · {back} back",
		about("open", m.keys.Open.Help().Key),
		about("up_down", m.keys.Up.Help().Key+m.keys.Down.Help().Key),
		about("back", m.keys.Back.Help().Key))
	out = append(out, "", fit("  "+Paint(Dim).Render(waysOut), w))

	return fill(out, h)
}
