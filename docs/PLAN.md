# PLAN: input: Tag Support + Widget Defaults — tinywasm/orm

**Module:** `github.com/tinywasm/orm`
**Breaking change:** Yes — removes `Format`, `parseValidateTag`, hardcoded form validators.
**Status:** Pending
**Blocker:** None (dom cleanup is independent, can run in parallel)
**Flow diagram:** diagrams/ORMC_FLOW.md

---

## Core Design Decisions

1. **No Resolver, no aliases.** Widget assigned by Go type default + `input:` tag override.
2. **Directives control widget generation:**
   - `// ormc:form` = DB + widgets
   - `// ormc:formonly` = widgets, no DB (already exists)
   - No directive = DB only, no widgets
3. **Go type defaults** (only when struct has `form` or `formonly` directive):
   - `string` → `input.Text()`
   - `int`, `int64`, `uint`, etc. → `input.Number()`
   - `bool` → `input.Checkbox()`
   - `float32`, `float64` → `input.Number()`
   - FieldStruct fields → no widget (nested structs are not form inputs)
4. **`input:"email"`** overrides the Go type default.
5. **`input:"-"`** excludes field from form (no widget).
6. **Modifiers** in tag: `input:"email,required,min=5,max=100"`.
7. **Modifiers-only tag** (`input:"required,min=5"`) keeps Go type default widget + applies modifiers.

---

## Development Rules

- **Standard Library Only:** No external assertion libraries. Use `testing`.
- **Testing Runner:** Use `gotest`.
- **Build tag:** `ormc.go` uses `//go:build !wasm` — all ormc code is backend-only.
- **Max 500 lines per file.**
- **TinyGo Compatible for non-ormc code.**

---

## Stage 1: Update FieldInfo + Tag Parsing

### 1.1 Update FieldInfo struct

- [ ] Add `WidgetConstructor string` (e.g., `"input.Email()"`)
- [ ] Remove `Format string`

### 1.2 Go type → widget default map

```go
var defaultWidgets = map[string]string{
    "string":  "input.Text()",
    "int":     "input.Number()",
    "int32":   "input.Number()",
    "int64":   "input.Number()",
    "uint":    "input.Number()",
    "uint32":  "input.Number()",
    "uint64":  "input.Number()",
    "float32": "input.Number()",
    "float64": "input.Number()",
    "bool":    "input.Checkbox()",
}
```

Note: FieldStruct types (nested structs, slices) are NOT in this map — they never get widgets.

### 1.3 Input type override map

```go
var inputWidgets = map[string]string{
    "text":     "input.Text()",
    "email":    "input.Email()",
    "password": "input.Password()",
    "textarea": "input.Textarea()",
    "phone":    "input.Phone()",
    "number":   "input.Number()",
    "date":     "input.Date()",
    "hour":     "input.Hour()",
    "ip":       "input.IP()",
    "rut":      "input.Rut()",
    "address":  "input.Address()",
    "checkbox": "input.Checkbox()",
    "datalist": "input.Datalist()",
    "select":   "input.Select()",
    "radio":    "input.Radio()",
    "filepath": "input.Filepath()",
    "gender":   "input.Gender()",
}
```

### 1.4 Replace tag parsing

- [ ] Remove `parseValidateTag` function
- [ ] Remove `validate:` and `form:` tag parsing branches
- [ ] Add `input:` tag parsing branch — extract raw string
- [ ] New `parseInputModifiers(inputTag string, fi *FieldInfo)` — parses only Permitted modifiers (required, min=, max=, letters, numbers, spaces, tilde, name). Skips first segment if it's not a modifier (it's the type override, already handled).

### 1.5 Widget resolution in ParseStruct

```go
// Only if struct has form or formonly directive
if isForm {
    inputTag := "" // extracted from tags
    if inputTag == "-" {
        // No widget — field excluded from form
    } else if inputTag != "" {
        typeName := firstSegment(inputTag) // before first ","
        if !isModifier(typeName) {
            // Explicit type override
            if ctor, ok := inputWidgets[typeName]; ok {
                fi.WidgetConstructor = ctor
            } else {
                o.log("Warning: unknown input type", typeName, "for field", fi.Name)
                // Fall back to Go type default
                if ctor, ok := defaultWidgets[fi.GoType]; ok {
                    fi.WidgetConstructor = ctor
                }
            }
        } else {
            // Modifiers-only tag (e.g., input:"required,min=5")
            // Assign default widget by Go type
            if ctor, ok := defaultWidgets[fi.GoType]; ok {
                fi.WidgetConstructor = ctor
            }
        }
        parseInputModifiers(inputTag, &fi)
    } else {
        // No input tag — default by Go type
        if ctor, ok := defaultWidgets[fi.GoType]; ok {
            fi.WidgetConstructor = ctor
        }
    }
}
```

---

## Stage 2: Update Struct Directives

### 2.1 Parse `// ormc:form` directive

- [ ] Add `IsForm bool` to `StructInfo` (alongside existing `FormOnly bool`)
- [ ] Detect `// ormc:form` in doc comments (same pattern as existing `// ormc:formonly`)
- [ ] `formonly` implies `IsForm = true`

### 2.2 Logic

```
IsForm = hasDirective("form") || hasDirective("formonly")
FormOnly = hasDirective("formonly")
```

Widget resolution only runs when `IsForm == true`.

---

## Stage 3: Source File Tag Cleanup (ormc_tags.go)

