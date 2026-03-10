# ORM Skill

## Installation

```bash
go get github.com/tinywasm/orm
go install github.com/tinywasm/orm/cmd/ormc@latest
```

## Public API Contract

### Interfaces
- `Model`: `fmt.Fielder` + `TableName()` *(auto-implemented by `ormc`)*
- `Compiler`: `Compile(Query, Model) (Plan, error)`
- `Executor`: `Exec()`, `QueryRow()`, `Query()`, `Close()`
- `TxExecutor`: `BeginTx()`
- `TxBoundExecutor`: Embeds `Executor`, `Commit()`, `Rollback()`

### Model Interface

`Model` embeds `fmt.Fielder` and adds a table name:

```go
import "github.com/tinywasm/fmt"

type Model interface {
    fmt.Fielder           // Schema() []fmt.Field + Values() []any + Pointers() []any
    TableName() string
}
```

`fmt.Fielder` provides:

```go
type Fielder interface {
    Schema() []fmt.Field
    Values() []any
    Pointers() []any
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
| `time.Time` | ⚠️ **not allowed** — `ormc` warns and skips the field; use `int64` + `tinywasm/time`. Add `db:"-"` to suppress the warning |

### Schema Constraints (`fmt.Field` bool fields, no bitmask)
| Field | db tag | Notes |
|---|---|---|
| `PK bool` | `db:"pk"` | Auto-detected via `tinywasm/fmt.IDorPrimaryKey` |
| `Unique bool` | `db:"unique"` | |
| `NotNull bool` | `db:"not_null"` | |
| `AutoInc bool` | `db:"autoincrement"` | Numeric fields only |
| `Input string` | `form:"email"` | Hint for form rendering; `form:"-"` = skip |
| `JSON string` | `json:"name"` | Hint for JSON codec; `json:"-"` = skip |
| FK reference | `db:"ref=table"` or `db:"ref=table:column"` | stored in `FieldExt.Ref` + `FieldExt.RefColumn` |
| Ignore field | `db:"-"` | Silently excluded from `Schema()`, `Values()`, `Pointers()` |

> **String PKs:** must be set by caller via `github.com/tinywasm/unixid` before calling `db.Create()`. The ORM does not generate IDs.

### DB-only FK Metadata: `FieldExt`

```go
type FieldExt struct {
    fmt.Field
    Ref       string // FK: target table name. Empty = no FK.
    RefColumn string // FK: target column. Empty = auto-detect PK of Ref table.
}
```

Used internally by adapters (e.g., SQLite) for `CREATE TABLE` FK constraints.

### Auto-Generated Code (`cmd/ormc`)

Run `ormc` from the **project root**. It recursively scans all subdirectories for
`model.go` or `models.go`, and generates a single `model_orm.go` next to each source file.

```bash
ormc
```

Typical project structure:
```
project/
  modules/
    user/model.go      → generates modules/user/model_orm.go  (all structs)
    product/models.go  → generates modules/product/model_orm.go (all structs)
```

Use a single `//go:generate` at the project root — **not** per struct:
```go
//go:generate ormc
```

### For every struct, `ormc` generates

- `func (m *T) TableName() string` *(only if NOT already declared in source)*
- `func (m *T) FormName() string` — returns lowercase snake_case of the struct name
- `func (m *T) Schema() []fmt.Field`
- `func (m *T) Values() []any`
- `func (m *T) Pointers() []any`
- `T_` metadata struct with typed column name constants
- `ReadOneT(qb *orm.QB, model *T) (*T, error)`
- `ReadAllT(qb *orm.QB) ([]*T, error)`

### `form:` and `json:` tags

`ormc` reads struct tags and propagates them to `fmt.Field`:

| Tag | Field | Generated |
|---|---|---|
| `form:"email"` | `Input = "email"` | `Input: "email"` |
| `form:"-"` | `Input = "-"` | `Input: "-"` (form skips it) |
| `json:"name"` | `JSON = "name"` | `JSON: "name"` |
| `json:"bio,omitempty"` | `JSON = "bio,omitempty"` | `JSON: "bio,omitempty"` |
| `json:"-"` | `JSON = "-"` | `JSON: "-"` (json skips it) |

