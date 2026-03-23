# PLAN: Ormc v3 — Field v3 Migration + Permitted in Schema (tinywasm/orm)

← [README](../README.md) | Depends on: [fmt PLAN.md](../../fmt/docs/PLAN.md)

## Development Rules

- **Standard Library Only:** No external assertion libraries. Use `testing`.
- **Testing Runner:** Use `gotest` (install: `go install github.com/tinywasm/devflow/cmd/gotest@latest`).
- **Max 500 lines per file.** If exceeded, subdivide by domain.
- **Flat hierarchy.** No subdirectories for library code.
- **Documentation First:** Update docs before coding.
- **Publishing:** Use `gopush 'message'` after tests pass and docs are updated.

## Prerequisite

Update `go.mod` to the new `tinywasm/fmt` version (Field v3):

```bash
go get github.com/tinywasm/fmt@latest
```

## Context

Ormc is the code generator that produces `model_orm.go` files. With Field v3:

- `Field.JSON` and `Field.Input` are removed.
- `Field` embeds `fmt.Permitted` — validation rules live in the schema.
- `Field.Validate(value)` is a method, not a standalone function.
- `fmt.ValidateFielder(m)` is the generic validation entry point.
- Ormc generates **schema with Permitted config** for lightweight checks (characters and bounds).
- Ormc generates a **custom `Validate()`** method that invokes `fmt.ValidateFielder(m)` plus standalone validators for structural patterns.

### New tag: `validate:"..."`

Comma-separated list of rules. Ormc maps them to `fmt.Permitted` fields, and triggers standalone format validation inside `Validate()`:

| Tag | Maps to |
|-----|---------|
| `validate:"required"` | `Field.NotNull = true` |
| `validate:"email"` | Function call `validateEmail` |
| `validate:"phone"` | Function call `validatePhone` |
| `validate:"ip"` | Function call `validateIP` |
| `validate:"rut"` | Function call `validateRut` |
| `validate:"date"` | Function call `validateDate` |
| `validate:"min=N"` | `Permitted.Minimum = N` |
| `validate:"max=N"` | `Permitted.Maximum = N` |
| `validate:"letters"` | `Permitted.Letters = true` |
| `validate:"numbers"` | `Permitted.Numbers = true` |
| `validate:"tilde"` | `Permitted.Tilde = true` |
| `validate:"spaces"` | `Permitted.Spaces = true` |

Multiple rules combine: `validate:"required,letters,tilde,spaces,min=2,max=100"`

### Predefined Permitted presets

Common combinations ormc recognizes as shorthand:

| Preset tag | Expands to |
|------------|-----------|
| `validate:"email"` | Generated schema without `Format`, injects call to `validateEmail` in `Validate()` |
| `validate:"phone"` | Generated schema without `Format`, injects call to `validatePhone` in `Validate()` |
| `validate:"name"` | `Permitted{Letters: true, Tilde: true, Spaces: true}` |

---

## Stage 1: Update `FieldInfo` struct

**File:** `ormc.go`

### 1.1 Replace `Input` and `JSON` with Permitted-related fields

```go
// BEFORE
type FieldInfo struct {
    Name       string
    ColumnName string
    Type       FieldType
    PK         bool
    Unique     bool
    NotNull    bool
    AutoInc    bool
    Ref        string
    RefColumn  string
    IsPK       bool
    GoType     string
    Input      string
    JSON       string
}

// AFTER
type FieldInfo struct {
    Name       string
    ColumnName string
    Type       FieldType
    PK         bool
    Unique     bool
    NotNull    bool
    AutoInc    bool
    Ref        string
    RefColumn  string
    IsPK       bool
    GoType     string
    OmitEmpty  bool
    // Permitted config — populated from validate:"..." tag
    Letters    bool
    Tilde      bool
    Numbers    bool
    Spaces     bool
    Extra      []rune
    Minimum    int
    Maximum    int
    Format     string // "email", "phone", etc. (triggers validator call generation)
}
```

---

## Stage 2: Update tag parsing in `ParseStruct`

**File:** `ormc.go`

### 2.1 Replace tag parsing

```go
// BEFORE:
dbTag := ""
formTag := ""
jsonTag := ""

// AFTER:
dbTag := ""
jsonTag := ""
validateTag := ""
```

Remove `form:"..."` parsing entirely. Add `validate:"..."`:

```go
} else if HasPrefix(p, "validate:\"") {
    validateTag = Convert(p).TrimPrefix(`validate:"`).TrimSuffix(`"`).String()
}
```

