# PLAN: Migrate to `fmt.Field` and Generate Form Code in `ormc`

## Development Rules

- **Standard Library Only:** No external assertion libraries. Use `testing`, `net/http/httptest`, `reflect` (tests only).
- **Testing Runner:** Install and use `gotest`:
  ```bash
  go install github.com/tinywasm/devflow/cmd/gotest@latest
  ```
- **Max 500 lines per file.** If exceeded, subdivide by domain.
- **Flat hierarchy.** No subdirectories for library code.
- **Mandatory DI:** No direct system calls in logic. Interfaces for external deps.
- **Documentation First:** Update docs before coding.
- **Publishing:** Use `gopush 'message'` after tests pass and docs are updated.

## Prerequisites

- **`tinywasm/fmt`** must be published with `Field`, `FieldType`, and `Fielder` interface available. Update `go.mod` to require the new `fmt` version before starting.

## Context

The `tinywasm/orm` package provides a zero-reflection ORM with compile-time code generation (`ormc`). Currently:

- `orm.Field` and `orm.FieldType` are defined locally in `field_type.go`.
- `orm.Model` interface is: `TableName() + Schema() []Field + Values() []any + Pointers() []any`.
- `ormc` generates `model_orm.go` with ORM bindings only.

This plan:

1. **Migrates** `Field`/`FieldType`/`Constraint` to use `fmt.Field`/`fmt.FieldType` (with bools instead of bitmask).
2. **Redefines** `Model` as `fmt.Fielder` + `TableName()`.
3. **Extends** `ormc` to parse `form:` and `json:` tags, populating `fmt.Field.Input` and `fmt.Field.JSON` in generated `Schema()`.
4. **Adds** `orm.FieldExt` for database-only metadata (FK refs).
5. **Adds** `FormName()` generation for form ID/action URL resolution.
6. **Detects** nested struct fields and generates `Type: fmt.FieldStruct` for them.

**Key principle:** `ormc` does NOT import or generate code that references `tinywasm/form` or `tinywasm/json`. It only populates `fmt.Field.Input` and `fmt.Field.JSON` (plain strings) in the generated `Schema()`. Each downstream package reads these fields autonomously. **orm, form, and json never know about each other.**

### What This Plan Does NOT Cover

- Changes to `tinywasm/form` — that has its own independent plan.
- Changes to `tinywasm/crudp` — noted as downstream impact only.

---

## Stage 1: Migrate `Field` and `FieldType` to `fmt`

