package settings

// Folding a group: its heading is a place the cursor can stand, and
// choosing it shows or hides the settings under it.
//
// Every group starts folded, so the screen opens as a list of what there
// is to set, and the reader opens the group they came for. Which are open
// is the screen's own and goes when it closes: a fold is a way of reading
// the table this once, not a setting.

// stop is a place the cursor can stand: a group's heading, or a row. The
// rows of a folded group are not stops, because they are not drawn.
type stop struct {
	group string
	row   int // -1 on a heading
}

// stops is every place the cursor can stand, top to bottom.
func (s State) stops(rows []Row) []stop {
	var out []stop

	for i, r := range rows {
		if headed(rows, i) {
			out = append(out, stop{group: r.Group, row: -1})
		}

		if !s.shut(r.Group) {
			out = append(out, stop{group: r.Group, row: i})
		}
	}

	return out
}

// here is which of the stops the cursor is on, and -1 when it is on none:
// a row that a fold has just hidden.
func (s State) here(stops []stop) int {
	for i, at := range stops {
		if s.head != "" && at.row < 0 && at.group == s.head {
			return i
		}

		if s.head == "" && at.row == s.sel {
			return i
		}
	}

	return -1
}

// step moves the cursor by d stops, round from one end to the other when
// wrap says so and held at the end when it does not: the arrows go round,
// the wheel stops.
func (s State) step(d int, rows []Row, wrap bool) State {
	stops := s.stops(rows)
	if len(stops) == 0 {
		return s
	}

	at := max(s.here(stops), 0) + d

	switch {
	case wrap:
		at = ((at % len(stops)) + len(stops)) % len(stops)
	default:
		at = min(max(at, 0), len(stops)-1)
	}

	return s.standOn(stops[at])
}

// standOn puts the cursor on one stop.
func (s State) standOn(at stop) State {
	if at.row < 0 {
		s.head = at.group

		return s
	}

	s.head, s.sel = "", at.row

	return s
}

// lastOf is whether row i is the last of its group, where the line down
// the side of the group ends.
func lastOf(rows []Row, i int) bool {
	return i == len(rows)-1 || rows[i+1].Group != rows[i].Group
}

// shut is whether a group's settings are hidden. A row in no group has no
// heading to open it from, so it is never hidden.
func (s State) shut(group string) bool { return group != "" && !s.opened[group] }

// counted is how many settings a group holds, which a folded heading shows.
func counted(rows []Row, group string) int {
	n := 0

	for _, r := range rows {
		if r.Group == group {
			n++
		}
	}

	return n
}
