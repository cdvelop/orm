# ORM Skill

## Installation

```bash
go get github.com/tinywasm/orm
go install github.com/tinywasm/orm/cmd/ormc@latest
```

## Public API Contract

### Interfaces
- `Model`: `TableName()`, `Schema()`, `Values()`, `Pointers()` *(auto-implemented by `ormc`)*
- `Compiler`: `Compile(Query, Model) (Plan, error)`
- `Executor`: `Exec()`, `QueryRow()`, `Query()`, `Close()`
- `TxExecutor`: `BeginTx()`
- `TxBoundExecutor`: Embeds `Executor`, `Commit()`, `Rollback()`

### Model Interface
`Columns() []string` has been **replaced** by `Schema() []orm.Field`:
```go
type Model interface {
    TableName() string
    Schema() []orm.Field   // Key: column name, includes type + constraints
    Values() []any
    Pointers() []any
}
```

### Schema Field Types (`orm.FieldType`)
| Go Type | FieldType |
|---|---|
| `string` | `TypeText` |
| `int`, `int32`, `int64` | `TypeInt64` |
| `float32`, `float64` | `TypeFloat64` |
| `bool` | `TypeBool` |
| `[]byte` | `TypeBlob` |
| `time.Time` | ⚠️ **not allowed** — `ormc` warns and skips the field; use `int64` + `tinywasm/time`. Add `db:"-"` to suppress the warning |

### Schema Constraints (`orm.Constraint`, bitmask)
| Constant | db tag | Notes |
|---|---|---|
| `ConstraintPK` | `db:"pk"` | Auto-detected via `tinywasm/fmt.IDorPrimaryKey` |
| `ConstraintUnique` | `db:"unique"` | |
| `ConstraintNotNull` | `db:"not_null"` | |
| `ConstraintAutoIncrement` | `db:"autoincrement"` | Numeric fields only |
| FK reference | `db:"ref=table"` or `db:"ref=table:column"` | `Field.Ref` + `Field.RefColumn` |
| Ignore field | `db:"-"` | Silently excluded from `Schema()`, `Values()`, `Pointers()` |

> **String PKs:** must be set by caller via `github.com/tinywasm/unixid` before calling `db.Create()`. The ORM does not generate IDs.

### Auto-Generated Code (`cmd/ormc`)

Run `ormc` from the **project root**. It recursively scans all subdirectories looking
for `model.go` or `models.go`, and generates a single `model_db.go` next to each
source file found (always overwritten). All structs in the same file are generated
into one output file.

```bash
ormc
```

Typical project structure:
```
project/
  modules/
    user/model.go      → generates modules/user/model_db.go  (all structs)
    product/models.go  → generates modules/product/model_db.go (all structs)
```

Use a single `//go:generate` at the project root — **not** per struct:
```go
//go:generate ormc
```

### Programmatic usage (`ormc` embedded in another tool)

```go
o := orm.NewOrmc()
o.SetLog(func(messages ...any) {   // optional: silently discarded if not set
    myLogger.Warn(messages...)
})
o.SetRootDir("/path/to/project")   // optional: defaults to "."
if err := o.Run(); err != nil {    // Run() uses o.rootDir, no parameter
    return err
}
```

**`Ormc` methods** (all return `error`, never call `os.Exit`):

| Method | Description |
|--------|-------------|
| `NewOrmc() *Ormc` | Create handler; `rootDir` defaults to `"."` |
| `(o) SetLog(func(...any))` | Set warning/info log function |
| `(o) SetRootDir(dir string)` | Set scan root (useful for tests: no `os.Chdir` needed) |
| `(o) Run() error` | Scan `rootDir` for `model.go`/`models.go`, generate `_db.go` |
| `(o) GenerateForStruct(name, file string) error` | Generate for a single named struct |
| `(o) ParseStruct(name, file string) (StructInfo, error)` | Parse struct metadata only |
| `(o) GenerateForFile(infos []StructInfo, file string) error` | Write all infos to one `_db.go` |

For a `struct User`, `ormc` generates in `model_db.go`:
- `func (m *User) Schema() []orm.Field`
- `func (m *User) Values() []any`
- `func (m *User) Pointers() []any`
- `func (m *User) TableName() string` *(only if NOT already declared in source)*
- `User_` struct with typed column name constants
- `ReadOneUser(qb *orm.QB, model *User) (*User, error)`
- `ReadAllUser(qb *orm.QB) ([]*User, error)`

### `db:"-"` — field exclusion

Fields tagged `db:"-"` are **silently** excluded from `Schema()`, `Values()`, and `Pointers()`.

Slice-of-struct fields (e.g. `[]Role`) are **automatically mapped** to one-to-many relations if the child struct has a matching `db:"ref=..."` field pointing to the parent table.

```go
type User struct {
    ID    string
    Name  string
    Roles []Role // auto-mapped via Role.UserID db:"ref=user"
}
```

This generates `ReadAllRoleByUserID(db, id)` in the child's `_db.go`.


### `TableName()` auto-detection

If the source file already declares `func (X) TableName() string`, `ormc` **will not generate
a duplicate**. If absent, `ormc` generates it as the snake_case of the struct name.

### Core Structs
- `DB`: `New(Executor, Compiler)`, `Create`, `Update`, `Delete`, `Query`, `Tx`, `Close`, `RawExecutor`, `CreateTable`, `DropTable`, `CreateDatabase`
- `QB` (Fluent API): `Where("col")`, `Limit(n)`, `Offset(n)`, `OrderBy("col")`, `GroupBy("cols...")`
- `Clause` (Chainable): `.Eq()`, `.Neq()`, `.Gt()`, `.Gte()`, `.Lt()`, `.Lte()`, `.Like()`, `.In()`
- `OrderClause` (Chainable): `.Asc()`, `.Desc()`
- `Plan`: `Mode`, `Query`, `Args`

### Constants
- `Action`: `ActionCreate`, `ActionReadOne`, `ActionUpdate`, `ActionDelete`, `ActionReadAll`, `ActionCreateTable`, `ActionDropTable`, `ActionCreateDatabase`

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