← None | Next → [Stage 2](#stage-2-redefine-model-interface)

### 1.1 Update `field_type.go`

**Remove:** `FieldType`, `Field`, `Constraint`, and all associated constants.

**Replace with:**

```go
package orm

import "github.com/tinywasm/fmt"

// FieldExt extends fmt.Field with database-specific metadata (foreign keys).
// Used internally by adapters that support FK constraints.
type FieldExt struct {
    fmt.Field
    Ref       string // FK: target table name. Empty = no FK.
    RefColumn string // FK: target column. Empty = auto-detect PK of Ref table.
}
```

**Rename** `field_type.go` → `field_ext.go` to reflect the new domain (only DB extensions now).

### 1.2 Update all internal references to use `fmt` directly

No aliases. All code uses `fmt.Field`, `fmt.FieldText`, `fmt.FieldInt`, etc. directly.

Search and replace across the package:
- `orm.Field` → `fmt.Field`
- `orm.FieldType` → `fmt.FieldType`
- `TypeText` → `fmt.FieldText`, `TypeInt` → `fmt.FieldInt`, `TypeFloat` → `fmt.FieldFloat`, `TypeBool` → `fmt.FieldBool`, `TypeBlob` → `fmt.FieldBlob`
- `Constraint` bitmask checks → bool field access:
  - `f.Constraints&ConstraintPK != 0` → `f.PK`
  - `f.Constraints&ConstraintNotNull != 0` → `f.NotNull`
  - `f.Constraints&ConstraintUnique != 0` → `f.Unique`
  - `f.Constraints&ConstraintAutoIncrement != 0` → `f.AutoInc`
- `Constraints: ConstraintPK | ConstraintNotNull` → `PK: true, NotNull: true`

Files that reference `Field` or `Constraint` (search `Constraint` and `FieldType` across all `.go` files):
- `ormc.go` (struct parsing, code generation)
- `ormc_relations.go` (FK detection)
- Adapter-facing code (if any uses `Field` directly)
- All test files

### 1.4 Update `FieldExt` usage for FK

In `ormc.go` and `ormc_relations.go`, FK metadata (`Ref`, `RefColumn`) now lives in `FieldExt`. The generated `Schema()` returns `[]fmt.Field` (without FK). FK information is used only during `CREATE TABLE` generation — store it separately in the `StructInfo`:

```go
type StructInfo struct {
    Name        string
    Fields      []fmt.Field    // Schema fields (no FK)
    FKRefs      []FieldExt     // Fields with FK metadata (for CREATE TABLE only)
    SliceFields []SliceField   // One-to-many relations
    // ...existing fields...
}
```

### 1.5 Tests

```bash
gotest
```

All existing tests must pass with zero behavior changes. The migration is purely structural.

---

## Stage 2: Redefine `Model` Interface

← [Stage 1](#stage-1-migrate-field-and-fieldtype-to-fmt) | Next → [Stage 3](#stage-3-extend-ormc-to-generate-form-code)

### 2.1 Update `model.go`

```go
package orm

import "github.com/tinywasm/fmt"

// Model represents a database model.
// It extends fmt.Fielder with a table name for database operations.
// Implementations are generated by ormc.
type Model interface {
    fmt.Fielder
    TableName() string
}
```

### 2.2 Verify compilation

All existing code that calls `m.Schema()`, `m.Values()`, `m.Pointers()` on a `Model` should compile without changes because `Model` now embeds `fmt.Fielder` which provides those methods.

### 2.3 Tests

```bash
gotest
```

---

## Stage 3: Parse `form:` Tag and Populate `Field.Input` Hint

← [Stage 2](#stage-2-redefine-model-interface) | Next → [Stage 4](#stage-4-support-form-only-structs)

**Key principle:** `ormc` does NOT import `tinywasm/form`. It only reads the `form:` struct tag and writes the value into `fmt.Field.Input` in the generated `Schema()`. The `form` package reads `Field.Input` autonomously to decide which input type to use.

### 3.1 Parse `form:` and `json:` tags in `ormc.go`

In the `ParseStruct` function, for each field, also read the `form` and `json` struct tags:

**`form:` tag → `Field.Input`:**
- `form:"-"` → `Field.Input = "-"` (form will skip this field).
- `form:"email"` → `Field.Input = "email"` (form will use email input instead of name heuristic).
- No `form` tag → `Field.Input = ""` (form uses its name-based heuristic, as it does today).

**`json:` tag → `Field.JSON`:**
- `json:"-"` → `Field.JSON = "-"` (json codec will skip this field).
- `json:"email"` → `Field.JSON = "email"` (json codec uses "email" as key).
- `json:"email,omitempty"` → `Field.JSON = "email,omitempty"` (copied verbatim).
- No `json` tag → `Field.JSON = ""` (json codec uses `Field.Name` as key).

**Nested struct fields → `FieldStruct`:**
- If a field's Go type is a struct (not `time.Time`, not a slice), set `Type: fmt.FieldStruct`.
- The generated `Values()` returns the struct value (which must also implement `Fielder`).
- The generated `Pointers()` returns a pointer to the struct field.

### 3.2 Generate `Field.Input` in `Schema()`

The generated `Schema()` now includes the `Input` hint. Example for:

```go
type User struct {
    ID       string
    Name     string                          `json:"name"`
    Email    string `form:"email"             json:"email"`
    Password string `form:"password"          json:"password"`
    Bio      string `form:"textarea"          json:"bio,omitempty"`
    Age      int64  `form:"-"                 json:"age"`
}
```

Generated `Schema()`:

```go
func (m *User) Schema() []fmt.Field {
    return []fmt.Field{
        {Name: "ID", Type: fmt.FieldText, PK: true},
        {Name: "Name", Type: fmt.FieldText, NotNull: true},
        {Name: "Email", Type: fmt.FieldText, NotNull: true, Input: "email", JSON: "email"},
        {Name: "Password", Type: fmt.FieldText, Input: "password", JSON: "password"},
        {Name: "Bio", Type: fmt.FieldText, Input: "textarea", JSON: "bio,omitempty"},
        {Name: "Age", Type: fmt.FieldInt, Input: "-", JSON: "age"},
    }
}
```

**No `form` import is added to the generated file.** `fmt.Field.Input` is a plain string — zero coupling.

### 3.3 Generate `FormName()` method

For every struct (both Model and formonly), generate:

```go
func (m *User) FormName() string { return "user" }
```

This returns the lowercase struct name. The `form` package uses it (via optional interface assertion) for the form ID and action URL.

### 3.4 Tests

- `TestOrmcFormTag`: Parse a struct with `form:` tags, verify `Field.Input` values in generated `Schema()`.
- `TestOrmcFormExclusion`: Field with `form:"-"` has `Input: "-"` in Schema.
- `TestOrmcFormNoTag`: Field without `form:` tag has `Input: ""`.
- `TestOrmcFormName`: Verify `FormName()` returns lowercase struct name.
- `TestOrmcJSONTag`: Parse a struct with `json:` tags, verify `Field.JSON` values in generated `Schema()`.
- `TestOrmcJSONOmitEmpty`: Field with `json:"name,omitempty"` has `JSON: "name,omitempty"`.
- `TestOrmcJSONExclusion`: Field with `json:"-"` has `JSON: "-"` in Schema.
- `TestOrmcJSONNoTag`: Field without `json:` tag has `JSON: ""`.
- `TestOrmcFieldStruct`: Nested struct field generates `Type: fmt.FieldStruct`.

```bash
gotest
```

---

## Stage 4: Support Form-Only Structs

← [Stage 3](#stage-3-extend-ormc-to-generate-form-code) | Next → [Stage 5](#stage-5-documentation-and-publish)

### 4.1 Implement `// ormc:formonly` directive

In `ormc.go`, when scanning struct declarations, check for a comment directive immediately above the struct:

```go
// ormc:formonly
type LoginRequest struct {
    Email    string
    Password string `form:"password"`
}
```

When `// ormc:formonly` is present:
- **Generate:** `Schema() []fmt.Field`, `Values() []any`, `Pointers() []any` (implements `fmt.Fielder`).
- **Generate:** `FormName()` returning the lowercase struct name.
- **Do NOT generate:** `TableName()`, `ReadOne*`, `ReadAll*`, relation loaders.
- The struct satisfies `fmt.Fielder` but NOT `orm.Model` (no `TableName()`).

### 4.2 Tests

- `TestOrmcFormOnly`: Parse a struct with `// ormc:formonly`, verify only `Fielder` methods + `FormName()` are generated (no `TableName`, no read helpers).
- `TestOrmcFormOnlyNoFK`: Verify `db:"ref=..."` tags are ignored on formonly structs.

```bash
gotest
```

---

## Stage 5: Documentation and Publish

← [Stage 4](#stage-4-support-form-only-structs) | None →

### 5.1 Update `docs/SKILL.md`

Add sections:
- `fmt.Field` migration: new bool constraints, no bitmask.
- `orm.Model` now embeds `fmt.Fielder`.
- `FieldExt` for FK metadata.
- `ormc` form generation: `form:` tag syntax, auto-mapping rules, `// ormc:formonly`.

### 5.2 Update `docs/ARQUITECTURE.md`

Reflect the new dependency: `orm.Model` → `fmt.Fielder`.

### 5.3 Update `README.md`

- Update the Model interface example.
- Add section on form generation with `form:` tags.
- Document `// ormc:formonly` directive.

### 5.4 Run full test suite

```bash
gotest
```
---

## Summary of Changes

| File | Action |
|------|--------|
| `field_type.go` → `field_ext.go` | Remove `Field`/`FieldType`/`Constraint`, add `FieldExt` with FK fields |
| `model.go` | `Model` embeds `fmt.Fielder` + `TableName()` |
| `ormc.go` | Parse `form:`/`json:` tags into `Field.Input`/`Field.JSON`, detect `FieldStruct`, generate `FormName()`, handle `// ormc:formonly` |
| `ormc_relations.go` | Use `FieldExt` for FK resolution |
| All tests | Update `Field{}` literals to use bool flags instead of bitmask |
| `docs/SKILL.md` | Updated |
| `docs/ARQUITECTURE.md` | Updated |
| `README.md` | Updated |

## Downstream Impact

- **`tinywasm/form`**: Must update to accept `fmt.Fielder` (has its own plan).
- **`tinywasm/crudp`**: `DataValidator` interface should change `any` → `fmt.Fielder` in `ValidateData`. This is a one-line interface change — no separate plan needed.
- **All adapters** (sqlite-wasm, indexeddb): Must update `Field` references to `fmt.Field`. If they use `Constraint` bitmask, migrate to bool fields.
