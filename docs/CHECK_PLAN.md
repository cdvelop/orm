# PLAN: Fielder v2 Migration (tinywasm/orm)

← [README](../README.md) | Depends on: [fmt PLAN_FIELDER_V2](../../fmt/docs/PLAN_FIELDER_V2.md)

## Development Rules

- **Standard Library Only:** No external assertion libraries. Use `testing`.
- **Testing Runner:** Use `gotest` (install: `go install github.com/tinywasm/devflow/cmd/gotest@latest`).
- **Max 500 lines per file.** If exceeded, subdivide by domain.
- **Flat hierarchy.** No subdirectories for library code.
- **Documentation First:** Update docs before coding.
- **Publishing:** Use `gopush 'message'` after tests pass and docs are updated.

## Prerequisite

Update `go.mod` to the new `tinywasm/fmt` version (Fielder v2, without `Values()`):

```bash
go get github.com/tinywasm/fmt@latest
```

## Context

The `orm` library uses `Values()` in two places:
- `db.go:34` — `Create()`: `q.Values = m.Values()`
- `db.go:60` — `Update()`: `q.Values = m.Values()`

Both pass `[]any` to the Query struct, which the Compiler uses to build SQL args.

Since DB operations are I/O bound (disk/network latency dominates), the allocation cost of
`fmt.ReadValues()` is negligible. This migration is straightforward.

Additionally, `ormc.go` generates `Schema()`, `Values()`, and `Pointers()` for every model struct.
The code generator must be updated to:
1. Stop generating `Values()`
2. Generate `Schema()` as a package-level variable (0 allocs)

---

## Stage 1: Update `db.go` — Replace `m.Values()` with `fmt.ReadValues()`

**File:** `db.go`

### 1.1 Update `Create()`

```go
// BEFORE (line 34):
Values: m.Values(),

// AFTER:
Values: fmt.ReadValues(schema, m.Pointers()),
```

Note: `schema` is already available at line 25 (`schema := m.Schema()`), so no extra call needed.
`m.Pointers()` is called once for the Values extraction.

### 1.2 Update `Update()`

```go
// BEFORE (line 60):
Values: m.Values(),

// AFTER:
Values: fmt.ReadValues(schema, m.Pointers()),
```

Same pattern: `schema` is already available at line 51.

### 1.3 Update `emptyModel`

**File:** `db.go` (lines 71-76)

Remove the `Values()` method from `emptyModel`:

```go
// BEFORE:
type emptyModel struct{}
func (e emptyModel) TableName() string   { return "" }
func (e emptyModel) Schema() []fmt.Field { return nil }
func (e emptyModel) Values() []any       { return nil }
func (e emptyModel) Pointers() []any     { return nil }

// AFTER:
type emptyModel struct{}
func (e emptyModel) TableName() string   { return "" }
func (e emptyModel) Schema() []fmt.Field { return nil }
func (e emptyModel) Pointers() []any     { return nil }
```

---

## Stage 2: Update `ormc.go` — Code Generator

**File:** `ormc.go`

### 2.1 Generate Schema as package-level variable

Replace the Schema generation block (lines ~361-400):

```go
// BEFORE:
buf.Write(Sprintf("func (m *%s) Schema() []fmt.Field {\n", info.Name))
buf.Write("\treturn []fmt.Field{\n")
// ... fields ...
buf.Write("\t}\n")
buf.Write("}\n\n")

// AFTER:
// Package-level variable — 0 allocs on every Schema() call
buf.Write(Sprintf("var _schema%s = []fmt.Field{\n", info.Name))
for _, f := range info.Fields {
    // ... same field generation logic ...
}
buf.Write("}\n\n")
buf.Write(Sprintf("func (m *%s) Schema() []fmt.Field { return _schema%s }\n\n", info.Name, info.Name))
```

### 2.2 Remove Values generation

Delete the Values generation block (lines ~402-408):

```go
// DELETE THIS ENTIRE BLOCK:
buf.Write(Sprintf("func (m *%s) Values() []any {\n", info.Name))
buf.Write("\treturn []any{\n")
for _, f := range info.Fields {
    buf.Write(Sprintf("\t\tm.%s,\n", f.Name))
}
buf.Write("\t}\n")
buf.Write("}\n\n")
```

### 2.3 Keep Pointers generation unchanged

The `Pointers()` generation (lines ~410-416) stays exactly as-is.
Pointers are stable for a struct instance and elements (pointers) fit in interface word (0 boxing allocs).

---

## Stage 3: Update Model interface

**File:** `model.go`

Update the comment that references `Values()`:

```go
// BEFORE:
type Model interface {
    fmt.Fielder           // Schema() []fmt.Field + Values() []any + Pointers() []any
    TableName() string
}

// AFTER:
type Model interface {
    fmt.Fielder           // Schema() []fmt.Field + Pointers() []any
    TableName() string
}
```

---

## Stage 4: Update tests

### 4.1 Remove `Values()` from test mocks

Search all test files for `Values()` implementations:

```bash
grep -rn "func.*Values().*\[\]any" tests/ *_test.go
```

Remove every `Values()` method from mock structs.

### 4.2 Run tests

```bash
gotest
```

---

## Stage 5: Update documentation

### 5.1 Update `docs/SKILL.md`

- Remove `Values() []any` from the Fielder interface listing (line 27, 37)
- Update the "For every struct, `ormc` generates" section (line 106): remove `Values()` bullet
- Update any code examples that reference `Values()`

### 5.2 Update `docs/ARQUITECTURE.md` if it references `Values()`

```bash
grep -n "Values" docs/ARQUITECTURE.md
```

---

## Stage 6: Regenerate all downstream model_orm.go files

After publishing the new `ormc`, ALL projects using `ormc` must regenerate their models:

```bash
ormc
```

This is a **breaking change** — all `model_orm.go` files will lose their `Values()` method.
Downstream projects must update their `tinywasm/orm` dependency first.

---

## Stage 7: Publish

```bash
gopush 'orm: Fielder v2 migration — remove Values(), Schema as package var in ormc'
```

---

## Summary

| Stage | File(s) | Action |
|-------|---------|--------|
| 1 | `db.go` | Replace `m.Values()` with `fmt.ReadValues()` |
| 2 | `ormc.go` | Schema as pkg var, stop generating `Values()` |
| 3 | `model.go` | Update interface comment |
| 4 | `*_test.go` | Remove Values() from mocks |
| 5 | `docs/` | Update SKILL.md and ARQUITECTURE.md |
| 6 | — | Regenerate downstream models |
| 7 | — | `gotest` + `gopush` |
