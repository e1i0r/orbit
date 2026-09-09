package web

// The page, and everything it loads.
//
// What is served is what ui/ built: a request that names a file gets that
// file, and every other path gets index.html, because the app decides what a
// path means and the server has no opinion about it. That is what makes the
// back button and a pasted link work without the two of them agreeing on a
// list of routes.

import (
	"io/fs"
	"net/http"
	"strings"
)

// servePage answers with a built file, or with the app itself.
func (s *Server) servePage(w http.ResponseWriter, r *http.Request) {
	if s.files == nil {
		fail(w, http.StatusInternalServerError, "the window was not built into this binary", nil)

		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/")
	if name != "" {
		if _, err := fs.Stat(s.files, name); err == nil {
			// Every built asset carries a hash of its own contents in its
			// name, so one that is asked for again is one that has not
			// changed. A year is the longest anything is allowed to say.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			http.FileServerFS(s.files).ServeHTTP(w, r)

			return
		}
	}

	// Not an asset, so it is a place in the app. index.html knows what to do
	// with it and the server does not.
	body, err := fs.ReadFile(s.files, "index.html")
	if err != nil {
		fail(w, http.StatusInternalServerError, "read the window", err)

		return
	}

	// And the page itself is never cached: it is what names the assets, so a
	// browser holding yesterday's copy loads yesterday's window from a
	// binary that has today's. That is not a theory — it happened, and the
	// window it drew was one build behind with no way to tell.
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if _, err := w.Write(body); err != nil {
		_ = err //nolint:wsl // the reader closed the tab; there is nowhere to say so
	}
}