### 2.2 Parse `json:` for OmitEmpty only

```go
omitEmpty := false
if jsonTag != "" {
    parts := Convert(jsonTag).Split(",")
    for _, p := range parts {
        if p == "omitempty" {
            omitEmpty = true
        }
    }
}
```

### 2.3 Parse `validate:` into Permitted fields

```go
// parseValidateTag maps validate:"..." rules to FieldInfo Permitted fields.
func parseValidateTag(tag string, fi *FieldInfo) {
    parts := Convert(tag).Split(",")
    for _, v := range parts {
        switch {
        case v == "required":
            fi.NotNull = true
        case v == "email":
            fi.Format = "email"
        case v == "phone":
            fi.Format = "phone"
        case v == "ip":
            fi.Format = "ip"
        case v == "rut":
            fi.Format = "rut"
        case v == "date":
            fi.Format = "date"
        case v == "name":
            fi.Letters = true
            fi.Tilde = true
            fi.Spaces = true
        case v == "letters":
            fi.Letters = true
        case v == "numbers":
            fi.Numbers = true
        case v == "tilde":
            fi.Tilde = true
        case v == "spaces":
            fi.Spaces = true
        case HasPrefix(v, "min="):
            n, _ := Convert(v).TrimPrefix("min=").Int64()
            fi.Minimum = int(n)
        case HasPrefix(v, "max="):
            n, _ := Convert(v).TrimPrefix("max=").Int64()
            fi.Maximum = int(n)
        }
    }
}
```

---

## Stage 3: Update `GenerateForFile` — Schema literals with Permitted

**File:** `ormc.go`

### 3.1 Generate Permitted config in Field literals

```go
// BEFORE:
if f.Input != "" {
    buf.Write(Sprintf(", Input: \"%s\"", f.Input))
}
if f.JSON != "" {
    buf.Write(Sprintf(", JSON: \"%s\"", f.JSON))
}

// AFTER:
if f.OmitEmpty {
    buf.Write(", OmitEmpty: true")
}
// Permitted fields (only non-zero values)
writePermittedFields(buf, f)
```

```go
func writePermittedFields(buf *Conv, f FieldInfo) {
    // Use nested Permitted literal
    hasPerm := f.Letters || f.Tilde || f.Numbers || f.Spaces ||
        len(f.Extra) > 0 || f.Minimum > 0 || f.Maximum > 0

    if !hasPerm {
        return
    }

    buf.Write(", Permitted: fmt.Permitted{")
    parts := []string{}
    if f.Letters {
        parts = append(parts, "Letters: true")
    }
    if f.Tilde {
        parts = append(parts, "Tilde: true")
    }
    if f.Numbers {
        parts = append(parts, "Numbers: true")
    }
    if f.Spaces {
        parts = append(parts, "Spaces: true")
    }
    if f.Minimum > 0 {
        parts = append(parts, Sprintf("Minimum: %d", f.Minimum))
    }
    if f.Maximum > 0 {
        parts = append(parts, Sprintf("Maximum: %d", f.Maximum))
    }
    if len(f.Extra) > 0 {
        buf2 := "Extra: []rune{"
        for i, r := range f.Extra {
            if i > 0 { buf2 += ", " }
            buf2 += Sprintf("'%s'", string(r))
        }
        buf2 += "}"
        parts = append(parts, buf2)
    }

    // Join parts
    for i, p := range parts {
        if i > 0 { buf.Write(", ") }
        buf.Write(p)
    }
    buf.Write("}")
}
```

---

## Stage 4: Generate composite `Validate()` method

**File:** `ormc.go` (in `GenerateForFile`, after `Pointers()` generation)

### 4.1 Generate Validate() for structs with any validation

Generate the standard `fmt.ValidateFielder` call for low-level checking, and inject structural validators downstream if required by `Format`.

```go
hasValidation := false
for _, f := range info.Fields {
    if f.NotNull || f.Letters || f.Numbers || f.Tilde || f.Spaces ||
        len(f.Extra) > 0 || f.Minimum > 0 || f.Maximum > 0 || f.Format != "" {
        hasValidation = true
        break
    }
}

if hasValidation {
    buf.Write(Sprintf("func (m *%s) Validate() error {\n", info.Name))
    buf.Write("    if err := fmt.ValidateFielder(m); err != nil { return err }\n")
    for _, f := range info.Fields {
        if f.Format != "" {
            // E.g. "email" -> "ValidateEmail"
            validatorName := "form.Validate" + capitalize(f.Format) // Replaced with target library format
            buf.Write(Sprintf("    if err := %s(m.%s); err != nil { return err }\n", validatorName, capitalize(f.Name)))
        }
    }
    buf.Write("    return nil\n")
    buf.Write("}\n\n")
}
```

