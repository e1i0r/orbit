import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// The app is served by orbit itself: `orbit web` embeds what is built here
// and answers /api from the same origin. So there is no base path to set and
// no host to configure — in development the proxy below stands in for the
// binary, and in production there is nothing between them.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  build: {
    outDir: "dist",
    emptyOutDir: true,
    // Embedded in a Go binary, so every asset has to be reachable from the
    // one directory go:embed is pointed at.
    assetsDir: "assets",
  },
  server: {
    proxy: {
      "/api": "http://127.0.0.1:7777",
    },
  },
});
