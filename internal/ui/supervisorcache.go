package ui

// The rows the supervisor thread was last drawn to, kept so that a redraw
// which changes nothing does not render every message again.
//
// The screen renders the whole thread and then shows the two dozen rows that
// fit, and Bubble Tea redraws on every message it delivers: the half-second
// board tick, the spinner ten times a second, and every key pressed while an
// answer is being typed. A thread of forty-three messages came to a thousand
// rows a frame and twenty-one times the cost of the board behind it — paid
// again for each letter typed into the line at the foot.

// threadCache is one rendering of the thread and the state it was made from.
//
// It hangs off the Model as a pointer because Model is a value: it is copied
// for every message the window handles, so a cache kept by value would be
// filled in by whichever copy happened to draw and thrown away with it. The
// pointer is the one thing the copies share. Nothing guards it, and nothing
// needs to — the window draws and updates on one goroutine, and the work
// that does not is a tea.Cmd that never touches a Model.
type threadCache struct {
	key    threadKey
	held   bool
	rows   []string
	starts []int
}

// threadKey is everything the rendering is a function of. What is left out
// of it is a change that would be drawn with the rows from before it — and
// the thread's own content is left out, which is why invalidate exists.
type threadKey struct {
	n       int
	width   int
	picking bool
	pick    int
	busy    bool
	theme   string
}

// rowsFor gives back the rendering when it was made for this exact state.
//
// A Model built without a cache — a zero value in a test — has none, and
// answers that it holds nothing rather than refusing to draw.
func (c *threadCache) rowsFor(k threadKey) (rows []string, starts []int, held bool) {
	if c == nil || !c.held || c.key != k {
		return nil, nil, false
	}

	return c.rows, c.starts, true
}

// keep puts one rendering away under the state it was made from.
func (c *threadCache) keep(k threadKey, rows []string, starts []int) {
	if c == nil {
		return
	}

	c.key, c.held, c.rows, c.starts = k, true, rows, starts
}

// invalidate throws the rendering away, for the changes the key cannot see:
// the thread's own turns. It is called wherever the thread is read from the
// record, which is the only place they change.
func (c *threadCache) invalidate() {
	if c == nil {
		return
	}

	c.held = false
}

// threadKeyAt is the state the thread would be rendered from at this width.
func (m Model) threadKeyAt(cw int) threadKey {
	return threadKey{
		n:       len(m.supervisor.lines),
		width:   cw,
		picking: m.supervisor.picking,
		pick:    m.supervisor.pick,
		busy:    m.supervisorBusy,
		// The palette is chosen at run time and every row carries its
		// colours, so a thread drawn before the theme changed is a thread
		// in the old one.
		theme: CurrentTheme(),
	}
}