---

## Stage 5: Remove `form:` tag from documentation and examples

**Files:** `docs/SKILL.md`, `docs/ARQUITECTURE.md`

- Remove all references to `form:"..."` tag.
- Document new `validate:"..."` tag with all rules and presets.
- Document that `json:"..."` is only used for `omitempty` detection.
- Document that JSON key = `Field.Name` (snake_case column name) always.
- Document one-liner Validate() generation.

---

## Stage 6: Update `db.go` — pre-insert validation

**File:** `db.go`

### 6.1 Call Validate() before Create/Update

```go
func (db *DB) Create(m Model) error {
    if v, ok := m.(fmt.Validator); ok {
        if err := v.Validate(); err != nil {
            return err
        }
    }
    // ... existing create logic
}
```

Same pattern for `Update()`.

---

## Stage 7: Update tests

### 7.1 Update mock models in `tests/`

- Remove `Input:` and `JSON:` from all `fmt.Field` literals.
- Add `OmitEmpty: true` and `Permitted:` where appropriate.
- Add `validate:"..."` tags to test struct definitions.

### 7.2 Add Permitted schema generation tests

Test that `ParseStruct` correctly maps `validate:"email"` to `Permitted{Letters: true, Numbers: true, Extra: ...., Format: FormatEmail}` in the generated schema.

### 7.3 Test one-liner Validate() generation

Verify generated `Validate()` includes `fmt.ValidateFielder(m)` plus specific custom validators if tags request them.

### 7.4 Run tests

```bash
gotest
```

---

## Stage 8: Publish

```bash
gopush 'orm: Field v3 — Permitted in schema, composite Validate, validate tags, remove Input/JSON/form tags'
```

---

## Summary

| Stage | File(s) | Action |
|-------|---------|--------|
| 1 | `ormc.go` | Update `FieldInfo`: remove Input/JSON, add Permitted fields |
| 2 | `ormc.go` | Update tag parsing: remove form, parse validate into Permitted, omitempty from json |
| 3 | `ormc.go` | Generate Field literals with `Permitted:` config |
| 4 | `ormc.go` | Generate composite `Validate()` → `fmt.ValidateFielder(m)` + custom validation links |
| 5 | `docs/` | Update documentation |
| 6 | `db.go` | Call Validate() pre-Create/Update |
| 7 | `tests/` | Update mocks, add Permitted generation tests |
| 8 | — | `gotest` + `gopush` |

## Generated output example

Given:
```go
type User struct {
    ID    string
    Email string `db:"not_null" json:"email,omitempty" validate:"required,email"`
    Name  string `validate:"required,letters,tilde,spaces,min=2,max=100"`
    Phone string `validate:"phone"`
}
```

Ormc generates:
```go
var _schemaUser = []fmt.Field{
    {Name: "id", Type: fmt.FieldText, PK: true},
    {Name: "email", Type: fmt.FieldText, NotNull: true, OmitEmpty: true},
    {Name: "name", Type: fmt.FieldText, NotNull: true,
        Permitted: fmt.Permitted{Letters: true, Tilde: true, Spaces: true, Minimum: 2, Maximum: 100}},
    {Name: "phone", Type: fmt.FieldText},
}

func (m *User) Schema() []fmt.Field { return _schemaUser }

func (m *User) Pointers() []any {
    return []any{&m.ID, &m.Email, &m.Name, &m.Phone}
}

func (m *User) Validate() error {
    if err := fmt.ValidateFielder(m); err != nil { return err }
    if err := form.ValidateEmail(m.Email); err != nil { return err }
    if err := form.ValidatePhone(m.Phone); err != nil { return err }
    return nil
}
```

**Comparison with old approach (custom Validate body):**

| Aspect | Old (custom body) | New (schema + composite Validate) |
|--------|-------------------|--------------------------|
| Generated lines | ~20 per struct | ~4 per struct |
| Validation logic location | Scattered in generated code | Centralized in `fmt.Field.Validate` + `fmt.Permitted.Validate` |
| Adding new rule | Change ormc generator code | Add to `validate:"..."` tag, ormc maps to Permitted |
| Runtime behavior | Identical | Identical |
