---
name: orbit-workflow
description: >-
  Standard operating procedures for Orbit feature development, task lifecycle pipelines,
  git worktree isolation, architecture layers, and release workflows.
---

# Orbit Workflow & Development Guide 🛰️

This skill outlines the standard workflows for designing, developing, and deploying features within Orbit.

---

## 🧭 Architecture Layers & Responsibilities

Orbit enforces a strict unidirectional dependency graph defined in `internal/arch/imports_test.go`:

| Layer / Package | Allowed Imports | Primary Responsibility |
| :--- | :--- | :--- |
| `cmd/orbit` | `internal/cli` | Application entrypoint and process exit handling. |
| `internal/cli` | `board`, `engine`, `flow`, `logger`, `quota`, `repo`, `store`, `task`, `ui`, `view`, `words` | CLI command parsing, flag binding, and terminal dispatch. |
| `internal/board` | `record`, `repo`, `store`, `task`, `view` | Real-time task board grouping, bands, and polling sweeps. |
| `internal/task` | `engine`, `flow`, `record`, `repo`, `store` | Task execution lifecycle, gates, operator notes, and child process management. |
| `internal/record` | *(none)* | Append-only JSONLines event logs (`events.jsonl`). |
| `internal/store` | *(none)* | Pure on-disk filesystem path calculation and state roots. |
| `internal/engine` | *(none)* | Multi-engine adapters (Claude Code, OpenAI Codex, OpenCode). |
| `internal/ui` | `board`, `flow`, `repo`, `task`, `ui/layout`, `view`, `words` | Bubble Tea TUI cockpit rendering, input routing, and modals. |
| `internal/ui/layout`| `view` | Pure geometric bounding boxes, cell grids, and dimension fitting. |
| `internal/logger` | *(none)* | Thread-safe internal file diagnostic logging. |

---

## 🛠️ Step-by-Step Feature Workflow

1. **Check Architectural Boundaries:**
   - Verify that any new package dependencies are allowed by `internal/arch/imports_test.go`.
2. **Design with Pure Layouts:**
   - Put all geometry and bounding math into `internal/ui/layout` without side effects.
3. **Keep File Size $< 300$ Lines:**
   - Keep files modular ($\le 295$ lines).
4. **Instrument Transparent Logging:**
   - Log critical state transitions and errors using `logger.Info("module", ...)` / `logger.Error("module", ...)`.
5. **Enforce Internationalization:**
   - Use `p.T("key", "default English")` for all user-facing strings and update `internal/words/lang/es.json`.
6. **Run Full Verification:**
   ```bash
   make check
   ```
