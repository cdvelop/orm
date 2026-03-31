# Plan: ORM — FieldDB Generation + Zero-Arg Input Constructors

## Depends on

- tinywasm/fmt PLAN.md (must be published first)
- tinywasm/form PLAN.md (should be published first so tests can use new input signatures)

## Problem

1. `ormc` generates flat `PK: true, Unique: true, AutoInc: true` in Field literals — must change to `DB: &fmt.FieldDB{...}`.
2. `ormc` already generates `input.Text()` without args — verify this is correct.
3. `formonly` structs should generate `DB: nil` (no FieldDB) since they have no database concerns.

## Changes

### 1. Update ormc_generate.go — Schema field generation

Replace flat field generation:

```go
// Before (ormc_generate.go ~lines 72-82)
if f.PK {
    buf.Write(", PK: true")
}
if f.Unique {
    buf.Write(", Unique: true")
}
if f.AutoInc {
    buf.Write(", AutoInc: true")
}
```

With grouped FieldDB generation:

```go
// After
if f.PK || f.Unique || f.AutoInc {
    buf.Write(", DB: &fmt.FieldDB{")
    parts := []string{}
    if f.PK {
        parts = append(parts, "PK: true")
    }
    if f.Unique {
        parts = append(parts, "Unique: true")
    }
    if f.AutoInc {
        parts = append(parts, "AutoInc: true")
    }
    buf.Write(strings.Join(parts, ", "))
    buf.Write("}")
}
```

**Condition**: Only emit `DB: &fmt.FieldDB{...}` when struct is NOT `formonly`. For `formonly` structs, skip DB fields entirely (they get `DB: nil`).

### 2. Update ormc.go — FieldInfo struct

The `FieldInfo` struct in ormc.go still needs PK, Unique, AutoInc as parsing fields (read from `db:` tags). No change needed here — these are parser-level, not output-level.

### 3. Update db.go

```go
// Before
if f.PK && f.AutoInc {

// After
if f.IsPK() && f.IsAutoInc() {
```

### 4. Verify input constructor generation

Confirm that ormc templates generate `input.Text()` without arguments. Based on current ormc_test.go assertions (`Widget: input.Text()`), this is already correct.

### 5. Update tests

**ormc_test.go** — Update expected string assertions:

```go
// Before
"{Name: \"id\", Type: fmt.FieldInt, PK: true}"

// After
"{Name: \"id\", Type: fmt.FieldInt, DB: &fmt.FieldDB{PK: true}}"
```

**tests/models.go** — Update any manual schema literals.

### 6. Regenerate all _orm.go files

After ormc is updated, regenerate:
- tinywasm/user/models_orm.go
- tinywasm/mcp/model_orm.go
- tinywasm/skills/models_orm.go

These are generated files — run `ormc` on each module.

## Consumer modules

sqlite, postgres, indexdb have their own PLAN.md — see master plan at docs/PLAN_ORM_FORM.md.

## Execution Order

1. Wait for tinywasm/fmt with FieldDB published
2. Bump fmt dependency
3. Update ormc_generate.go (FieldDB generation, formonly condition)
4. Update db.go (use helpers)
5. Update ormc_test.go assertions
6. Run `go test ./...` in orm
7. Publish orm
8. Regenerate _orm.go in user, mcp, skills

## Verification

- `go test ./...` passes in orm
- Generated code for formonly structs has NO `DB:` field
- Generated code for db structs has `DB: &fmt.FieldDB{PK: true, ...}`
- Generated code for form structs has both `DB:` and `Widget:`
- All consumer modules compile and pass tests
