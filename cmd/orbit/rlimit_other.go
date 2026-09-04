//go:build !darwin

package main

// raiseFileLimit is a no-op on non-darwin platforms where the default file
// descriptor limit is typically sufficient.
func raiseFileLimit() {}
