# PLAN: orm — Validate(action byte) generado por ormc [COMPLETED]

← [README](../README.md) | Depende de: [fmt PLAN.md](../../fmt/docs/PLAN.md)

## Development Rules

- **Standard Library Only:** No external assertion libraries. Use `testing`.
- **Testing Runner:** Use `gotest` (install: `go install github.com/tinywasm/devflow/cmd/gotest@latest`).
- **Max 500 lines per file.** If exceeded, subdivide by domain.
- **Flat hierarchy.** No subdirectories for library code.
- **Documentation First:** Update docs before coding.


## Prerequisite

```bash
go get github.com/tinywasm/fmt@latest  # versión con ValidateFields(action, f)
```

## Contexto

La validación de campos **no** pertenece al DB — pertenece a la frontera del sistema (crudp/form).
`db.Create()` y `db.Update()` no deben validar valores: reciben datos internos ya validados.

Los cambios en orm son:
1. **Eliminar** las llamadas a `v.Validate()` en `db.go`
2. **Actualizar** ormc para generar `Validate(action byte)` con la nueva firma

---

## Stage 1: Eliminar validación de `db.go`

**File:** `db.go`

### 1.1 `Create()` — eliminar bloque Validator

```go
// ANTES:
func (db *DB) Create(m Model) error {
    if v, ok := m.(fmt.Validator); ok {
        if err := v.Validate(); err != nil {
            return err
        }
    }
    if err := validateQuery(ActionCreate, m); err != nil {
        return err
    }
    // ...

// DESPUÉS:
func (db *DB) Create(m Model) error {
    if err := validateQuery(ActionCreate, m); err != nil {
        return err
    }
    // ...
```

### 1.2 `Update()` — eliminar bloque Validator

```go
// ANTES:
func (db *DB) Update(m Model, cond Condition, rest ...Condition) error {
    if v, ok := m.(fmt.Validator); ok {
        if err := v.Validate(); err != nil {
            return err
        }
    }
    if err := validateQuery(ActionUpdate, m); err != nil {
        return err
    }
    // ...

// DESPUÉS:
func (db *DB) Update(m Model, cond Condition, rest ...Condition) error {
    if err := validateQuery(ActionUpdate, m); err != nil {
        return err
    }
    // ...
```

**Justificación:** La validación de campos ocurre en la frontera del sistema
(crudp → `ValidateData(action, payload)` → `fmt.ValidateFields(action, data)`).
El DB es capa interna — confía en que los datos ya fueron validados.
`validateQuery(ActionCreate, m)` se mantiene porque valida estructura de query (tabla, schema), no valores.

### 1.3 Renombrar `validate` → `validateQuery`

**File:** `validate.go`

```go
// ANTES:
func validate(action Action, m Model) error {

// DESPUÉS:
func validateQuery(action Action, m Model) error {
```

Actualizar todas las llamadas en `db.go`, `qb.go` y cualquier otro archivo que use `validate(`:

```
validate(ActionCreate, m)       → validateQuery(ActionCreate, m)
validate(ActionUpdate, m)       → validateQuery(ActionUpdate, m)
validate(ActionDelete, m)       → validateQuery(ActionDelete, m)
validate(ActionReadOne, m)      → validateQuery(ActionReadOne, m)
validate(ActionReadAll, m)      → validateQuery(ActionReadAll, m)
validate(ActionCreateTable, m)  → validateQuery(ActionCreateTable, m)
validate(ActionDropTable, m)    → validateQuery(ActionDropTable, m)
validate(ActionCreateDatabase, m) → validateQuery(ActionCreateDatabase, m)
```

---

## Stage 2: Actualizar `ormc.go` — generar `Validate(action byte)`

**File:** `ormc.go`, en `GenerateForFile`

El método generado `Validate()` acepta `action byte` y lo pasa a `fmt.ValidateFields`.
Los format validators (`form.ValidateEmail`, etc.) solo se ejecutan en `'c'` y `'u'`:

```go
// ANTES (generado):
func (m *User) Validate() error {
    if err := fmt.ValidateFielder(m); err != nil { return err }
    if err := form.ValidateEmail(m.Email); err != nil { return err }
    return nil
}

// DESPUÉS (generado):
func (m *User) Validate(action byte) error {
    if err := fmt.ValidateFields(action, m); err != nil { return err }
    if action == 'c' || action == 'u' {
        if err := form.ValidateEmail(m.Email); err != nil { return err }
    }
    return nil
}
```

En `ormc.go`, actualizar la generación:

