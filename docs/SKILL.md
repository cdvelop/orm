# ORM Skill

## Public API Contract

### Interfaces
- `Model`: `TableName()`, `Columns()`, `Values()`, `Pointers()`
- `Adapter`: `Execute(Query, Model, factory, each)`
- `TxAdapter`: `BeginTx()`
- `TxBound`: Embeds `Adapter`, `Commit()`, `Rollback()`

### Structs
- `DB`: `New(Adapter)`, `Create`, `Update`, `Delete`, `Query`, `Tx`
- `QB`: `Where`, `Limit`, `Offset`, `OrderBy`, `GroupBy`, `ReadOne`, `ReadAll`
- `Condition`: Helpers `Eq`, `Neq`, `Gt`, `Gte`, `Lt`, `Lte`, `Like`, `Or`
- `Order`: `Column()`, `Dir()`

### Constants
- `Action`: `Create`, `ReadOne`, `Update`, `Delete`, `ReadAll`

## Usage Snippet

```go
db.Query(m).
    Where(orm.Eq("age", 18), orm.Like("name", "A%")).
    OrderBy("created_at", "DESC").
    Limit(10).
    ReadAll(factory, each)
```
