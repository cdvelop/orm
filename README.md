# tinywasm/orm
<img src="docs/img/badges.svg">

**Ultra-lightweight, strongly-typed ORM engineered for WebAssembly and backend environments.**

## Features

- **Zero Reflection**: Interface-driven schema via `github.com/tinywasm/fmt`
- **Isomorphic**: Same generated code works in Go (backend) and WASM (frontend)
- **Three Layers**: DB persistence, JSON transport, and Form UI — from one struct definition
- **Code Generator**: `ormc` CLI generates boilerplate from struct tags

## Installation

```bash
go get github.com/tinywasm/orm
go install github.com/tinywasm/orm/cmd/ormc@latest
```

## Three Layers, One Struct

`ormc` generates code for three layers depending on the directive and tags present:

| Layer | Concern | Controlled by | Generated |
|-------|---------|---------------|-----------|
| **DB** | Persistence (tables, queries, CRUD) | `db:` tags | `ModelName()`, `ReadOne*`, `ReadAll*`, `T_`, `FieldDB` |
| **JSON** | Transport (serialization) | `json:` tags | `OmitEmpty` in schema |
| **Form** | UI (input widgets, validation) | `input:` tags + directive | `Widget` in schema, `Validate()` |

### Directives

| Directive | DB layer | Form layer | Use case |
|-----------|----------|------------|----------|
| *(none)* | Yes | No | DB-only struct (config, logs, metrics) |
| `// ormc:form` | Yes | Yes | Business entity with UI (user, product) |
| `// ormc:formonly` | No | Yes | Transport/UI struct without DB (login request, RPC params) |

## Quick Start

### 1. Define your structs

```go
package user

// ormc:form — DB + Form: full CRUD with UI rendering
type User struct {
    ID      string
    Name    string
    Email   string `db:"unique" input:"email"`
    Bio     string `json:",omitempty" input:"textarea"`
    Address Address
}

// ormc:formonly — Form only: validation + UI, no DB
type Address struct {
    Street string
    City   string
    Zip    string `input:"number"`
}
```

`ormc` auto-detects: `ID` as PK, field names as column/JSON names, default `input.Text()` widget for string fields. Only add tags when overriding defaults.

### 2. Generate

```bash
ormc
```

Generates `model_orm.go` next to each source file.

> [!TIP]
> `ormc` automatically runs `go get` for required dependencies and `go mod tidy` if a `go.mod` is detected.

### 3. Use it

```go
func GetActiveUsers(db *orm.DB) ([]*user.User, error) {
    return user.ReadAllUser(
        db.Query(&user.User{}).
            Where(user.User_.Email).Like("%@gmail.com").
            Limit(10),
    )
}
```

## Tags Reference

### `db:` — DB layer

| Tag | Effect |
|-----|--------|
| `db:"pk"` | Marks field as primary key (auto-detected for `ID` fields) |
| `db:"unique"` | Unique constraint |
| `db:"not_null"` | NOT NULL constraint |
| `db:"autoincrement"` | Auto-increment (numeric fields only) |
| `db:"ref=table"` | Foreign key to table (default column: `id`) |
| `db:"ref=table:col"` | Foreign key to specific column |
| `db:"-"` | Exclude field from schema entirely |

DB flags are grouped in `Field.DB *FieldDB` (nil for `formonly` structs). Helpers: `field.IsPK()`, `field.IsUnique()`, `field.IsAutoInc()`.

> **String PKs:** must be set by caller via `github.com/tinywasm/unixid` before `db.Create()`. The ORM does not generate IDs.

### `json:` — JSON layer

| Tag | Effect |
|-----|--------|
| `json:",omitempty"` | Sets `OmitEmpty: true` in schema |
| `json:"-"` | Exclude from JSON (field still in schema for DB/Form) |

### `input:` — Form layer

| Tag | Effect |
|-----|--------|
| `input:"email"` | `Widget: input.Email()` |
| `input:"textarea"` | `Widget: input.Textarea()` |
| `input:"password"` | `Widget: input.Password()` |
| `input:"number"` | `Widget: input.Number()` |
| `input:"required"` | `NotNull: true` |
| `input:"min=2,max=100"` | `Permitted: fmt.Permitted{Minimum: 2, Maximum: 100}` |
| `input:"letters,spaces"` | `Permitted: fmt.Permitted{Letters: true, Spaces: true}` |
| `input:"-"` | No widget (field skipped in form rendering) |

