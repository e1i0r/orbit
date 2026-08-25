# Contributing to Orbit 🚀

Thank you for your interest in contributing to Orbit! Orbit is a cockpit designed for supervising and orchestrating coding agents (Claude Code, Codex, OpenCode, etc.) across parallel multi-phase task workflows.

---

## 🏛️ Codebase Philosophy & Architecture Rules

Before submitting code, please review our core architectural tenets:

1. **Strict File Size Limit ($< 300$ lines):**
   - Every Go file (`*.go`) must strictly stay under **300 lines** (target $\le 295$ lines).
   - If a file approaches or grows beyond 300 lines, split cohesive sub-responsibilities into adjacent focused files (e.g. `settingsdials.go`, `flowsmouse.go`, `mouseroute.go`).
2. **File Naming & Component Structure:**
   - Use clear, lower-case, cohesive file names (`snake_case` or compact compound nouns, e.g. `pane_overview.go`, `settings_dials.go`, `flow_template.go`).
   - Group package code logically: data structures and interfaces first, followed by implementation, message handlers, and view renderers.
   - Keep domain packages decoupled: `internal/board` owns task queue state, `internal/task` owns execution, `internal/record` owns append-only logs, and `internal/ui` owns TUI rendering.
3. **$\ge 90\%$ Test Coverage Standard:**
   - All contributions must maintain or improve repository test coverage, targeting **$\ge 90\%$ total coverage**.
   - Include unit tests, full multi-phase task lifecycle E2E tests, and property/fuzz tests where input parsers or event streams are modified.
4. **Explicit Error Handling:**
   - **Never ignore or silence errors silently.** Always check `if err != nil` and propagate errors with `%w` wrapping.
   - Avoid `_ = f()` discards. If a discard is strictly intentional (e.g. best-effort closing of an already-errored reader), document it explicitly with a clear `//nolint:errcheck // reason` comment.
5. **No Paraphrasing / Evidence Integrity:**
   - LLM and tool outputs are recorded verbatim in append-only JSONLines event logs (`events.jsonl`).
6. **Pure Layout Calculations:**
   - `internal/ui/layout` performs pure geometric calculations without side effects or styling.
7. **Internationalization & Translation Honesty:**
   - All user-facing UI text goes through `words.Printer` (`p.T(...)`).
   - Every key declared in `es.json` / `en.json` must be used in Go code (`TestEveryTranslationKeyIsHonest`).

---

## 🛠️ Development Setup

### Prerequisites
- **Go 1.26+** installed.
- **Git** installed.
- **golangci-lint** (`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`).

### Clone & Build
```bash
git clone https://github.com/e1i0r/orbit.git
cd orbit

# Build binary
make build

# Run Orbit Cockpit
./orbit top .
```

---

## 🧪 Testing & Verification

Every contribution must pass the full verification suite before being merged:

```bash
# 1. Run all checks (format, lint, vet, test, go.mod tidy)
make check

# 2. Run test suite with coverage
go test -cover ./...

# 3. Update UI golden files (if UI layouts or text changed intentionally)
go test ./internal/ui -update
```

### Testing Types in Orbit
- **Unit Tests:** Individual package functions and state machines.
- **End-to-End Task Lifecycle (`internal/task`):** Multi-phase runs, operator notes, human gates, shell gate checks, and process reconciliation.
- **Fuzz Testing (`testing.F`):** Native fuzzing for stream parsers, event scanners, quota decoders, and terminal layout fitters.
- **Property-Based Invariant Tests:** Monotonic costs, persistence round-trips, and geometric tiling.

---

## 📬 Pull Request Process

1. **Fork the repository** and create a descriptive branch name (e.g., `feature/my-cool-feature` or `fix/task-stream-parsing`).
2. **Commit your changes** with clear, concise commit messages.
3. Ensure all tests and linters pass (`make check`).
4. **Open a Pull Request** against the `main` branch with a summary of changes, motivation, and test coverage evidence.
5. A maintainer will review your PR and provide constructive feedback!

---

## ⚖️ Code of Conduct

Please note that this project is released with a [Contributor Code of Conduct](CODE_OF_CONDUCT.md). By participating in this project you agree to abide by its terms.
