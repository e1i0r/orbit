package known

// The shape of one row of the tray: a sentence somebody said that read as a
// rule, waiting to be told whether it was one.
//
// A file of its own because known.go carries the screen and the world it was
// written against, and the two together went over the ceiling. said.go is
// what the tray *does*; this is what one of its rows *is*.

import "time"

// A Said is one sentence somebody said to the supervisor that read as a
// rule, waiting to be told whether it was one.
//
// The screen's own shape, and not the shape of the package that holds the
// tray: what reaches the window is data, through a port, here as everywhere
// else.
type Said struct {
	// At is when it was said, and it is the sentence's name: the thread is
	// append-only and no two turns share an instant.
	At   time.Time
	Text string
	// From is where it was said: the task it was typed at, or the way in it
	// came through when it was about no task. It is what a reader needs
	// before they can agree with anything — the same words said while
	// correcting one run and said to the supervisor are the same rule, and
	// which it was is how somebody decides whether it was meant that widely.
	From string
	// Where is the folder the work was in when it was said, relative to the
	// checkout it came out of, and empty when it came out of no one folder.
	// It is what the editor's place line opens with: the commonest correction
	// is a path, and the commonest path is this one.
	Where string
	// Gate is the command this sentence arrived with, and empty for the
	// sources that bring only words. It is what the form's check row opens
	// with: a rule read off what the checkout already refuses work over
	// arrives with a command that has been running for years, and retyping
	// it is copying it out of a file Orbit already read.
	Gate string
}
