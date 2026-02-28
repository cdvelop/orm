# ORM Architecture

The `tinywasm/orm` package is an ultra-lightweight, strongly-typed, zero-magic (no `reflect`), and isomorphic (WASM/Backend) ORM, strictly following the architectural mandates of the `tinywasm` ecosystem.

## 1. Primary Architectural Pattern: Dependency Injection and Explicit Models

Unlike traditional ORMs (GORM, ObjectDB) that use `reflect` to infer tables and columns at runtime, `tinywasm/orm` requires **Entities (Structs)** to implement an explicit interface (`orm.Model`). This ensures:

1. **O(1) Performance in WASM:** Reflective processing overhead is eliminated.
2. **Reduced Binary Size:** Importing `reflect` inflates the WASM binary.
3. **Strict Type Safety:** The compiler detects errors before runtime.

The ORM acts as an agnostic orchestrator. It is unaware of whether the destination is Postgres, SQLite, or IndexedDB. It delegates the query translation to a **Compiler** and execution to an **Executor** injected from `cmd/main.go`.

---

## 2. Flowchart (The Lifecycle of a Query)

[The Lifecycle of a Query: ORM_FLOW.md](diagrams/ORM_FLOW.md)

`Model` → `Query` → `Compiler` → `Plan` → `Executor`

---

## 3. Fundamental Components

### 3.1. `Model` Interface (User Entity)

The application developer must make their structs implement this interface. The ORM extracts everything it needs to build a `Query` from here.

```go
type Model interface {
    // TableName returns the physical name of the table or store.
    TableName() string
    // Columns returns the ordered list of column names.
    Columns() []string
    // Values returns the entity values in the same order as Columns(). Used in Create/Update.
    Values() []any
    // Pointers returns pointers to the entity fields to inject data during a Read.
    Pointers() []any
}
```

> `Values()` and `Pointers()` are only called by the Executor logic for the operations that require them.

---

### 3.2. Agnostic Structures

#### `Action` (Typed Constant)

```go
type Action int

const (
    ActionCreate  Action = iota
    ActionReadOne
    ActionUpdate
    ActionDelete
    ActionReadAll
)
```

Using a typed constant (not a raw string) ensures the compiler catches any invalid action at compile time.

#### `Condition` (Filter)

A sealed value type. Consumers **must** construct via helpers (`Eq`, `Gt`, `Or`, etc.) — direct struct literal construction from outside the package is a compile error due to unexported fields. Compilers read values via getter methods.

```go
type Condition struct {
    field    string
    operator string  // "=", "!=", ">", ">=", "<", "<=", "LIKE"
    value    any
    logic    string  // "AND" (default) | "OR" — applies between this and the NEXT condition
}

func (c Condition) Field() string    { return c.field }
func (c Condition) Operator() string { return c.operator }
func (c Condition) Value() any       { return c.value }
func (c Condition) Logic() string    { return c.logic }
```

#### `Order` (Sorting)

A sealed value type. Constructed **only** internally by `QB.OrderBy()` — never by consumers or compilers directly. Compilers read values via getter methods.

```go
type Order struct {
    column string
    dir    string  // "ASC" | "DESC"
}

func (o Order) Column() string { return o.column }
func (o Order) Dir() string    { return o.dir }
```

#### `Query` (Agnostic Request)

The central struct passed from the ORM core to the Compiler. Contains everything the Compiler needs to translate into a native operation.

```go
type Query struct {
    Action     Action
    Table      string
    Columns    []string
    Values     []any
    Conditions []Condition
    OrderBy    []Order
    GroupBy    []string
    Limit      int
    Offset     int
}
```

---

### 3.3. `Compiler` Interface

Responsible for translating ORM queries into executable plans for a specific engine.

Examples:
- SQL Compiler
- IndexedDB Compiler
- KV Compiler

The ORM core remains engine-agnostic.

```go
type Compiler interface {
    Compile(q Query, m Model) (Plan, error)
}
```

#### `Plan` (Execution Instructions)

Ensure `Plan` is minimal and efficient.

```go
type Plan struct {
    Mode  Action
    Query string
    Args  []any
}
```

---

### 3.4. `Executor` Interface (Execution)

