# PLAN: ORM Tests and Documentation Completion

## Development Rules

- **Testing Runner:** `gotest`
- **Publishing:** `gopush 'message'`

## Context

Code was fully migrated. The following is confirmed implemented:
- `field_ext.go`: `FieldExt` struct replacing `field_type.go`
- `model.go`: `Model` embeds `fmt.Fielder` + `TableName()`
- `ormc.go`: Parses `form:`, `json:` tags; detects `FieldStruct`; generates `FormName()`; supports `// ormc:formonly`

**What remains:**
1. Tests for `json:` tags and `FieldStruct` (mock models `UserWithJSON` and `WithPointers` already exist).
2. `docs/SKILL.md` — still references old bitmask API; must be updated.
3. `docs/ARQUITECTURE.md` — still references old `Columns()` and bitmask; must be updated.
4. `README.md` — missing links and `PLAN.md` doc.

---

## Stage 1: Add Missing Tests

← None | Next → [Stage 2](#stage-2-update-skillmd)

Add to `tests/ormc_test.go` inside `TestOrmc`:

```go
t.Run("JSON tags", func(t *testing.T) {
    err := orm.NewOrmc().GenerateForStruct("UserWithJSON", "mock_generator_model.go")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    outFile := "mock_generator_model_orm.go"
    contentBytes, err := os.ReadFile(outFile)
    if err != nil {
        t.Fatalf("failed to read: %v", err)
    }
    defer os.Remove(outFile)
    content := string(contentBytes)
    expected := []string{
        `JSON: "id"`,
        `JSON: "name"`,
        `JSON: "email"`,
        `JSON: "bio,omitempty"`,
    }
    for _, e := range expected {
        if !strings.Contains(content, e) {
            t.Errorf("missing: %s\nContent:\n%s", e, content)
        }
    }
})

t.Run("FieldStruct for nested struct", func(t *testing.T) {
    err := orm.NewOrmc().GenerateForStruct("UserWithJSON", "mock_generator_model.go")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    outFile := "mock_generator_model_orm.go"
    contentBytes, err := os.ReadFile(outFile)
    if err != nil {
        t.Fatalf("failed to read: %v", err)
    }
    defer os.Remove(outFile)
    content := string(contentBytes)
    if !strings.Contains(content, "fmt.FieldStruct") {
        t.Errorf("expected FieldStruct in generated output, got:\n%s", content)
    }
})
```

```bash
gotest
```

---

## Stage 2: Update `docs/SKILL.md`

← [Stage 1](#stage-1-add-missing-tests) | Next → [Stage 3](#stage-3-update-arquitecturemd)

Replace the outdated content with the current API:

- **Model Interface:** `fmt.Fielder` + `TableName()` (not `Columns()`).
- **Schema Field Types:** Table using `fmt.FieldText`, `fmt.FieldInt`, etc. (not `TypeText`/`TypeInt64`).
- **Constraints:** Bool fields (`PK`, `Unique`, `NotNull`, `AutoInc`) — **no bitmask**.
- **`FieldExt`:** Document `Ref` and `RefColumn` for FK metadata.
- **`FormName()`:** Generated for every struct; returns lowercase snake_case of struct name.
- **`form:` and `json:` tags:** Document the tag → field mapping rules.
- **`// ormc:formonly`:** Document the directive.

---

## Stage 3: Update `docs/ARQUITECTURE.md`

← [Stage 2](#stage-2-update-skillmd) | Next → [Stage 4](#stage-4-update-readmemd)

- Update section 3.1 `Model` interface: Replace `Columns() []string` with `Schema() []fmt.Field`.
- Remove bitmask `Constraint` references; use bool fields.
- Note that `Model` now embeds `fmt.Fielder` from `tinywasm/fmt`.

---

## Stage 4: Update `README.md`

← [Stage 3](#stage-3-update-arquitecturemd) | Next → [Stage 5](#stage-5-publish)

- Add `PLAN.md` link if created.
- Ensure all docs are linked.

---

## Stage 5: Publish

← [Stage 4](#stage-4-update-readmemd) | None →

```bash
gopush 'orm: add json/FieldStruct tests, update SKILL and ARCHITECTURE docs'
```
