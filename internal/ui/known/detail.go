package known

// One rule, opened: everything about it, and the decisions under that.
//
// Built out of the same pieces a task's overview is — the strip of figures,
// the folding sections, the actions as a label over its key — because a
// reader who has opened a task already knows how to read this.
//
// The evidence first and the decisions under it, because the whole point of
// this screen is that nothing is decided before it is read. There is no
// score: a rule that works perfectly never stops anything, so "it stopped
// the work zero times" means two opposite things and no number tells them
// apart. What is written down is the friction.

import (
	"strconv"

	"github.com/e1i0r/orbit/internal/knowledge"
	"github.com/e1i0r/orbit/internal/ui/cells"
	"github.com/e1i0r/orbit/internal/ui/prose"
	"github.com/e1i0r/orbit/internal/ui/theme"
)

// detailRows is the whole screen while one rule is open.
func (s State) detailRows(h, w int, e Env) []string {
	f, ok := s.onFact()
	if !ok {
		return cells.Fill(nil, h)
	}

	cw := content(w)

	head := s.detailHead(f, cw, e)
	foot := s.pinned(f, cw, e)

	body := s.asked(f, cw, e)
	body = append(body, prose.Strip(s.figures(f, e), cw)...)
	body = append(body, "")
	body = append(body, s.tells(f, cw, e)...)
	body = append(body, s.paused(f, cw, e)...)
	body = append(body, s.friction(cw, e)...)

	return cells.Fill(fitAll(framed(head, body, foot, h, s.deep), w), h)
}

// detailHead is the rule itself, and the line under it that says which rule
// this is: its name, how far it reaches, and where it stands.
func (s State) detailHead(f knowledge.Fact, cw int, e Env) []string {
	out := []string{""}

	for i, line := range cells.Lines(f.Phrase, cw-len(prose.Gutter)) {
		ink := theme.Text(theme.Primary)
		if i > 0 {
			ink = theme.Text(theme.Secondary)
		}

		out = append(out, prose.Gutter+ink.Render(line))
	}

	return append(out, prose.Gutter+prose.Meta(
		theme.Paint(theme.Accent).Render(cells.OrDef(f.ID, e.Words.T("knowledge.no_name_yet", "no name yet"))),
		where(f, e),
		bandName(bandOf(f), e),
	), "")
}

// asked is the block that says this rule is waiting on somebody, and which
// keys answer it. It is the board's own NEEDS YOU, in the one place it means
// the same thing.
func (s State) asked(f knowledge.Fact, cw int, e Env) []string {
	if !f.Review {
		return nil
	}

	p := e.Words

	out := []string{prose.Gutter + theme.Paint(theme.Warn).Bold(true).Render("│ "+
		p.T("knowledge.band_waiting", "WAITING ON YOU"))}

	for _, line := range cells.Lines(p.T("knowledge.asked_why",
		"it stopped you, or you paused it. Say it better with 'c', have it apply "+
			"again with 'u', or decide against it with 'o'."), cw-len(prose.Gutter)) {
		out = append(out, prose.Gutter+theme.Text(theme.Secondary).Render(line))
	}

	return append(out, "")
}

// figures is the strip: four things a reader compares at a glance rather
// than four more rows of grey label and bold value.
func (s State) figures(f knowledge.Fact, e Env) []prose.Stat {
	p := e.Words

	travels := p.T("knowledge.travels_machine", "this machine")
	if f.Scope.Repo != "" {
		travels = p.T("knowledge.travels_repo", "with the repo")
	}

	stands, role := standing(f, e)

	return []prose.Stat{
		{Label: p.T("knowledge.card_state", "right now"), Value: stands, Role: role},
		{Label: p.T("knowledge.card_said_by", "said by"), Value: shortFrom(f, e), Role: theme.Accent},
		{Label: p.T("knowledge.card_since", "since"), Value: cells.OrDef(when(f.At), "—"), Role: theme.Accent},
		{
			Label: p.T("knowledge.card_told", "told"),
			Value: p.P("knowledge.card_times", f.Used, "{n} time", "{n} times",
				about("n", strconv.Itoa(f.Used))),
			Role: theme.Live,
		},
		{Label: p.T("knowledge.card_travels", "travels"), Value: travels, Role: theme.OK},
	}
}

// standing is where the rule stands with the reader, and the colour that
// says it at a glance.
//
// A card of its own, and the loudest thing on the screen after the sentence
// itself. Somebody who has just switched a rule off and come back to look
// needs the answer in the place their eye already is, not worked out from
// which heading the row is under.
func standing(f knowledge.Fact, e Env) (string, theme.Role) {
	p := e.Words

	switch f.State {
	case knowledge.Paused:
		return p.T("knowledge.stands_paused", "paused"), theme.Warn
	case knowledge.Off:
		return p.T("knowledge.stands_off", "switched off"), theme.Bad
	}

	return p.T("knowledge.stands_active", "applying"), theme.OK
}

