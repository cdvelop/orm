# ORM Skill

## Installation

```bash
go get github.com/tinywasm/orm
go install github.com/tinywasm/orm/cmd/ormc@latest
```

## Public API Contract

### Interfaces
- `Model`: `TableName()`, `Columns()`, `Values()`, `Pointers()` *(auto-implemented by `ormc`)*
- `Compiler`: `Compile(Query, Model) (Plan, error)`
- `Executor`: `Exec()`, `QueryRow()`, `Query()`
- `TxExecutor`: `BeginTx()`
- `TxBoundExecutor`: Embeds `Executor`, `Commit()`, `Rollback()`

### Auto-Generated Code (`cmd/ormc`)
To auto-generate the `orm.Model` interface and typed definitions, place the `//go:generate` directive above an standard Go struct (no special tags needed):

```go
//go:generate ormc -struct User
type User struct {
    ID       string // Automatically detected as PK by tinywasm/fmt
    Username string
    Age      int
}
```

For a `struct User`, the `ormc` compiler generates:
- `UserMeta` struct containing table and typed column names (e.g. `UserMeta.Username`).
- `ReadOneUser(qb *orm.QB) (*User, error)`
- `ReadAllUser(qb *orm.QB) ([]*User, error)`
- `ReadAllUser(qb *orm.QB) ([]*User, error)`

### Core Structs
- `DB`: `New(Executor, Compiler)`, `Create`, `Update`, `Delete`, `Query`, `Tx`
- `QB` (Fluent API): `Where("col")`, `Limit(n)`, `Offset(n)`, `OrderBy("col")`, `GroupBy("cols...")`
- `Clause` (Chainable): `.Eq()`, `.Neq()`, `.Gt()`, `.Gte()`, `.Lt()`, `.Lte()`, `.Like()`
- `OrderClause` (Chainable): `.Asc()`, `.Desc()`
- `Plan`: `Mode`, `Query`, `Args`

### Constants
- `Action`: `Create`, `ReadOne`, `Update`, `Delete`, `ReadAll`

## Usage Snippet

```go
// 1. Where clauses use generated Meta descriptors (no magic strings)
// 2. Query builder uses a Fluent API chain
// 3. Results are executed and cast by auto-generated typed functions
qb := db.Query(m).
    Where(UserMeta.Age).Eq(18).
    Or().Where(UserMeta.Name).Like("A%").
    OrderBy(UserMeta.CreatedAt).Desc().
    Limit(10)

users, err := models.ReadAllUser(qb)
```
