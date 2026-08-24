// Package layout is the window's arithmetic and nothing else: where the four
// regions of the screen begin, how tall each one is, and how wide each column
// of a row may be. It draws nothing, it imports no terminal library beyond
// the one that measures a string in cells, and every answer it gives is a
// pure function of the numbers it was handed.
//
// That is the whole point. The program this replaces expressed one screen's
// geometry as several hundred literal offsets spread across the file that
// drew it — a constant here, a subtraction there — so a pane that grew by a
// row silently broke a pane nobody had touched, and the only way to find out
// was to look at a terminal. Here the offsets are one function with a table
// test on it, and a pane that grows by a row fails the build.
package layout

import "fmt"

// MinWidth is the narrowest terminal Orbit will draw a board in.
//
// Below it the window says so and stops rather than drawing whatever fits:
// a crooked table is worse than a sentence, because a reader can act on a
// sentence. Sixty is where the aligned row stops being possible at all — an
// id, a state word and an elapsed time with room between them — which is
// why it is this number and not a round one.
const MinWidth = 60

// Strip is one region of the screen, in rows and cells. Y is its first row,
// counted from the top of the terminal; H is how many rows it owns; W is how
// many cells wide it is, which is the terminal's full width for all four
// regions the window has today.
//
// There is no X, and that is why it is a strip and not a rectangle. Every
// region spans the terminal, and a region that did not would be a column
// layout — which is a different decision, and one this window has not made.
// The name Region belongs to the enum below, which is what a caller holding
// a y actually asks for.
type Strip struct {
	Y, H, W int
}

// Frame is the screen as four regions stacked from the top.
//
// The order of the fields is the order they are drawn in, and the order the
// window gives them up in as the terminal shrinks is Fit's, not this one's.
type Frame struct {
	Header Strip // product, folder, and the standing settings
	Body   Strip // the bands and their rows: the only region measured in tasks
	Band   Strip // the activity band, which never empties
	Bar    Strip // the keys that can be pressed right now
}

// TooNarrowError is a terminal below MinWidth.
//
// It carries both numbers rather than only a sentence because the window
// draws this refusal through internal/words like every other string a reader
// sees, and a translator needs the numbers loose to put them where Spanish
// puts them. Error() states it in English for a log, a test and `orbit
// --once`, in the same words the plan writes it.
type TooNarrowError struct {
	Need int // MinWidth
	Got  int // what the terminal actually offers
}

// Error states the refusal in English.
func (e TooNarrowError) Error() string {
	return fmt.Sprintf("orbit needs %d columns; this terminal has %d", e.Need, e.Got)
}

// Fit lays a terminal of w by h out as four regions, or refuses it.
//
// It is called Fit and not Frame because Go will not let a function and a
// type share a name, and Frame is the noun the window passes around —
// internal/view made the same call for the same reason when it named BandOf.
//
// The activity band's height is settled before the body's, and that is the
// decision this function encodes. The band is two rows whenever the terminal
// can afford them, so the body is what is left rather than the band being
// what is left over; the alternative — sizing the body first and giving the
// band the remainder — is how a status area ends up zero rows tall on a short
// terminal, and a status area that goes blank reads as broken. The cost is
// one row of reflow at the moment the terminal crosses a threshold, and that
// was judged the cheaper of the two.
//
// It is total in h. A height of zero, of one, or of a negative number a
// resize handler computed wrong all answer with regions that tile whatever
// height was actually given, and never with a negative one.
func Fit(w, h int) (Frame, error) {
	if w < MinWidth {
		return Frame{}, TooNarrowError{Need: MinWidth, Got: w}
	}
	header, body, band, bar := rows(h)

	var f Frame
	y := 0
	for _, region := range []struct {
		height int
		into   *Strip
	}{
		{header, &f.Header},
		{body, &f.Body},
		{band, &f.Band},
		{bar, &f.Bar},
	} {
		*region.into = Strip{Y: y, H: region.height, W: w}
		y += region.height
	}
	return f, nil
}

// rows hands out h rows in the order the frame gives them up.
//
// Read the claims forwards and it is the order rows are earned; read it
// backwards and it is the order they are surrendered as the terminal
// shrinks. Both readings are the same list, which is the reason it is a list
// and not four subtractions: an order written once cannot disagree with
// itself.
//
// The order says a window that cannot name what it is showing is worse than
// one showing less — so the header's first row is claimed before anything —
// and that a row of tasks outranks the key bar, which outranks the activity
// band, which outranks the header's rule. Everything past the frame's own
// furniture is a task.
//
// At full height that is a header of two rows — the header line and the rule
// under it — a band of two — the rule above it and the band itself — and a
// bar of one, which is where the body's h-5 comes from. Those three numbers
// are written nowhere else: they are how many times each region appears in
// the list below.
func rows(h int) (header, body, band, bar int) {
	if h < 0 {
		// A negative height is not a terminal. It is arithmetic that went
		// wrong somewhere above, and the honest answer is an empty frame
		// rather than a region with a negative height, which draws as a
		// pane that eats the one below it.
		h = 0
	}
	for _, claim := range []*int{&header, &body, &header, &bar, &band, &band} {
		if h == 0 {
			return
		}
		*claim++
		h--
	}
	body += h
	return
}

// Region names one of the frame's stacked strips. It is what a caller asks
// for when it has a row of the terminal and needs to know what is drawn
// there — a reader's pointer, a resize handler, a pane deciding whether a
// key belongs to it.
//
// Before this existed every caller re-derived the answer from the heights,
// and each one drew its own boundary: one of them treated the row under the
// body's last row as the body's, because it subtracted Y and never checked
// H. The arithmetic is the same arithmetic Fit already did, so it is
// answered here, once, from the same numbers.
type Region int

const (
	// RegionNone is a row outside the frame — above it, below it, or inside
	// a strip a short terminal could not afford any rows for. It is the
	// zero value on purpose: a caller that forgets to look at the answer
	// gets nothing rather than the header.
	RegionNone Region = iota
	RegionHeader
	RegionBody
	RegionBand
	// RegionStatus is the status line — spent, tasks, events, read time,
	// quota. It is named here and answered by nothing yet, because the
	// frame has no status strip until the task that adds one; naming it now
	// is what keeps that task from having to reopen this file and renumber
	// a constant every switch in the window is written against.
	RegionStatus
	RegionBar
)

// At says which region the row y falls in.
//
// It walks the same four strips Fit handed rows out to, in the same order,
// and answers with the first that contains y. A strip of zero rows contains
// nothing, which falls out of the arithmetic rather than needing a case of
// its own — so a terminal too short for a band is a terminal where no y is
// ever the band's.
func (f Frame) At(y int) Region {
	for _, s := range []struct {
		strip  Strip
		region Region
	}{
		{f.Header, RegionHeader},
		{f.Body, RegionBody},
		{f.Band, RegionBand},
		{f.Bar, RegionBar},
	} {
		if y >= s.strip.Y && y < s.strip.Y+s.strip.H {
			return s.region
		}
	}
	return RegionNone
}

// BodyRow is which row of the body y is, counted from zero, and whether y
// was in the body at all.
//
// This is the conversion every hit on the board needs and the one every
// caller gets wrong at the same place: the row just past the body's last is
// not the body's, and a subtraction that does not check the height answers
// "one past the end" as if it were a row — which indexes a list of tasks
// one beyond what it holds. Written here it is wrong in one place or in
// none, and it is checked through At so there is only one boundary.
func (f Frame) BodyRow(y int) (int, bool) {
	if f.At(y) != RegionBody {
		return 0, false
	}
	return y - f.Body.Y, true
}