// tells is the section that answers what the rule does when work reaches it.
func (s State) tells(f knowledge.Fact, cw int, e Env) []string {
	p := e.Words

	out := []string{prose.Section(p.T("knowledge.sec_does", "what it does"), "", cw, true)}

	switch {
	case f.Action() == knowledge.Stops:
		out = append(out, prose.Gutter+prose.Gutter+theme.Paint(theme.Bad).Bold(true).Render(
			p.T("knowledge.does_stops", "it stops the work"))+
			theme.Paint(theme.Dim).Render(cells.Dot+p.T("knowledge.does_check", "the check is")+" ")+
			theme.Text(theme.Primary).Render(f.Check))
	case f.Stops:
		out = append(out, quoted(p.T("knowledge.does_no_check",
			"it was asked to stop the work and has no command to stop it with, "+
				"so it only says its sentence. Give it one with 'c'."), cw)...)
	default:
		out = append(out, quoted(p.T("knowledge.does_says",
			"it is put in front of the agent before every run, and the work goes ahead"), cw)...)
	}

	return append(out, "")
}

// paused is why the rule is not applying, in the words of whoever stopped it.
//
// A pause with no reason is a switch under another name, and this is what it
// was for: the sentence somebody reads when they come back.
func (s State) paused(f knowledge.Fact, cw int, e Env) []string {
	if f.Tells() {
		return nil
	}

	p := e.Words

	why := f.Why
	if why == "" {
		why = p.T("knowledge.off_why", "you decided against it. It stays where it is, "+
			"nothing is told it, and 'u' has it apply again.")
	}

	out := []string{prose.Section(p.T("knowledge.sec_paused", "why it is not applying"), "", cw, true)}

	return append(append(out, quoted(why, cw)...), "")
}

// friction is everything the rule has put somebody through.
func (s State) friction(cw int, e Env) []string {
	p := e.Words

	out := []string{prose.Section(p.T("knowledge.sec_friction", "what it has put you through"),
		"", cw, true)}

	lines := s.story(e)
	if len(lines) == 0 {
		lines = []string{p.T("knowledge.nothing_happened", "nothing has happened to this one yet")}
	}

	for _, one := range lines {
		out = append(out, quoted(one, cw)...)
	}

	return append(out, "")
}

// pinned holds the decisions against the bottom of the screen, so they are
// in the same place however long the story above them ran.
func (s State) pinned(f knowledge.Fact, cw int, e Env) []string {
	p := e.Words

	apply := prose.Field{Label: p.T("knowledge.act_pause", "pause it"), Key: "p"}
	if !f.Tells() {
		apply = prose.Field{Label: p.T("knowledge.act_resume", "apply again"), Key: "u"}
	}

	foot := prose.Fields([]prose.Field{
		{Label: p.T("knowledge.act_correct", "say it better"), Key: "c"},
		apply,
		{Label: p.T("knowledge.act_off", "switch it off"), Key: "o"},
		{Label: p.T("knowledge.act_back", "back"), Key: "esc"},
	}, 4, cw-len(prose.Gutter))

	for i, line := range foot {
		foot[i] = prose.Gutter + line
	}

	return append([]string{""}, foot...)
}

// quoted is a line of a section's body, indented under its heading.
func quoted(text string, cw int) []string {
	var out []string

	for _, line := range cells.Lines(text, cw-2*len(prose.Gutter)) {
		out = append(out, prose.Gutter+prose.Gutter+theme.Text(theme.Primary).Render(line))
	}

	return out
}

// shortFrom is where a rule came from, in the two or three words a card has
// room for.
func shortFrom(f knowledge.Fact, e Env) string {
	p := e.Words

	return map[knowledge.Source]string{
		knowledge.FromCode:       p.T("knowledge.by_code", "the code"),
		knowledge.Human:          p.T("knowledge.by_human", "you"),
		knowledge.FromRecord:     p.T("knowledge.by_record", "a model"),
		knowledge.FromProduction: p.T("knowledge.by_prod", "production"),
		knowledge.FromDocs:       p.T("knowledge.by_docs", "the project"),
		knowledge.FromHistory:    p.T("knowledge.by_history", "the history"),
	}[f.Source]
}

// fitAll cuts every row to the window, which the list does through rowsFit
// and this screen does not: its rows carry their own gutter already.
func fitAll(rows []string, w int) []string {
	for i, row := range rows {
		rows[i] = cells.Fit(row, w)
	}

	return rows
}
