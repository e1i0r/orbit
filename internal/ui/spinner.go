package ui

// The one moving thing on screen, and the clock behind it.
//
// Orbit's screens are otherwise still: a board that redraws twice a second
// looks identical twice a second, and a reader cannot tell a run that is
// working from a run that is wedged by looking at it. The spinner is the
// answer to that and it is deliberately the only answer — one glyph, one
// rhythm, everywhere something is in motion — because two different
// animations are two different claims about what "busy" means.

import (
	tea "charm.land/bubbletea/v2"
)

// spinnerFrames is a braille cycle rather than the usual /-\| because every
// frame is one cell wide and the same weight, so nothing beside it shifts as
// it turns.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// spin is the frame for right now.
//
// It is a function of the clock rather than of a counter kept in the model,
// so every spinner on screen turns together and none of them is left on a
// stale frame by a redraw that did not go through the tick.
func (m Model) spin() string {
	return spinnerFrames[int(m.now.UnixMilli()/120)%len(spinnerFrames)]
}

// spinner is the frame painted and followed by the space that separates it
// from whatever it is spinning beside — two cells, always, so a label with a
// spinner in front of it sits exactly where the same label behind a static
// glyph used to.
func (m Model) spinner(r Role) string {
	return Paint(r).Render(m.spin()) + " "
}

// moving is whether anything on screen is actually in motion, and it is what
// decides whether the frame clock runs at all.
//
// A window watching nothing must not wake ten times a second to redraw a
// picture that has not changed. So the clock is not a standing ticker: it is
// started when there is something to animate and stops itself when there is
// not, which is why every place that can begin motion asks for the next
// frame rather than assuming one is coming.
func (m Model) moving() bool {
	return m.supervisorBusy || m.anyWorking() || m.delivering.verb != "" || m.watching != nil
}

// anyWorking is whether any run on the board is inside a phase right now.
//
// working (affordance.go) is the question and not Live, because Live only
// says a process is believed to hold the task: a run held by the reader or
// waiting at a gate still has its process, and a spinner over it would say
// the opposite of what the pause said. The affordances already draw that
// line to decide what r lets go of; the spinner asks the same one so that
// what the screen animates and what the keys act on cannot drift apart.
func (m Model) anyWorking() bool {
	for _, t := range m.board.Tasks {
		if working(t) {
			return true
		}
	}

	return false
}

// runGlyph is the mark on anything that says a run is under way: the spinner
// while one actually is, and the static bolt when it is not — a band with a
// paused run in it, or one whose process is gone.
//
// The fallback is the point. A spinner drawn while the frame clock is
// stopped is frozen on whatever frame it stopped at, and a frozen spinner
// reads as a wedged run rather than as an idle one. Nothing spins unless
// moving() is also true, so the picture and the clock can never disagree.
func (m Model) runGlyph(spinning bool) string {
	if spinning {
		return m.spin() + " "
	}

	return "⚡ "
}

// nextFrame asks for the next animation frame, and only when one is not
// already on its way.
//
// The guard is the point. Bubble Tea's tick is a one-shot, so an animation
// keeps itself going by asking for the next frame from the last one — and a
// second asker piles a second chain on top of the first, which never ends
// and doubles the frame rate every time it happens. spinning says a frame is
// already coming; the tick clears it as it arrives.
func (m Model) nextFrame() (Model, tea.Cmd) {
	if m.spinning || !m.moving() {
		return m, nil
	}

	m.spinning = true

	return m, spinnerTick()
}
