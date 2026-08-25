# Orbit Code Style & Formatting Standards

These guidelines define the formatting, spacing, and structural conventions of Orbit.

---

## 📏 1. Line Length & Spacing Standards

- **Maximum Line Width:** **$\le 100$ characters** per line for code, signatures, and doc comments.
- **Vertical Spacing:**
  - Leave **1 blank line** between distinct logical steps inside a function (e.g., input validation $\rightarrow$ initialization $\rightarrow$ execution $\rightarrow$ error handling $\rightarrow$ return).
  - Leave **1-2 blank lines** between top-level declarations, types, and functions.
  - Avoid dense clusters of multi-statement blocks without breathing space.
- **Comments & Documentation:**
  - Wrap comment paragraphs at 80-90 characters.
  - Focus comments on design rationale, edge cases, and why a certain trade-off was made.

---

## 🗂️ 2. File Organization & Sizing

- **Maximum Lines per File:** **$< 300$ lines** (strictly enforced). Target $\le 295$ lines.
- **Naming Conventions:**
  - Lowercase with underscores or compact compound names (e.g., `pane_overview.go`, `settings_dials.go`, `flow_template.go`).
  - Unit tests placed beside their source files (`*_test.go`).
  - Fuzz tests named `*_fuzz_test.go` or using `Fuzz*` functions.

---

## 🛡️ 3. Error Handling & Wrapping

- **No Silent Discards:**
  - Bad: `home, _ := os.UserHomeDir()`
  - Good:
    ```go
    home, err := os.UserHomeDir()
    if err != nil || home == "" {
        home = os.Getenv("HOME")
    }
    ```
- **Error Wrapping:**
  - Always format error returns with `%w`: `return fmt.Errorf("read task %s: %w", id, err)`.

---

## 🪵 4. Diagnostic Logging

- Use `internal/logger` for all internal diagnostic and operational logging.
- Include the subsystem/module tag as the first argument:
  ```go
  logger.Info("cli/run", "starting task %s with flow %s", id, flowName)
  logger.Error("engine/claude", "stream parse error: %v", err)
  ```
