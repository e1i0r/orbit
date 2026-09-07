package ui

// The sugar every translated sentence is written with.
//
// A words.Arg is a name and a value, and a call that spells that struct out
// is three times as long as the sentence it belongs to. It lives here rather
// than beside the key map it was written for, because the key map is a
// package now and this is the window's own shorthand.

import "github.com/e1i0r/orbit/internal/words"

// about names a value and what the sentence calls it.
func about(name, value string) words.Arg {
	return words.Arg{Name: name, Value: value}
}
