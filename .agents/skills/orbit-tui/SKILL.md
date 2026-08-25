---
name: orbit-tui
description: >-
  Standard operating procedures for Orbit's Bubble Tea v2 Terminal UI: 11 inspector tabs,
  pure geometric layout math, mouse routing, command palette, and color palettes.
---

# Orbit TUI Development & Design Guide 🎨

This skill covers the architecture, components, and design patterns of the Orbit Terminal Cockpit.

---

## 🎛️ Architecture of the Cockpit

The Orbit Cockpit is built on **Bubble Tea v2** and **Lip Gloss**:

1. **Pure Geometry Layer (`internal/ui/layout`):**
   - Pure mathematical calculations for frame sizing, header bounds, task table columns, and tab bar bounding boxes.
   - Zero Lip Gloss or Bubble Tea dependencies.
2. **Interactive Target Routing (`internal/ui/target.go`):**
   - Every clickable screen region maps to a `Target{Kind, ID, Pane, Field}` struct.
   - Mouse click events are routed deterministically based on layout grid hit-testing.
3. **Tab Inspector Hierarchy (11 Tabs):**
   - `tabOverview` (1) · `tabFlow` (2) · `tabGates` (3) · `tabCost` (4) · `tabRefused` (5) · `tabTimeline` (6)
   - `tabReport` (7) · `tabArtifacts` (8) · `tabNotes` (9) · `tabDiff` (0) · `tabThinking` (w)
4. **Theme Palettes (`internal/ui/theme.go`):**
   - Curated palettes: `frauddi` (default), `monokai`, `nord`, `tokyo-night`, `dracula`, `catppuccin`.
   - Adaptive color rendering for light/dark terminal backgrounds.

---

## 📐 Layout Rules & Mouse Routing

- Always define layout dimensions and cell boundaries in `internal/ui/layout`.
- Register interactive targets in `cells.go` using `t.grid[y][x] = Target{...}`.
- Handle left and right clicks in `mouseroute.go` / `flowsmouse.go`.
- Ensure all screens support both full keyboard navigation and mouse pointer interactions.
