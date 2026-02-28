# ORM Skill

## Public API Contract

### Interfaces
- `Model`: `TableName()`, `Columns()`, `Values()`, `Pointers()`
- `Compiler`: `Compile(Query, Model) (Plan, error)`
- `Executor`: `Exec()`, `QueryRow()`, `Query()`
- `TxExecutor`: `BeginTx()`
- `TxBoundExecutor`: Embeds `Executor`, `Commit()`, `Rollback()`

### Structs
- `DB`: `New(Executor, Compiler)`, `Create`, `Update`, `Delete`, `Query`, `Tx`
- `QB`: `Where`, `Limit`, `Offset`, `OrderBy`, `GroupBy`, `ReadOne`, `ReadAll`
- `Condition`: Helpers `Eq`, `Neq`, `Gt`, `Gte`, `Lt`, `Lte`, `Like`, `Or`
- `Order`: `Column()`, `Dir()`
- `Plan`: `Mode`, `Query`, `Args`

### Constants
- `Action`: `Create`, `ReadOne`, `Update`, `Delete`, `ReadAll`

## Usage Snippet

```go
db.Query(m).
    Where(orm.Eq("age", 18), orm.Like("name", "A%")).
    OrderBy("created_at", "DESC").
    Limit(10).
    ReadAll(newFunc, onRowFunc)
```
