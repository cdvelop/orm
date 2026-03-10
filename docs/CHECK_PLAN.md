# PLAN: Consolidated ORM Fixes and Documentation

## Development Rules

- **Standard Library Only:** No external assertion libraries.
- **Testing Runner:** Use `gotest`.
- **Documentation First:** Update docs before final publish.
- **Publishing:** Use `gopush 'message'`.

## Context

The `tinywasm/orm` has partially implemented the previous plan. 
Specifically:
- [x] Migration to `fmt.Field` and `fmt.FieldType`.
- [x] Redefinition of `Model` interface.
- [x] Support for `// ormc:formonly`.
- [/] Parsing `form:` tags (implemented, but missing `json:` tag parsing and nested structs).

This consolidated plan focuses on the missing pieces to complete the transition.

---

## Stage 1: Complete `ormc` Tag Parsing and Field Detection

← None | Next → [Stage 2](#stage-2-documentation-updates)

**Key principle:** `ormc` does NOT import `tinywasm/form`. It only reads the `form:` and `json:` struct tags and writes them into `fmt.Field.Input` / `fmt.Field.JSON` in the generated `Schema()`. Each downstream package reads these fields autonomously.

### 1.1 Parse `form:` and `json:` tags in `ormc.go`

In `ParseStruct`, for each field, read the `form` and `json` struct tags:

**`form:` tag → `Field.Input`:**
- `form:"-"` → `Field.Input = "-"` (form will skip this field).
- `form:"email"` → `Field.Input = "email"` (form uses email input).
- No `form` tag → `Field.Input = ""` (form uses name-based heuristic).

**`json:` tag → `Field.JSON`:**
- `json:"-"` → `Field.JSON = "-"` (json codec will skip this field).
- `json:"email"` → `Field.JSON = "email"` (json codec uses "email" as key).
- `json:"email,omitempty"` → `Field.JSON = "email,omitempty"` (copied verbatim).
- No `json` tag → `Field.JSON = ""` (json codec uses `Field.Name` as key).

**Nested struct fields → `FieldStruct`:**
- If a field's Go type is a struct (not `time.Time`, not a slice), set `Type: fmt.FieldStruct`.
- The generated `Values()` returns the struct value (must also implement `Fielder`).
- The generated `Pointers()` returns a pointer to the struct field.

### 1.2 Reference: generated `Schema()` example

Input:

```go
type User struct {
    ID       string
    Name     string                     `json:"name"`
    Email    string `form:"email"       json:"email"`
    Password string `form:"password"    json:"password"`
    Bio      string `form:"textarea"    json:"bio,omitempty"`
    Age      int64  `form:"-"           json:"age"`
}
```

Expected generated `Schema()`:

```go
func (m *User) Schema() []fmt.Field {
    return []fmt.Field{
        {Name: "ID", Type: fmt.FieldText, PK: true},
        {Name: "Name", Type: fmt.FieldText, NotNull: true, JSON: "name"},
        {Name: "Email", Type: fmt.FieldText, NotNull: true, Input: "email", JSON: "email"},
        {Name: "Password", Type: fmt.FieldText, Input: "password", JSON: "password"},
        {Name: "Bio", Type: fmt.FieldText, Input: "textarea", JSON: "bio,omitempty"},
        {Name: "Age", Type: fmt.FieldInt, Input: "-", JSON: "age"},
    }
}
```

### 1.3 Tests

- `TestOrmcJSONTag`: Parse a struct with `json:` tags, verify `Field.JSON` values in generated `Schema()`.
- `TestOrmcJSONOmitEmpty`: Field with `json:"name,omitempty"` has `JSON: "name,omitempty"`.
- `TestOrmcJSONExclusion`: Field with `json:"-"` has `JSON: "-"` in Schema.
- `TestOrmcJSONNoTag`: Field without `json:` tag has `JSON: ""`.
- `TestOrmcFieldStruct`: Nested struct field generates `Type: fmt.FieldStruct`.

---

## Stage 2: Documentation Updates

← [Stage 1](#stage-1-complete-ormc-tag-parsing-and-field-detection) | Next → [Stage 3](#stage-3-verification-and-publish)

### 2.1 Update `docs/SKILL.md`
- Remove references to bitmask constraints.
- Use `fmt.Field` boolean flags (`PK`, `Unique`, `NotNull`, `AutoInc`).
- Update `Model` interface definition.
- Document `form:` and `json:` tag support in `ormc`.

### 2.2 Update `docs/ARQUITECTURE.md`
- Replace `Columns()` with `Schema() []fmt.Field`.
- Update diagrams/descriptions to reflect `fmt.Fielder` embedding.

### 2.3 Update `README.md`
- Modernize the usage example.

---

## Stage 3: Verification and Publish

← [Stage 2](#stage-2-documentation-updates) | None →

### 3.1 Run tests
```bash
gotest
```

### 3.2 Publish
```bash
gopush 'orm: consolidated migration to fmt.Field and ormc doc updates'
```

---

## Summary of Remaining Changes

| File | Action |
|------|--------|
| `ormc.go` | Parse `json:` tags, detect nested structs, generate `JSON` and `FieldStruct` in `Schema()` |
| `docs/SKILL.md` | Update to current API (bool flags, no bitmask) |
| `docs/ARQUITECTURE.md` | Update to current API (`Schema()` vs `Columns()`) |
| `README.md` | Update documentation |
