package web

// The page itself.
//
// One file, embedded, no build step. That is deliberate for now: what the
// browser is here to do first is show a diff a terminal shows badly, and
// that needs a page, not a toolchain. The component layer — frauddi's
// patterns, and the diagrams — arrives when it is decided how it is built,
// and this file is what it replaces.
//
// The colours are Orbit's, named the way internal/ui/theme names them, in
// the two-layer model frauddi's own tokens.css keeps: the hexes are
// primitives at the top and nothing below them uses one directly.

import (
	_ "embed"
	"net/http"
)

//go:embed page.html
var page []byte

// servePage answers with the one page there is.
func (s *Server) servePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		fail(w, http.StatusNotFound, "no page at "+r.URL.Path, nil)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if _, err := w.Write(page); err != nil {
		_ = err //nolint:wsl // the reader closed the tab; there is nowhere to say so
	}
}
