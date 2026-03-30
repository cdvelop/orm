# PLAN: input: Tag Support in ormc — tinywasm/orm

**Module:** `github.com/tinywasm/orm`
**Breaking change:** Yes — removes `form:` and `validate:` tag parsing; replaces with `input:` tag.
**Execution order:** Requires `tinywasm/fmt` PLAN_WIDGET.md and `tinywasm/form` PLAN.md to be published first.

---

## Context

`tinywasm/orm` provides `ormc` — a code generator that reads `model.go` / `models.go` files and generates `model_orm.go` with `Schema()`, `Pointers()`, `Validate()`, typed descriptors (`T_`), and typed read helpers (`ReadOneT`, `ReadAllT`).

### Current Tag System (to be removed)

| Tag | Purpose |
|---|---|
| `db:"pk,unique,..."` | DB constraints — **stays** |
| `json:"name"` | JSON field name — **remove name, keep omitempty** |
| `json:",omitempty"` | Omit empty in JSON — **keep** |
| `json:"-"` | Exclude from JSON — **keep** |
| `form:"email"` | HTML input type hint — **remove entirely** |
| `validate:"email,required,min=2"` | Validation rules — **remove entirely** |

### New Tag System

| Tag | Purpose |
|---|---|
| `db:"..."` | DB constraints — unchanged |
| `json:",omitempty"` | Omit empty — unchanged |
| `json:"-"` | Exclude from JSON — unchanged |
| `input:"email"` | Input type + implicit semantic validation |
| `input:"email,required,min=2"` | Input type + explicit Permitted rules |
| `input:"text,required,min=2"` | Any type + Permitted modifiers |

### `input:` Tag Parsing Rules

```
input:"<type>[,modifier1[,modifier2...]]"
```

- First segment = input type name (e.g., `email`, `textarea`, `text`, `phone`, `rut`, `ip`)
- Remaining segments = Permitted modifiers using the same syntax as the old `validate:` tag:
  - `required` → `NotNull = true`
  - `min=N` → `Permitted.Minimum = N`
  - `max=N` → `Permitted.Maximum = N`
  - `letters` → `Permitted.Letters = true`
  - `numbers` → `Permitted.Numbers = true`
  - `spaces` → `Permitted.Spaces = true`
  - `tilde` → `Permitted.Tilde = true`

