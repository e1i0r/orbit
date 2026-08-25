# Orbit Agent Directives & Engineering Standards 🛰️

Welcome to Orbit. When contributing or modifying code in this repository, you must strictly follow these engineering principles:

---

## 🏛️ 1. Core Architectural Laws

1. **Strict File Size Ceiling ($< 300$ Lines):**
   - No `.go` source file may exceed **300 lines** (target $\le 295$ lines).
   - If a file approaches 300 lines, extract cohesive responsibilities into adjacent dedicated files (e.g. `settings_dials.go`, `flow_mouse.go`, `mouse_routing.go`).
2. **Line Length & Spacing Discipline ($\le 100$ Characters):**
   - Keep all lines of code, signatures, comments, and strings within **100 characters**.
   - Code must breathe: separate logical thought blocks with single empty lines, avoid dense code walls.
   - Place clear, explanatory comments above non-trivial decisions explaining *why*, not just *what*.
3. **Explicit Error Handling (Zero Silent Discards):**
   - Never ignore errors silently with `_ = f()` or unhandled assignments.
   - Always check `if err != nil` and wrap with `%w` for error chain integrity (`fmt.Errorf("do thing: %w", err)`).
4. **Structured Diagnostic Logging (`internal/logger`):**
   - Log all actionable failures, timeouts, and state transitions through `internal/logger`.
   - Always include the subsystem module tag: `logger.Error("cli/run", "failed: %v", err)`.
   - Never write debug logs directly to `os.Stdout` or `os.Stderr` in Cockpit code to avoid corrupting the TUI.
5. **Rigorous Test Coverage ($\ge 90\%$):**
   - Maintain $\ge 90\%$ statement coverage across packages.
   - Include unit tests, full task lifecycle E2E flows, native Go fuzzing (`testing.F`), and property invariants.
6. **Translation & Internationalization Honesty (`words.Printer`):**
   - All user-facing strings must be passed through `p.T("key", "default English")`.
   - Every translation key in `es.json` and `en.json` must be actively referenced in Go code (`TestEveryTranslationKeyIsHonest`).
7. **Pure Layout Calculations (`internal/ui/layout`):**
   - Layout geometry calculations must remain pure functions without styling, escape codes, or side effects.
