# tinywasm/orm
<img src="docs/img/badges.svg">

**Ultra-lightweight, strongly-typed ORM engineered for WebAssembly and backend environments.**

## Features

- **Standard Library Only**: No external assertion libraries or heavy dependencies.
- **Isomorphic & Agnostic**: Works identically in Go (backend) and WASM (frontend). Generated code contains no build tags.
- **Interface over Reflection**: Zero use of `reflect` at runtime for maximum performance.
- **Typed Schema**: Uses `github.com/tinywasm/fmt` for deterministic field mapping.
- **Boilerplate Generator**: `ormc` CLI automates the `Model` interface implementation and handles dependencies.

## Installation

```bash
go get github.com/tinywasm/orm
go install github.com/tinywasm/orm/cmd/ormc@latest
```

## Quick Start

### 1. Define your Model

```go
// modules/user/model.go
package user

type User struct {
    ID    string `db:"pk"           json:"id"`
    Name  string                    `json:"name"`
    Email string `db:"unique"       json:"email"   form:"email"`
    Bio   string `form:"textarea"   json:"bio,omitempty"`
}
```

### 2. Generate the Boilerplate

Run `ormc` from your project root:

```bash
ormc
```

This generates `modules/user/model_orm.go` with the `Model` implementation and typed helpers.

> [!TIP]
> `ormc` automatically runs `go get` for required dependencies and `go mod tidy` if a `go.mod` is detected.

### 3. Use it

```go
import (
    "github.com/tinywasm/orm"
    "yourproject/modules/user"
)

func GetActiveUsers(db *orm.DB) ([]*user.User, error) {
    return user.ReadAllUser(
        db.Query(&user.User{}).
            Where(user.User_.Email).Like("%@gmail.com").
            Limit(10),
    )
}
```

## API Reference

### Interfaces

- `Compiler`: `Compile(Query, Model) (Plan, error)`
- `Executor`: `Exec()`, `QueryRow()`, `Query()`, `Close()`
- `TxExecutor`: Embeds `Executor`, adds `BeginTx()`
- `TxBoundExecutor`: Embeds `Executor`, adds `Commit()`, `Rollback()`

### Model Interface

`Model` lives in `github.com/tinywasm/fmt` and is auto-implemented by `ormc`:

```go
type Model interface {
    fmt.Fielder           // Schema() []fmt.Field + Pointers() []any
    ModelName() string
}
```

### Schema Field Types (`fmt.FieldType`)

| Go Type | FieldType |
|---|---|
| `string` | `fmt.FieldText` |
| `int`, `int32`, `int64`, `uint`, `uint32`, `uint64` | `fmt.FieldInt` |
| `float32`, `float64` | `fmt.FieldFloat` |
| `bool` | `fmt.FieldBool` |
| `[]byte` | `fmt.FieldBlob` |
| struct (embedded) | `fmt.FieldStruct` |
| `time.Time` | ⚠️ **not allowed** — use `int64` + `tinywasm/time`. Add `db:"-"` to suppress the warning |

### Schema Constraints (`fmt.Field`)

| Field | db tag | Notes |
|---|---|---|
| `PK bool` | `db:"pk"` | Auto-detected via `tinywasm/fmt.IDorPrimaryKey` |
| `Unique bool` | `db:"unique"` | |
| `NotNull bool` | `db:"not_null"` | |
| `AutoInc bool` | `db:"autoincrement"` | Numeric fields only |
| `OmitEmpty bool` | `json:",omitempty"` | Propagated from `json` tag |
| `Permitted fmt.Permitted` | `validate:"..."` | Validation rules for characters and bounds |
| FK reference | `db:"ref=table"` or `db:"ref=table:column"` | Stored in `FieldExt.Ref` + `FieldExt.RefColumn` |
| Ignore field | `db:"-"` | Silently excluded from `Schema()`, `Pointers()` |

> **String PKs:** must be set by caller via `github.com/tinywasm/unixid` before calling `db.Create()`. The ORM does not generate IDs.

`FieldExt` carries FK metadata used by adapters (e.g., SQLite) for `CREATE TABLE` constraints:

```go
type FieldExt struct {
    fmt.Field
    Ref       string // FK: target table name. Empty = no FK.
    RefColumn string // FK: target column. Empty = auto-detect PK of Ref table.
}
```

### DB / QB / Clause

- `DB`: `New(Executor, Compiler)`, `Create`, `Update(m, cond, rest...)`, `Delete(m, cond, rest...)`, `Query`, `Tx`, `Close`, `RawExecutor`, `CreateTable`, `DropTable`, `CreateDatabase`
- `QB` (Fluent API): `Where("col")`, `Limit(n)`, `Offset(n)`, `OrderBy("col")`, `GroupBy("cols...")`
- `Clause` (Chainable): `.Eq()`, `.Neq()`, `.Gt()`, `.Gte()`, `.Lt()`, `.Lte()`, `.Like()`, `.In()`
- `OrderClause` (Chainable): `.Asc()`, `.Desc()`
- `Plan`: `Mode`, `Query`, `Args`

### Constants

`ActionCreate`, `ActionReadOne`, `ActionUpdate`, `ActionDelete`, `ActionReadAll`, `ActionCreateTable`, `ActionDropTable`, `ActionCreateDatabase`

### `Update` and `Delete` require at least one Condition

Enforced at compile time — the first condition is non-variadic:

```go
db.Update(&res, orm.Eq(Reservation_.ID, res.ID))           // ✅
db.Update(&cfg, orm.Eq(Config_.TenantID, tid), orm.Eq(...)) // ✅
db.Update(&res)                                             // ❌ compile error
```

See `docs/ARQUITECTURE.md` section 3.6.

### `validate:` and `json:` tags

| Tag | Generated |
|---|---|
| `validate:"required"` | `NotNull: true` |
| `validate:"email"` | Injects `form.ValidateEmail` in `Validate()` |
| `validate:"min=2"` | `Permitted: {Minimum: 2}` |
| `json:"bio,omitempty"` | `OmitEmpty: true` |

`ormc` generates a `Validate(action byte)` method calling `fmt.ValidateFields(action, m)` plus format validators. Validators only run for `'c'` (create) or `'u'` (update).

## ormc — Code Generation

Run from the **project root**. Scans subdirectories for `model.go` / `models.go` and generates `model_orm.go` next to each:

```
project/
  modules/
    user/model.go      → modules/user/model_orm.go
    product/models.go  → modules/product/model_orm.go
```

Use a single `//go:generate` at the project root:

```go
//go:generate ormc
```

**Generated per struct:**

- `ModelName() string` *(only if not already declared)*
- `Schema() []fmt.Field`, `Pointers() []any`, `Validate(action byte) error`
- `T_` metadata struct with typed column name constants
- `ReadOneT(qb *orm.QB, model *T) (*T, error)`
- `ReadAllT(qb *orm.QB) ([]*T, error)`

**`// ormc:formonly` directive** — implements `fmt.Fielder` only (no `ModelName`, `ReadOne*`, `ReadAll*`, `T_`):

```go
// ormc:formonly
type LoginRequest struct {
    Email    string
    Password string `form:"password"`
}
```

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