Available widget types: `text`, `email`, `password`, `textarea`, `phone`, `number`, `date`, `hour`, `ip`, `rut`, `address`, `checkbox`, `datalist`, `select`, `radio`, `filepath`, `gender`.

`ormc` generates `Validate(action byte)` calling `fmt.ValidateFields(action, m)`. Validation runs for `'c'` (create), `'u'` (update), and `'d'` (delete, PK only).

## Schema Types

| Go Type | FieldType |
|---|---|
| `string` | `fmt.FieldText` |
| `int`, `int32`, `int64`, `uint`, `uint32`, `uint64` | `fmt.FieldInt` |
| `float32`, `float64` | `fmt.FieldFloat` |
| `bool` | `fmt.FieldBool` |
| `[]byte` | `fmt.FieldBlob` |
| struct (nested) | `fmt.FieldStruct` |
| `time.Time` | not allowed — use `int64` + `tinywasm/time` |

## API Reference

### DB Operations

```go
db := orm.New(executor, compiler)

db.Create(&user)
db.Update(&user, orm.Eq(User_.ID, user.ID))
db.Delete(&user, orm.Eq(User_.ID, user.ID))
db.CreateTable(&User{})
db.DropTable(&User{})
```

`Update` and `Delete` require at least one condition (compile-time enforced):

```go
db.Update(&res, orm.Eq(Reservation_.ID, res.ID))  // one condition
db.Update(&cfg, orm.Eq(Config_.TenantID, tid),     // multiple conditions
                orm.Eq(Config_.Key, key))
db.Update(&res)                                     // compile error
```

### Query Builder

```go
user.ReadAllUser(
    db.Query(&user.User{}).
        Where(user.User_.Email).Like("%@example.com").
        OrderBy(user.User_.Name).Asc().
        Limit(10).Offset(20),
)
```

Chainable: `Where(col)` → `.Eq()`, `.Neq()`, `.Gt()`, `.Gte()`, `.Lt()`, `.Lte()`, `.Like()`, `.In()` | `OrderBy(col)` → `.Asc()`, `.Desc()` | `Limit(n)`, `Offset(n)`, `GroupBy(cols...)`

### Interfaces

| Interface | Methods |
|-----------|---------|
| `Compiler` | `Compile(Query, Model) (Plan, error)` |
| `Executor` | `Exec()`, `QueryRow()`, `Query()`, `Close()` |
| `TxExecutor` | `Executor` + `BeginTx()` |
| `TxBoundExecutor` | `Executor` + `Commit()`, `Rollback()` |

## ormc — Code Generation

Run from the **project root**. Scans subdirectories for `model.go` / `models.go`:

```
project/
  modules/
    user/model.go      → modules/user/model_orm.go
    product/models.go  → modules/product/model_orm.go
```

```go
//go:generate ormc
```

**Generated per struct:**

| What | When |
|------|------|
| `Schema() []Field`, `Pointers() []any` | Always |
| `Validate(action byte) error` | When struct has validation rules or is a form |
| `ModelName() string` | DB structs only (not `formonly`) |
| `T_` metadata struct | DB structs only |
| `ReadOneT()`, `ReadAllT()` | DB structs only |

**Programmatic API:**

| Method | Description |
|--------|-------------|
| `NewOrmc() *Ormc` | Create handler; `rootDir` defaults to `"."` |
| `SetLog(func(...any))` | Set warning/info log function |
| `SetRootDir(dir string)` | Set scan root |
| `Run() error` | Scan and generate |
| `GenerateForStruct(name, file string) error` | Generate for a single struct |
| `ParseStruct(name, file string) (StructInfo, error)` | Parse struct metadata only |
| `GenerateForFile(infos []StructInfo, file string) error` | Write all infos to one `_orm.go` |

## More Documentation

- [Architecture](docs/ARQUITECTURE.md)