Executor must remain `database/sql` compatible, mock friendly, and engine independent. It does not import `database/sql`.

```go
type Executor interface {
    Exec(query string, args ...any) error
    QueryRow(query string, args ...any) Scanner
    Query(query string, args ...any) (Rows, error)
}
```

---

### 3.5. Transaction Interfaces (Optional Extension)

Not all database engines support transactions. Transaction support is **optional**: executors implement these interfaces only if their engine supports atomic operations.

```go
// TxBoundExecutor represents an executor bound to a transaction.
type TxBoundExecutor interface {
    Executor
    Commit() error
    Rollback() error
}

// TxExecutor represents an executor that supports transactions.
type TxExecutor interface {
    Executor
    BeginTx() (TxBoundExecutor, error)
}
```

---

### 3.6. The Core of the ORM (Public API)

The `DB` struct is instantiated from `cmd/main.go` and injected into handler/logic layers.

#### Write Operations (Direct — no builder)

```go
type DB struct { exec Executor, compiler Compiler }

func New(exec Executor, compiler Compiler) *DB

func (db *DB) Create(m Model) error
func (db *DB) Update(m Model, conds ...Condition) error
func (db *DB) Delete(m Model, conds ...Condition) error

// Tx executes fn inside an atomic transaction.
func (db *DB) Tx(fn func(tx *DB) error) error
```

#### Read Operations (Builder/Chain)

```go
// Query returns a QueryBuilder scoped to the given model.
func (db *DB) Query(m Model) *QB

type QB struct {
    db      *DB
    model   Model
    conds   []Condition
    orderBy []Order
    groupBy []string
    limit   int
    offset  int
}

func (q *QB) Where(conds ...Condition) *QB
func (q *QB) Limit(n int) *QB
func (q *QB) Offset(n int) *QB
func (q *QB) OrderBy(col, dir string) *QB
func (q *QB) GroupBy(cols ...string) *QB

// ReadOne executes the query and fills m (passed to db.Query) via Model.Pointers().
// Returns orm.ErrNotFound if no row matches.
func (q *QB) ReadOne() error

// ReadAll executes the query; for each row it calls new() to get a fresh Model,
// scans into its Pointers(), then calls onRow(m). The caller owns accumulation.
func (q *QB) ReadAll(new func() Model, onRow func(Model)) error
```

#### Condition Helpers

```go
func Eq(field string, val any) Condition   // field = val
func Neq(field string, val any) Condition  // field != val
func Gt(field string, val any) Condition   // field > val
func Gte(field string, val any) Condition  // field >= val
func Lt(field string, val any) Condition   // field < val
func Lte(field string, val any) Condition  // field <= val
func Like(field string, val any) Condition // field LIKE val
func Or(c Condition) Condition             // wraps c with Logic = "OR"
```

---

### 3.7. Sentinel Errors

```go
var (
    ErrNotFound     = errors.New("orm: record not found")
    ErrValidation   = errors.New("orm: model validation failed")
    ErrEmptyTable   = errors.New("orm: model returned empty table name")
    ErrNoTxSupport  = errors.New("orm: adapter does not support transactions")
)
```

Callers use `errors.Is(err, orm.ErrNotFound)` to branch on error type without string parsing.

---

## 4. Advantages of this Design

1. **Fully Stdlib & WASM Compatible:** `tinywasm/orm` core does not import `database/sql` nor interact with the OS or Network.
2. **Separation of Concerns:** The ORM packs/unpacks data (`Model` → `Query`). `Compiler` translates logic (`Query` → `Plan`). `Executor` runs operations on DB Engine.
3. **Zero-Alloc ReadMany:** The `ReadAll(new, onRow)` push-based pattern avoids internal slice management; the caller decides whether to accumulate, stream, or discard rows.
4. **Type-Safe Actions:** `Action int` constants prevent logic errors from typos at compile time.
5. **Composable Queries:** The `QB` builder allows incremental construction of complex queries in handler logic without string concatenation.
6. **Testable without Real Databases:** A `MockExecutor` and `MockCompiler` can validate all business logic without touching disks or ports.
7. **Optional Transactions:** The `TxExecutor`/`TxBoundExecutor` pattern allows engines to opt-in to transaction support without modifying the core `Executor` interface.
