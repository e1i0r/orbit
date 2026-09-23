package verb

// The settings, in groups.
//
// Eighteen of them in one flat list read as eighteen unrelated switches,
// and some cannot be understood alone: decisions and decision-floor only
// mean something read together, and so do max-running and memory-ceiling.
// So each setting belongs to one group, and every place that lists them
// lists them by group, in this order.
//
// The groups are a table of their own rather than a field on each rule,
// because where a setting is shown is a question about the list, not
// about what the setting accepts.

import (
	"slices"

	"github.com/e1i0r/orbit/internal/words"
)

// settingGroup is a heading, by its key, and the settings under it in the
// order they are shown.
type settingGroup struct {
	key     string
	members []string
}

// settingGroups is every group, in the order they are listed.
func settingGroups() []settingGroup {
	return []settingGroup{
		{"queue", []string{"max-running", "memory-ceiling", "autopilot", "unread-cap"}},
		{"tasks", []string{"engine", "model", "flow", "run-timeout"}},
		{"spending", []string{"budget-task", "budget-workspace", "quota-floor"}},
		{"decisions", []string{"decisions", "decision-floor"}},
		{"notifications", []string{"notify", "chat-id"}},
		{"appearance", []string{"language", "theme"}},
		{"maintenance", []string{"check-record"}},
	}
}

// groupTitle is a group's heading, in the reader's language.
func groupTitle(p *words.Printer, key string) string {
	switch key {
	case "queue":
		return p.T("settings.group_queue", "Queue")
	case "tasks":
		return p.T("settings.group_tasks", "Tasks")
	case "spending":
		return p.T("settings.group_spending", "Spending")
	case "decisions":
		return p.T("settings.group_decisions", "Decisions")
	case "notifications":
		return p.T("settings.group_notifications", "Notifications")
	case "appearance":
		return p.T("settings.group_appearance", "Appearance")
	default:
		return p.T("settings.group_maintenance", "Maintenance")
	}
}

// grouped is the settings in group order, each carrying its group's title.
// A setting no group names goes last, under no heading: a new setting shows
// up rather than vanishing, and a test fails until it is placed.
func grouped(p *words.Printer, all []Setting) []Setting {
	out := make([]Setting, 0, len(all))

	for _, g := range settingGroups() {
		for _, name := range g.members {
			if i := slices.IndexFunc(all, func(s Setting) bool { return s.Name == name }); i >= 0 {
				one := all[i]
				one.Group = groupTitle(p, g.key)
				out = append(out, one)
			}
		}
	}

	for _, s := range all {
		if !slices.ContainsFunc(out, func(o Setting) bool { return o.Name == s.Name }) {
			out = append(out, s)
		}
	}

	return out
}
