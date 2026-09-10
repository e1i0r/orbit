// The service worker exists so the board can be installed, and does nothing
// else.
//
// A browser will only offer "add to home screen" for a page that has one, so
// there has to be a file here. What there must not be is a cache: the app is
// served by `orbit web` from the binary it was built into, so a worker
// holding yesterday's bundle would answer with a board that cannot talk to
// the server behind it — and the reader would have no way to tell.
//
// Every request goes to the network, which is a machine on the same desk or
// the same house. Offline is not a state this board has: with the server
// gone there is nothing to show.

self.addEventListener("install", () => self.skipWaiting());
self.addEventListener("activate", (e) => e.waitUntil(self.clients.claim()));

// A fetch handler, because a browser will not offer to install a page whose
// worker has none. It answers nothing: not calling respondWith is how a
// worker says "let the network have it", which is the whole policy above.
self.addEventListener("fetch", () => {});