```go
buf.Write(Sprintf("func (m *%s) Validate(action byte) error {\n", info.Name))
buf.Write("\tif err := fmt.ValidateFields(action, m); err != nil { return err }\n")

// Format validators solo en 'c' y 'u'
hasFormat := false
for _, f := range info.Fields {
    if f.Format != "" {
        hasFormat = true
        break
    }
}
if hasFormat {
    buf.Write("\tif action == 'c' || action == 'u' {\n")
    for _, f := range info.Fields {
        if f.Format != "" {
            validatorName := "form.Validate" + capitalize(f.Format)
            buf.Write(Sprintf("\t\tif err := %s(m.%s); err != nil { return err }\n", validatorName, f.Name))
        }
    }
    buf.Write("\t}\n")
}

buf.Write("\treturn nil\n")
buf.Write("}\n\n")
```

---

## Stage 3: Actualizar tests

- Eliminar tests que verifican que `db.Create`/`db.Update` llaman `Validate()`
- Actualizar mocks que implementen `Validate()` → `Validate(action byte)`
- Agregar test: `Validate('d')` no ejecuta format validators

```bash
gotest
```

---

## Stage 4: Actualizar documentación

**File:** `docs/SKILL.md`

- Línea 121: `fmt.ValidateFielder(m)` → `fmt.ValidateFields(action, m)`
- Documentar que `Validate(action byte)` es generado por ormc
- Documentar que format validators solo se ejecutan en `'c'` y `'u'`
- Documentar que `db.Create`/`db.Update` **no** validan campos — la validación ocurre en la frontera (crudp/form)
- Actualizar ejemplo de código generado con la nueva firma

---

## Stage 5: `ModelName()` — única fuente de verdad para todas las capas

**Motivación:** Actualmente se generan dos métodos con semántica idéntica:
- `TableName() string` → interfaz `orm.Model`, consumido por DB/SQL
- `FormName() string` → generado por ormc, consumido por HTTP/form

Ambos retornan `snake_low(StructName)`. Es duplicación. Si aparece un nuevo
transporte (gRPC, WebSocket, cola de mensajes), se añadiría un tercer método.
`ModelName() string` es la única fuente de verdad: cada capa la consume y adapta
si necesita (e.g. SQL puede pluralizar, HTTP construye la ruta, etc.).

### 5.1 `model.go` — eliminar interfaz `Model` (ahora en `fmt`)

```go
// ELIMINAR de orm:
type Model interface {
    fmt.Fielder
    TableName() string
}

// Los consumidores externos ahora usan fmt.Model
```

### 5.2 `ormc.go` — renombrar campos internos

```go
// ANTES (StructInfo):
type StructInfo struct {
    Name              string
    TableName         string
    TableNameDeclared bool
    // ...
}

// DESPUÉS:
type StructInfo struct {
    Name               string
    ModelName          string
    ModelNameDeclared  bool
    // ...
}
```

### 5.3 `ormc.go` — detectModelName() en lugar de detectTableName()

```go
// ANTES:
func detectTableName(node *ast.File, structName string) string {
    // busca func (X) TableName() string

// DESPUÉS:
func detectModelName(node *ast.File, structName string) string {
    // busca func (X) ModelName() string
```

### 5.4 `ormc.go` — código generado

```go
// ANTES (dos métodos generados):
func (m *User) TableName() string { return "user" }
func (m *User) FormName() string  { return "user" }

// DESPUÉS (un único método):
func (m *User) ModelName() string { return "user" }
```

El descriptor de metadatos también se actualiza:

```go
// ANTES:
var User_ = struct {
    TableName string
    // campos...
}{
    TableName: "user",
    // ...
}

// DESPUÉS:
var User_ = struct {
    ModelName string
    // campos...
}{
    ModelName: "user",
    // ...
}
```

### 5.5 `db.go` / `qb.go` / `validate.go` — actualizar consumidores

Reemplazar todas las llamadas a `m.TableName()` por `m.ModelName()`.

```
m.TableName()  →  m.ModelName()
```

### 5.6 Overrides de usuario

Los structs que antes declaraban `TableName()` manualmente deben renombrarlo:

```go
// ANTES:
func (u *User) TableName() string { return "users" }

// DESPUÉS:
func (u *User) ModelName() string { return "users" }
```

`detectModelName()` lo detecta vía AST y activa `ModelNameDeclared = true`,
evitando que ormc genere el método (mismo comportamiento que antes).

### 5.7 Tests

- Actualizar todos los mocks que implementen `TableName()` → `ModelName()`
- Verificar que `validateQuery` sigue resolviendo el nombre correcto
- Agregar test: struct con override `ModelName()` no recibe método generado

```bash
gotest
```