- [ ] Create `ormc_tags.go` with `rewriteTagValue()` and `rewriteModelTags()`
- [ ] Rules: remove `json:"name"`, rewrite `json:"name,omitempty"` to `json:",omitempty"`, remove `form:` and `validate:` tags entirely, keep `input:`, `db:`, `json:"-"`, `json:",omitempty"`
- [ ] Uses `go/parser` + `go/token` for AST positions, raw byte replacement
- [ ] Call from `Run()` **BEFORE Pass 1** (collectAllStructs) — so ParseStruct reads already-cleaned tags

---

## Stage 4: Code Generation Updates (ormc_generate.go)

### 4.1 Schema generation with Widget

When `fi.WidgetConstructor != ""`:

```go
// Generated:
{Name: "email", Type: fmt.FieldText, NotNull: true,
    Widget: input.Email(),
    Permitted: fmt.Permitted{Minimum: 5}}
```

### 4.2 Import handling

- [ ] Replace `hasFormat` check with `hasWidget` check — scan all fields for `WidgetConstructor != ""`
- [ ] When `hasWidget`: add `"github.com/tinywasm/form/input"` import (replaces old `"github.com/tinywasm/form"`)
- [ ] Remove `hasFormat` variable and `"github.com/tinywasm/form"` import entirely
- [ ] Remove `form.Validate{Format}()` generation entirely (lines 115-133 in current ormc_generate.go)

### 4.3 Validate() generation

Validate() is generated when `IsForm || hasPermittedRules`:
- `IsForm` structs always get Validate() — widgets validate at runtime via `field.Widget.Validate()`
- Non-form structs only get Validate() if they have explicit Permitted rules (min, max, letters, etc.)

Generated body is always:

```go
func (m *User) Validate(action byte) error {
    return fmt.ValidateFields(action, m)
}
```

No more format-specific `form.Validate{Format}()` calls.

---

## Stage 5: Custom Inputs Discovery (Future)

- [ ] In `Run()`, scan `web/input/` via AST for `func *() fmt.Widget`
- [ ] Add discovered constructors to `inputWidgets` map with priority over stdlib
- [ ] Generated code uses `webinput.X()` with import `webinput "module/web/input"`

**Note:** Future enhancement. Initial version works without custom inputs.

---

## Stage 6: Update cmd/ormc/main.go

No changes needed — ormc handles everything internally now.

```go
// main.go stays the same:
func main() {
    o := orm.NewOrmc()
    o.SetLog(func(messages ...any) {
        fmt.Fprintln(os.Stderr, messages...)
    })
    if err := o.Run(); err != nil {
        log.Fatalf("ormc: %v", err)
    }
}
```

---

## Stage 7: Tests

### Tag rewrite tests (tests/ormc_tags_test.go)
- [ ] Golden file pattern: testdata/before_model.go → after_model.go
- [ ] 11 cases: json name removal, omitempty rewrite, form/validate removal, input preserved

### Widget assignment tests (tests/ormc_test.go)
- [ ] `// ormc:form` struct, string field, no tag → `input.Text()`
- [ ] `// ormc:form` struct, int64 field, no tag → `input.Number()`
- [ ] `// ormc:form` struct, bool field, no tag → `input.Checkbox()`
- [ ] `// ormc:form` struct, string field, `input:"email"` → `input.Email()`
- [ ] `// ormc:form` struct, string field, `input:"-"` → no widget
- [ ] `// ormc:form` struct, FieldStruct field → no widget (nested structs excluded)
- [ ] `// ormc:formonly` struct → widgets generated, no ORM code
- [ ] Struct without directive → no widgets at all
- [ ] Unknown input type `input:"foobar"` → warning, falls back to Go type default

### Modifier parsing tests (tests/ormc_test.go)
- [ ] `input:"email,required,min=5"` → WidgetConstructor + NotNull + Minimum
- [ ] `input:"required,min=5"` → Go type default widget + NotNull + Minimum
- [ ] `input:"textarea,letters,spaces"` → WidgetConstructor + Letters + Spaces

### Schema generation tests (tests/ormc_test.go)
- [ ] Generated code contains `Widget: input.Email()` for tagged fields
- [ ] Generated code imports `"github.com/tinywasm/form/input"` (not `"github.com/tinywasm/form"`)
- [ ] Generated Validate() has NO format-specific calls — only `fmt.ValidateFields()`
- [ ] Struct without form directive → no input import

---

## Stage 8: Cleanup & Publish

- [ ] Remove dead code: `Format`, `parseValidateTag`, `form.Validate*` generation, `hasFormat` check
- [ ] Simplify Pass 4 in Run(): remove hardcoded `go get` for fmt/orm/form — replace with just `go mod tidy` (tidy already resolves imports from generated code)
- [ ] Update test models: replace `validate:`/`form:` tags, add `// ormc:form` directive
- [ ] Run `gotest` — all tests pass

---

## Files to Modify

| File | Change |
|---|---|
| ormc.go | FieldInfo: add WidgetConstructor, remove Format. StructInfo: add IsForm. Remove parseValidateTag, add parseInputModifiers. Widget resolution logic. Parse `// ormc:form` directive. |
| ormc_tags.go | NEW — tag rewriting functions |
| ormc_generate.go | Replace hasFormat with hasWidget. Import form/input instead of form. Generate Widget in Schema. Remove format validator calls. Validate() generation logic. |
| ormc_handler.go | No changes |
| validate.go | Review only — no changes expected |
| tests/ormc_tags_test.go | NEW — golden file tag rewrite tests |
| tests/ormc_test.go | Add widget, modifier, and schema generation tests |
| tests/models.go | Update tags, add `// ormc:form` directives |
| tests/testdata/before_model.go | NEW — golden file |
| tests/testdata/after_model.go | NEW — golden file |

---

## go.mod Update

```bash
go get github.com/tinywasm/form@v0.1.0
go get github.com/tinywasm/fmt@v0.21.1
go mod tidy
```
