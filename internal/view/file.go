package view

// File is one file a task left behind: what it is called and how big it is.
//
// That is the whole of what a listing can honestly say about a file without
// opening it, and it is deliberately not more. A pane that invents a size or
// a name is a pane a reader cannot use to check anything, which is the one
// job a list of artifacts has.
type File struct {
	Name string
	Size int64
}

// FileText is what a file holds, as much of it as a pane will read.
//
// Whole is whether Text is all of it. A pane that showed the first part of a
// file without saying so would be a pane a reader draws conclusions from
// about the part that is not there.
type FileText struct {
	Text  string
	Whole bool
}