The input type name drives two things:
1. Which `fmt.Widget` constructor to call in the generated schema (`Widget: input.NewEmail()`)
2. Implicit semantic validation (via the input type's own `Validate()` method, called at runtime by `Field.Validate()`)

**Important:** `required` in `input:` tag sets `NotNull = true` on the field — this is used by `Field.Validate()` for the empty-value check, not by the Widget itself.

---

## Development Rules

- **Standard Library Only:** No external assertion libraries. Use `testing`.
- **Testing Runner:** Use `gotest` (`go install github.com/tinywasm/devflow/cmd/gotest@latest`).
- **Build tag:** `ormc.go` uses `//go:build !wasm` — all ormc code is backend-only.
- **Max 500 lines per file.**
- **TinyGo Compatible for non-ormc code:** `db.go`, `qb.go`, `conditions.go`, etc. must not use stdlib `fmt`/`strings`/`strconv`. `ormc.go` is backend-only and may use stdlib freely.
- **Publishing:** Use `gopush 'message'` after tests pass.

---

## Part 1: Source File Cleanup (modifying `model.go` / `models.go`)

`ormc` must clean struct tags **in-place** in the source files before (or after) generating `model_orm.go`.

### 1.1 Tag Rewrite Rules

For each struct field tag in every processed `model.go` / `models.go`:

| Found tag | Action |
|---|---|
| `json:"fieldname"` | Remove tag entirely |
| `json:"fieldname,omitempty"` | Rewrite to `json:",omitempty"` |
| `json:",omitempty"` | Keep unchanged |
| `json:"-"` | Keep unchanged |
| `form:"anything"` | Remove tag entirely |
| `validate:"anything"` | Remove tag entirely |
| `input:"anything"` | Keep unchanged (this is the new tag) |
| `db:"anything"` | Keep unchanged |

### 1.2 Implementation

Implement tag rewriting as a new function `rewriteModelTags(goFile string) error` in `ormc.go` (or a new file `ormc_tags.go`):

1. Parse the file with `go/parser` and `go/token`, preserving comments.
2. For each struct field, extract and rewrite the tag string.
3. Use the file set positions to perform string replacement directly on the raw file bytes (safer than AST rewriting which loses formatting).
4. Write the modified bytes back to the file.

**Tag string manipulation approach (no AST rewrite):**
- Read raw file as `[]byte`
- For each field's tag position (from AST: `field.Tag.ValuePos` and `field.Tag.Value`), compute byte offsets
- Apply the rewrite rules on the raw tag string
- Write back

**Rewrite function for a single tag string:**

```go
// rewriteTagValue takes the raw backtick tag value (without backticks)
// and returns the cleaned version.
func rewriteTagValue(raw string) string {
    // parse space-separated key:"value" pairs
    // apply rules per pair
    // reassemble
}
```

### 1.3 When to Run

`rewriteModelTags` runs on every `model.go` / `models.go` found during `Run()`, before or after generating `model_orm.go`. Order does not matter since tag rewrite only touches source files, not generated files.

---

## Part 2: `ormc.go` — Parse `input:` Tag

### 2.1 Update `FieldInfo` struct

Add `WidgetName string` field. Remove `Format string` (was used for email/phone/ip/rut/date format validators):

```go
type FieldInfo struct {
    Name          string
    ColumnName    string
    Type          fmt.FieldType
    PK            bool
    Unique        bool
    NotNull       bool
    AutoInc       bool
    Ref           string
    RefColumn     string
    IsPK          bool
    GoType        string
    OmitEmpty     bool
    WidgetName string   // ← NEW: e.g., "email", "textarea". Empty = no UI binding.
    // Permitted config:
    Letters bool
    Tilde   bool
    Numbers bool
    Spaces  bool
    Extra   []rune
    Minimum int
    Maximum int
    // Format string  ← REMOVED
}
```

### 2.2 Update tag parsing in `ParseStruct()`

Replace the `validateTag` and `jsonTag` parsing block with `inputTag` parsing:

```go
dbTag := ""
inputTag := ""
omitEmpty := false
jsonExclude := false

if field.Tag != nil {
    tagVal := // extract raw tag value without backticks
    parts := // split by space-separated key:"value" pairs
    for _, p := range parts {
        switch {
        case strings.HasPrefix(p, `db:"`):
            dbTag = // extract value
        case strings.HasPrefix(p, `input:"`):
            inputTag = // extract value
        case p == `json:"-"`:
            jsonExclude = true
        case strings.HasPrefix(p, `json:"`):
            val := // extract value
            for _, part := range strings.Split(val, ",") {
                if part == "omitempty" {
                    omitEmpty = true
                }
            }
        }
    }
}
```

Remove all `form:` tag parsing (it no longer exists).

### 2.3 Update `parseValidateTag` → rename to `parseInputTag`

Rename `parseValidateTag` to `parseInputTag`. Change it to:
1. Extract the first segment as `WidgetName`
2. Pass remaining segments to the existing Permitted parsing logic

```go
func parseInputTag(tag string, fi *FieldInfo) {
    parts := split(tag, ",")
    if len(parts) == 0 {
        return
    }
    fi.WidgetName = parts[0] // e.g., "email"
    // Parse remaining as Permitted modifiers:
    for _, v := range parts[1:] {
        switch {
        case v == "required":
            fi.NotNull = true
        case v == "letters":
            fi.Letters = true
        case v == "numbers":
            fi.Numbers = true
        case v == "spaces":
            fi.Spaces = true
        case v == "tilde":
            fi.Tilde = true
        case strings.HasPrefix(v, "min="):
            n, _ := parseIntSuffix(v, "min=")
            fi.Minimum = n
        case strings.HasPrefix(v, "max="):
            n, _ := parseIntSuffix(v, "max=")
            fi.Maximum = n
        }
    }
}
```

Call `parseInputTag(inputTag, &fi)` instead of `parseValidateTag(validateTag, &fi)`.

---

## Part 3: Input Type Lookup

When `fi.WidgetName != ""`, `ormc` must resolve which package the constructor comes from.

**Lookup order: custom inputs take priority over stdlib.** If a project defines a custom input with the same name as a stdlib input, the custom one wins. This allows projects to override stdlib behavior for any input type.

### 3.1 Custom Inputs (web/inputs/) — checked FIRST

Scan `web/inputs/` in the project root directory for a Go file containing an exported function matching `New<CamelCaseName>() fmt.Widget`. **This check runs before stdlib lookup.**

**AST scan for custom inputs:**

```go
func (o *Ormc) findCustomInput(name string) (constructor string, found bool) {
    webInputsDir := filepath.Join(o.rootDir, "web", "inputs")
    // Check if dir exists — if not, skip
    // Parse all *.go files in that dir
    // Look for func declarations matching: func New<X>() fmt.Widget
    // Match X (lowercased) against name (lowercased)
    // If found: return "webinputs.New<X>()", true
    return "", false
}
```

Import alias for the custom package: `webinputs "yourmodule/web/inputs"` — detect the module name from `go.mod` in `rootDir`.

### 3.2 Stdlib Inputs (tinywasm/form/input) — checked SECOND

Only if no custom input was found for the name, fall back to the stdlib map:

```go
// Keys must match the value returned by input.Type() (which equals htmlName in Base).
var stdlibInputs = map[string]string{
    "text":     "input.NewText()",
    "email":    "input.NewEmail()",
    "password": "input.NewPassword()",
    "textarea": "input.NewTextarea()",
    "phone":    "input.NewPhone()",
    "number":   "input.NewNumber()",
    "date":     "input.NewDate()",
    "hour":     "input.NewHour()",
    "ip":       "input.NewIp()",
    "rut":      "input.NewRut()",
    "address":  "input.NewAddress()",
    "checkbox": "input.NewCheckbox()",
    "datalist": "input.NewDatalist()",
    "select":   "input.NewSelect()",
    "radio":    "input.NewRadio()",
    "filepath": "input.NewFilepath()",
    "gender":   "input.NewGender()",
}
```

### 3.3 Not Found

If the name is not found in `web/inputs/` AND not in stdlib:
- Log a warning: `Warning: unknown input type "<name>" for field <Struct>.<Field>. Field will have no Widget.`
- Set `fi.WidgetName = ""` (field renders without Widget in schema)

---

## Part 4: Code Generation — Schema with Widget

### 4.1 Update `GenerateForFile()` / schema generation

When generating the `Schema()` method body for a field with `fi.WidgetName != ""`:

```go
// Generated output (example for email field):
{Name: "Email", Type: fmt.FieldText, NotNull: true, Widget: input.NewEmail(), Permitted: fmt.Permitted{Minimum: 5}}
```

### 4.2 Imports in generated file

When any field in the struct has an `WidgetName`:
- Add `"github.com/tinywasm/form/input"` to the generated file's import block (for stdlib inputs)
- Add `webinputs "<module>/web/inputs"` for custom inputs (if any)

The import is only added if at least one field uses it — no unused imports.

### 4.3 Remove `Validate()` format-specific calls

The current `Validate(action byte)` generated method calls format-specific functions like `form.ValidateEmail(...)`. This is now handled automatically by `Field.Validate()` calling `field.Widget.Validate(value)` at runtime.

**Remove** the `form.ValidateStructFormats()` call from generated `Validate()` methods. The generated method becomes:

```go
func (m *User) Validate(action byte) error {
    return fmt.ValidateFields(action, m)
}
```

No more format-specific calls. No more import of `tinywasm/form` in generated files for validation purposes.

### 4.4 Update `validate.go` in orm package

Check `validate.go` — it currently references format validation. Remove any format-specific validation calls from the orm layer; validation is now fully handled by `fmt.ValidateFields()` + `Widget.Validate()`.

---

## Part 5: Remove Old Code

- Delete `parseValidateTag` (replaced by `parseInputTag`)
- Remove `Format string` from `FieldInfo`
- Remove `form:` tag parsing from `ParseStruct()`
- Remove `validate:` tag parsing from `ParseStruct()`
- Remove any code generation that emits format-validator calls

---

## Files to Modify

| File | Change |
|---|---|
| `ormc.go` | Update `FieldInfo`, `ParseStruct()`, rename `parseValidateTag→parseInputTag`, update code generation for Widget |
| `ormc_tags.go` (new) | `rewriteTagValue()`, `rewriteModelTags()` — source file tag cleanup |
| `validate.go` | Remove format-specific validation if present |
| `ormc_handler.go` | Call `rewriteModelTags()` from `Run()` |

---

## Tests

### Tag rewrite tests (`ormc_tags_test.go`)

Use golden file pattern: write a `testdata/before_model.go` and `testdata/after_model.go`, run `rewriteModelTags`, compare output.

Test cases:
1. `json:"name"` → removed
2. `json:"name,omitempty"` → `json:",omitempty"`
3. `json:",omitempty"` → unchanged
4. `json:"-"` → unchanged
5. `form:"email"` → removed
6. `validate:"email,required"` → removed
7. `input:"email,required"` → unchanged
8. Field with multiple tags — only json/form/validate affected, db and input untouched
9. Field with no tags → unchanged

### Input tag parsing tests (`ormc_test.go`)

Add to existing test suite:
1. `input:"email"` → `WidgetName="email"`, no Permitted rules
2. `input:"email,required"` → `WidgetName="email"`, `NotNull=true`
3. `input:"text,required,min=2,max=100"` → `WidgetName="text"`, `NotNull=true`, `Minimum=2`, `Maximum=100`
4. `input:"textarea,letters,spaces"` → `WidgetName="textarea"`, `Letters=true`, `Spaces=true`
5. Unknown input type → `WidgetName=""`, warning logged

### Input type lookup tests (`ormc_test.go`)

6. Custom input with name `"email"` in `web/inputs/` → custom constructor used, stdlib `input.NewEmail()` NOT used (custom overrides stdlib)
7. Custom input with name `"rut"` in `web/inputs/` + stdlib also has `"rut"` → custom wins
8. No `web/inputs/` directory → falls back to stdlib without error
9. Name not in custom and not in stdlib → `WidgetName=""`, warning logged

### Schema generation tests

Verify that generated `model_orm.go` for a struct with `input:"email"` contains:
- `Widget: input.NewEmail()` in the schema field
- Import of `github.com/tinywasm/form/input`
- No format-specific validator calls in `Validate()`

Verify that when a custom `NewEmail()` exists in `web/inputs/`:
- Generated code uses `webinputs.NewEmail()` not `input.NewEmail()`
- Import of `yourmodule/web/inputs` present, `github.com/tinywasm/form/input` absent (or only for other fields)

---

## go.mod Update

```bash
go get github.com/tinywasm/fmt@<new_version_from_PLAN_FIELD_INPUT>
go mod tidy
```

`tinywasm/orm` does NOT import `tinywasm/form` directly — only the generated `model_orm.go` files import `tinywasm/form/input`. The `orm` package itself remains independent.

---

## Publishing

```bash
gopush 'feat: replace form:/validate: tags with unified input: tag; add source file tag cleanup to ormc'
```
