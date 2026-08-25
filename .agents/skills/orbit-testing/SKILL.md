---
name: orbit-testing
description: >-
  Standard operating procedures for Orbit testing strategy: >=90% test coverage,
  multi-phase E2E flows, native Go fuzz testing, property invariants, and golden files.
---

# Orbit Testing & Verification Guide 🧪

This skill guides the implementation and maintenance of Orbit's comprehensive test suite.

---

## 🎯 Testing Standards & Target Coverage

- **Target Coverage:** **$\ge 90\%$ statement coverage** across all packages.
- **Zero Flakes:** All tests must pass deterministically without timing races.

---

## 🔬 Test Categories & Patterns

### 1. Multi-Phase End-to-End Task Flows (`internal/task`)
- Validate task lifecycle across multi-stage flows (`plan` $\rightarrow$ `implement` $\rightarrow$ `test` $\rightarrow$ `review`).
- Verify pause/resume control signals, operator notes injection mid-flight, and gate checks.

### 2. Native Go Fuzz Testing (`testing.F`)
Use Go fuzzing for all input parsers, streaming event decoders, and formatting engines:
```go
func FuzzStreamParser(f *testing.F) {
    f.Add([]byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}]}}`))
    f.Fuzz(func(t *testing.T, data []byte) {
        // Must never panic on arbitrary malformed bytes
        _, _ = ParseStreamEvents(data)
    })
}
```

### 3. Property-Based Invariant Tests
- **Cost Monotonicity:** Task accumulated financial cost and token counts must never decrease.
- **Persistence Round-Trip:** Serializing and deserializing models must be lossless.
- **Grid Bounding:** TUI layout cells must never produce negative coordinates or exceed screen bounds.

### 4. Golden File Updates (`internal/ui`)
When UI designs or text are intentionally updated, regenerate golden files:
```bash
go test ./internal/ui -update
```

---

## 🚀 Running Test Commands

```bash
# 1. Run all tests with race detection
go test -race ./...

# 2. Check statement coverage per package
go test -cover ./...

# 3. Run full verification suite (vet, fmt, lint, test, tidy)
make check
```
