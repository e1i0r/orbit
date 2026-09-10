import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import { App } from "./App";

// The worker is what makes the board installable — a browser offers "add to
// home screen" only to a page that has one. It caches nothing; see
// public/sw.js for why.
if ("serviceWorker" in navigator) {
  navigator.serviceWorker.register("/sw.js").catch(() => {
    // A board that could not register one is a board that still works and
    // cannot be installed, which is not worth a message on screen.
  });
}

const root = document.getElementById("root");
if (!root) throw new Error("the page has no root to draw into");

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
