# ORMC Flow — Unified Form Validation (Front + Back)

## Core Principle

**ormc asigna widgets por defecto segun el Go type del campo.**
El tag `input:` solo se usa para override o modifiers.
Solo structs con directiva `// ormc:form` o `// ormc:formonly` generan widgets.

---

## Dependency Graph

```mermaid
graph TD
    fmt[fmt - base layer]
    dom[dom - HTML layout only]
    form[form - schema-driven forms]
    form_input[form/input - widgets]
    json[json - codec]
    orm[orm - db + codegen]

    fmt --> dom
    fmt --> form_input
    fmt --> json
    fmt --> orm
    dom --> form
    form_input --> form
    form_input -.->|import in generated code| orm
```

**Reglas:**
- dom: solo layout HTML (Div, H1, Button...). NO tiene funciones de formulario.
- form: unica forma de crear formularios. Usa form.New(parentID, &struct).
- form/input: widgets con validacion. Importado por form y por codigo generado.
- orm: no importa form/input en su codigo fuente. Solo el generado (*_orm.go).

---

## Struct Directives

```mermaid
graph LR
    subgraph Directives
        none["(sin directiva)"]
        form["// ormc:form"]
        formonly["// ormc:formonly"]
    end

    none -->|genera| DB[DB: ModelName, ReadOne, ReadAll]
    none -->|genera| Schema[Schema, Pointers, Validate]
    none -->|NO genera| Widgets[Widgets en Schema]

    form -->|genera| DB2[DB: ModelName, ReadOne, ReadAll]
    form -->|genera| Schema2[Schema, Pointers, Validate]
    form -->|genera| Widgets2[Widgets en Schema]

    formonly -->|NO genera| DB3[DB code]
    formonly -->|genera| Schema3[Schema, Pointers, Validate]
    formonly -->|genera| Widgets3[Widgets en Schema]
```

| Directiva | DB/ORM | Widgets |
| :--- | :--- | :--- |
| _(ninguna)_ | si | no |
| `// ormc:form` | si | si |
| `// ormc:formonly` | no | si |

---

## Widget Assignment (solo structs con form/formonly)

### Default por Go type

| Go type | Widget default |
| :--- | :--- |
| `string` | `input.Text()` |
| `int`, `int64`, `uint`, etc. | `input.Number()` |
| `float32`, `float64` | `input.Number()` |
| `bool` | `input.Checkbox()` |

### Override con tag `input:`

| Tag | Resultado |
| :--- | :--- |
| _(sin tag)_ | default por Go type |
| `input:"email"` | `input.Email()` (override) |
| `input:"textarea"` | `input.Textarea()` (override) |
| `input:"-"` | sin widget (excluido del form) |
| `input:"email,required,min=5"` | `input.Email()` + Permitted modifiers |
| `input:"required,min=5"` | default por Go type + Permitted modifiers |

---

## ormc Run() Flow

```mermaid
sequenceDiagram
    participant Ormc as orm.Ormc
    participant AST as Parser/AST
    participant Gen as Generator

    Ormc->>Ormc: Run()

    rect rgb(240, 240, 240)
    Note over Ormc: Pass 1: collectAllStructs()
    Ormc->>AST: ParseStruct(structAST)
    Note over AST: Detect directive: form / formonly / none
    Note over AST: Extract tags: db, json, input
    AST->>AST: If form/formonly: resolve widget
    Note over AST: 1. input:"-" → skip
    Note over AST: 2. input:"type" → inputWidgets[type]
    Note over AST: 3. no tag → defaultWidgets[goType]
    Note over AST: 4. parse modifiers (required, min, max...)
    AST-->>Ormc: Build FieldInfo
    end

    rect rgb(240, 240, 240)
    Note over Ormc: Pass 2: ResolveRelations()
    Ormc->>Ormc: match FK via db:"ref=parent"
    end

    rect rgb(240, 240, 240)
    Note over Ormc: Pass 3: GenerateForFile()
    Ormc->>Gen: Schema() with Widget if form/formonly
    Ormc->>Gen: Validate() → fmt.ValidateFields()
    Ormc->>Gen: ModelName, Descriptors, ReadOne/ReadAll if !formonly
    Note over Gen: Import form/input only if widgets present
    end

    rect rgb(240, 240, 240)
    Note over Ormc: Pass 4: go get + go mod tidy
    end
```

---

## Validation Flow (Runtime — Same Code Front + Back)

```mermaid
graph TD
    UI[User Input] --> Validate

    subgraph Runtime_Both_Sides
        direction TB
        Validate[fmt.ValidateFields action, model]
        Schema[for each field in Schema]
        FVal[field.Validate]

        Validate --> Schema
        Schema --> FVal

        subgraph Internal_Checks
            NotNull[1. NotNull check]
            WidgetVal[2. Widget.Validate semantic]
            PermVal[3. Permitted.Validate characters]
        end

        FVal --> NotNull
        FVal --> WidgetVal
        FVal --> PermVal
    end

    subgraph Entry_Points
        direction LR
        Front[FRONTEND WASM<br/>form.Validate]
        Back[BACKEND Server<br/>orm.Insert]
    end

    Front --> Validate
    Back --> Validate

    Internal_Checks --> Result[Same result: Frontend == Backend]
```

---

## Tag System (Final)

```
// ormc:form
type User struct {
    ID        string  `db:"pk"`              // string → input.Text() default
    Email     string  `input:"email"`        // override → input.Email()
    Phone     string  `input:"phone"`        // override → input.Phone()
    BirthDate string  `input:"date"`         // override → input.Date()
    Bio       string  `input:"textarea"`     // override → input.Textarea()
    Nick      string  `input:"required,min=3,max=20"` // default text + modifiers
    Age       int     `db:"not_null"`        // int → input.Number() default
    Active    bool                           // bool → input.Checkbox() default
    InternalID string `input:"-"`            // excluded from form
}
```

**Rules:**
- `input:"type"` → explicit widget override
- `input:"type,modifier,..."` → explicit widget + Permitted rules
- `input:"modifier,..."` → Go type default widget + Permitted rules
- `input:"-"` → no widget (excluded from form)
- No `input:` tag → Go type default widget
- `db:"not_null"` → NotNull (also settable via `input:"required"`)

---

## Source Tag Cleanup (rewriteModelTags)

ormc rewrites model.go in-place before/after generation:

| BEFORE | AFTER |
| :--- | :--- |
| `json:"email"` | (removed — redundant with field name) |
| `json:"email,omitempty"` | `json:",omitempty"` |
| `json:",omitempty"` | `json:",omitempty"` (unchanged) |
| `json:"-"` | `json:"-"` (unchanged) |
| `form:"email"` | (removed — legacy tag) |
| `validate:"email,required,min=5"` | (removed — legacy tag) |
| `input:"date"` | `input:"date"` (unchanged) |
| `db:"pk"` | `db:"pk"` (unchanged) |

---

## Custom Inputs (web/input/) — Future

```
web/input/
├── currency.go     ← func Currency() fmt.Widget
└── colorpicker.go  ← func Colorpicker() fmt.Widget

ormc AST-scans this directory:
  1. Finds func *() fmt.Widget
  2. Adds to inputWidgets map with priority over stdlib
  3. Generated code: webinput.Currency()
  4. Import: webinput "yourmodule/web/input"
```

Custom inputs take priority: if project defines `Email()` in web/input/,
it overrides stdlib `input.Email()`.