No `form` or `json` dependencies are imported or generated. The downstream `form` and `json` packages read `Field.Input` / `Field.JSON` autonomously.

### `// ormc:formonly` directive

Structs annotated with `// ormc:formonly` implement `fmt.Fielder` but NOT `orm.Model`:

```go
// ormc:formonly
type LoginRequest struct {
    Email    string
    Password string `form:"password"`
}
```

Generated methods: `FormName()`, `Schema()`, `Values()`, `Pointers()`.  
**Not generated:** `TableName()`, `ReadOne*`, `ReadAll*`, `T_` descriptor.

### Programmatic usage (`ormc` embedded in another tool)

```go
o := orm.NewOrmc()
o.SetLog(func(messages ...any) {   // optional
    myLogger.Warn(messages...)
})
o.SetRootDir("/path/to/project")   // optional: defaults to "."
if err := o.Run(); err != nil {
    return err
}
```

**`Ormc` methods:**

| Method | Description |
|--------|-------------|
| `NewOrmc() *Ormc` | Create handler; `rootDir` defaults to `"."` |
| `(o) SetLog(func(...any))` | Set warning/info log function |
| `(o) SetRootDir(dir string)` | Set scan root |
| `(o) Run() error` | Scan `rootDir` for `model.go`/`models.go`, generate `_orm.go` |
| `(o) GenerateForStruct(name, file string) error` | Generate for a single named struct |
| `(o) ParseStruct(name, file string) (StructInfo, error)` | Parse struct metadata only |
| `(o) GenerateForFile(infos []StructInfo, file string) error` | Write all infos to one `_orm.go` |

### Core Structs
- `DB`: `New(Executor, Compiler)`, `Create`, `Update(m, cond, rest...)`,
        `Delete(m, cond, rest...)`, `Query`, `Tx`, `Close`, `RawExecutor`,
        `CreateTable`, `DropTable`, `CreateDatabase`
- `QB` (Fluent API): `Where("col")`, `Limit(n)`, `Offset(n)`, `OrderBy("col")`, `GroupBy("cols...")`
- `Clause` (Chainable): `.Eq()`, `.Neq()`, `.Gt()`, `.Gte()`, `.Lt()`, `.Lte()`, `.Like()`, `.In()`
- `OrderClause` (Chainable): `.Asc()`, `.Desc()`
- `Plan`: `Mode`, `Query`, `Args`

### Constants
- `Action`: `ActionCreate`, `ActionReadOne`, `ActionUpdate`, `ActionDelete`, `ActionReadAll`, `ActionCreateTable`, `ActionDropTable`, `ActionCreateDatabase`

## API Safety Contract

### `Update` and `Delete` require at least one Condition

```go
// ✅ Correct — explicit WHERE clause, single row targeted
db.Update(&res, orm.Eq(Reservation_.ID, res.ID))

// ✅ Correct — multiple conditions still work
db.Update(&cfg, orm.Eq(Config_.TenantID, tid), orm.Eq(Config_.StaffID, sid))

// ❌ Compile error — zero conditions is forbidden
db.Update(&res)
```

This is enforced at compile time by Go's type system (non-variadic first argument).
See: `docs/ARQUITECTURE.md` section 3.6

---

## Usage Snippet

```go
// 1. Where clauses use generated _ descriptors (no magic strings)
// 2. Query builder uses a Fluent API chain
// 3. Results are executed and cast by auto-generated typed functions
qb := db.Query(m).
    Where(User_.Age).Eq(18).
    Or().Where(User_.Name).Like("A%").
    OrderBy(User_.CreatedAt).Desc().
    Limit(10)

user, err := models.ReadAllUser(qb)

// Schema DDL (adapter handles IF NOT EXISTS / IF EXISTS internally)
if err := db.CreateTable(&User{}); err != nil { ... }
if err := db.DropTable(&User{}); err != nil { ... }
if err := db.CreateDatabase("myapp"); err != nil { ... }
```
