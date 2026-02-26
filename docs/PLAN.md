# ORM Implementation

This document is the **Master Prompt (PLAN.md)** for the initial creation of the `tinywasm/orm` ecosystem. Every execution agent must follow this plan sequentially.
The architecture is governed by the strict **Interface over Reflection** pattern to ensure the highest speed in WebAssembly.

---

## Development Rules

These rules are non-negotiable constraints extracted from `tinywasm/devflow/docs/DEFAULT_LLM_SKILL.md`. All phases must comply.

### Core Principles
- **SRP:** Every file has a single, well-defined purpose reflected in its name (`orm.go`, `conditions.go`, `errors.go`, `validate.go`).
- **DI:** No global state. The `Adapter` interface is the only external dependency. It is injected exclusively in `cmd/<app>/main.go` by the consumer — never inside `tinywasm/orm` itself.
- **Flat Hierarchy:** No subdirectories in the library root. All source files live at `orm/`.
- **Max 500 lines per file:** If any file exceeds 500 lines, split it by domain.
- **Test Organization:** All test files go in `tests/` (already >1 test file expected).

### WASM Compatibility
- **Forbidden imports in `orm/`:** `database/sql`, `reflect`, `fmt`, `strings`, `strconv`, `errors` (stdlib). Use `tinywasm/fmt` instead of `fmt`/`strings`/`strconv`/`errors`.
- **No `map` declarations:** Use structs or slices for all internal collections to prevent WASM binary bloat.
- **No OS/Network calls:** The core library must be pure logic — zero system interaction.

### Testing
- **Runner:** Always use `gotest` (never `go test` directly).
- **Publish:** Use `gopush 'message'` after tests pass and docs are updated.
- **No external assertion libs:** Use only standard `testing` package.
- **Mock all I/O:** `MockAdapter` must be the only way external behavior is simulated.
- **WASM/Stdlib dual pattern:** `tests/orm_stlib_test.go` (`//go:build !wasm`) and `tests/orm_wasm_test.go` (`//go:build wasm`) both call a shared `RunCoreTests(t)`.
- **DDT:** Every branch in [ORM_FLOW.md](diagrams/ORM_FLOW.md) must have a corresponding test case.

### Documentation
- **Docs before code:** Documentation must be finalized before writing source files.
- **Readme as index:** Every file in `docs/` must be linked from `README.md`.

---

## Contextual References
- **Base Architecture:** [ARQUITECTURE.md](ARQUITECTURE.md)
- **Lifecycle Flow:** [diagrams/ORM_FLOW.md](diagrams/ORM_FLOW.md)

---

## Public API Contract

This section defines the exact visibility of every type and field. Execution agents MUST NOT deviate from this table. The rule is: **expose the minimum needed for consumers and adapters to function**.

### Exported Types (visible outside `orm` package)

| Symbol | Kind | Reason |
|---|---|---|
| `Model` | interface | Consumers implement it |
| `Adapter` | interface | Consumers inject it |
| `DB` | struct | Consumers instantiate it via `New()` |
| `QB` | struct | Consumers hold a `*QB` reference in variables for incremental building |
| `Query` | struct | Adapters (external packages) read its fields to build native queries |
| `Action` | type (`int`) | Adapters switch on it |
| `ActionCreate/ReadOne/Update/Delete/ReadAll` | constants | Adapters switch on them |
| `TxAdapter` | interface | Optional — adapters implement to signal transaction support |
| `TxBound` | interface | Active transaction handle: embeds `Adapter` + `Commit`/`Rollback` |
| `Condition` | struct | Sealed value type; consumers create via helpers; adapters read via getter methods |
| `Condition.Field()` | method | Adapter getter — reads unexported `field` |
| `Condition.Operator()` | method | Adapter getter — reads unexported `operator` |
| `Condition.Value()` | method | Adapter getter — reads unexported `value` |
| `Condition.Logic()` | method | Adapter getter — reads unexported `logic` |
| `Order` | struct | Sealed value type; built internally by `QB.OrderBy()`; adapters read via getter methods |
| `Order.Column()` | method | Adapter getter — reads unexported `column` |
| `Order.Dir()` | method | Adapter getter — reads unexported `dir` |
| `ErrNotFound` | var | Consumers check with `errors.Is` |
| `ErrValidation` | var | Consumers check with `errors.Is` |
| `ErrEmptyTable` | var | Consumers check with `errors.Is` |
| `ErrNoTxSupport` | var | Returned by `DB.Tx()` when adapter doesn't implement `TxAdapter` |
| `New` | func | Public constructor |
| `Eq/Neq/Gt/Gte/Lt/Lte/Like/Or` | funcs | Public condition helpers |

### Unexported Fields (private within `orm` package)

| Type | Field | Reason |
|---|---|---|
| `DB` | `adapter Adapter` | Injected at construction, never accessed directly |
| `QB` | `db *DB` | Internal back-reference |
| `QB` | `model Model` | Set once at `db.Query(m)`, not reassignable |
| `QB` | `conds []Condition` | Mutated only via `.Where()` |
| `QB` | `orderBy []Order` | Mutated only via `.OrderBy()` |
| `QB` | `groupBy []string` | Mutated only via `.GroupBy()` |
| `QB` | `limit int` | Mutated only via `.Limit()` |
| `QB` | `offset int` | Mutated only via `.Offset()` |
| `Condition` | `field string` | Set only by constructor helpers (`Eq`, `Gt`, etc.) |
| `Condition` | `operator string` | Set only by constructor helpers |
| `Condition` | `value any` | Set only by constructor helpers |
| `Condition` | `logic string` | Set only by constructor helpers; `Or()` wraps to `"OR"` |
| `Order` | `column string` | Set only internally by `QB.OrderBy()` |
| `Order` | `dir string` | Set only internally by `QB.OrderBy()` |

### Unexported Functions

| Symbol | Reason |
|---|---|
| `validate(action Action, m Model) error` | Internal guard, not part of the public contract |

### Construction Rules
- `Query` structs are **only built internally** by the ORM methods (`Create`, `Update`, `Delete`, `ReadOne`, `ReadAll`). Consumers and Adapters only **read** a `Query`, never create one manually. All `Query` fields are exported solely for Adapter read-access.
- `QB` fields are **never accessed directly** by consumers. Consumers only call its chain methods and terminal methods (`.ReadOne()`, `.ReadAll()`).
- `Condition` structs are **only constructable via helpers** (`Eq`, `Neq`, `Gt`, `Gte`, `Lt`, `Lte`, `Like`, `Or`). Direct struct literal construction from outside the package (`orm.Condition{Field: ...}`) is a **compile error**.
- `Order` structs are **only constructed internally** by `QB.OrderBy()`. Neither consumers nor adapters create them directly.

---

## Execution Phases

### Phase 1: Core Types and Structures (`orm.go`)
**Goal:** Define all agnostic types that represent the ORM contract. Zero external dependencies (`database/sql` forbidden).

1. Define `Action int` with constants: `ActionCreate`, `ActionReadOne`, `ActionUpdate`, `ActionDelete`, `ActionReadAll`.
2. Define `Condition` struct with **unexported fields**: `field string`, `operator string`, `value any`, `logic string`. Add 4 exported getter methods: `Field() string`, `Operator() string`, `Value() any`, `Logic() string`.
3. Define `Order` struct with **unexported fields**: `column string`, `dir string`. Add 2 exported getter methods: `Column() string`, `Dir() string`.
4. Define `Query` struct: `Action`, `Table`, `Columns []string`, `Values []any`, `Conditions []Condition`, `OrderBy []Order`, `GroupBy []string`, `Limit int`, `Offset int`. **No `Args` field.**
5. Define `Model` interface: `TableName() string`, `Columns() []string`, `Values() []any`, `Pointers() []any`.
6. Define `Adapter` interface: single method `Execute(q Query, m Model, factory func() Model, each func(Model)) error`.
7. Define `DB` struct with private `adapter Adapter` field.
8. Implement `New(adapter Adapter) *DB` factory.
9. Implement write methods directly on `DB`:
   - `Create(m Model) error` — builds Query with `ActionCreate`, calls `validate`, then `adapter.Execute`.
   - `Update(m Model, conds ...Condition) error` — builds Query with `ActionUpdate`, calls `validate`.
   - `Delete(m Model, conds ...Condition) error` — builds Query with `ActionDelete` (no Values needed).
10. Define `QB` struct (Query Builder) with fields: `db *DB`, `model Model`, `conds []Condition`, `orderBy []Order`, `groupBy []string`, `limit int`, `offset int`.
11. Implement `Query(m Model) *QB` on `DB`.
12. Implement QB chain methods (each returns `*QB`): `Where`, `Limit`, `Offset`, `OrderBy`, `GroupBy`.
13. Implement QB terminal methods:
    - `ReadOne() error` — sets `ActionReadOne`, limit 1, calls `adapter.Execute(q, model, nil, nil)`. Returns `ErrNotFound` if adapter signals no rows.
    - `ReadAll(factory func() Model, each func(Model)) error` — sets `ActionReadAll`, calls `adapter.Execute(q, nil, factory, each)`.

---

### Phase 2: Condition Helpers (`conditions.go`)
**Goal:** Provide type-safe constructor functions for `Condition` to avoid raw operator strings in application code.

1. Implement `Eq`, `Neq`, `Gt`, `Gte`, `Lt`, `Lte`, `Like` — each returns a `Condition` built via internal struct literal (within the same package, unexported fields are accessible): `return Condition{field: field, operator: "=", value: val, logic: "AND"}`.
2. Implement `Or(c Condition) Condition` — returns a copy of `c` with `logic` set to `"OR"`: `c.logic = "OR"; return c`.

---

### Phase 3: Sentinel Errors (`errors.go`)
**Goal:** Define typed errors that callers can match with `errors.Is`.

1. Declare package-level vars:
   - `ErrNotFound` — returned by `ReadOne()` when the Adapter signals no matching row.
   - `ErrValidation` — returned by `validate` when column/value count mismatches.
   - `ErrEmptyTable` — returned when `m.TableName()` returns `""`.
   - `ErrNoTxSupport` — returned by `DB.Tx()` when the adapter does not implement `TxAdapter`.

---

### Phase 4: Action-Aware Validation (`validate.go`)
**Goal:** Catch model inconsistencies before sending data to the Adapter.

1. Implement `validate(action Action, m Model) error`:
   - For `ActionCreate` and `ActionUpdate` only: check `len(m.Columns()) == len(m.Values())`. Return `ErrValidation` (wrapped with context) if mismatch.
   - For all actions: check `m.TableName() != ""`. Return `ErrEmptyTable` if empty.
   - For `ActionReadOne`, `ActionReadAll`, `ActionDelete`: skip Values/Columns length check entirely.
2. Call `validate` at the beginning of `Create`, `Update`, `Delete`, `ReadOne`, and `ReadAll`.

---

### Phase 5: Transaction Support (`tx.go`)
**Goal:** Implement optional atomic transaction scope via the functional pattern.

1. Define `TxBound` interface:
   - Embeds `Adapter` (so `Execute` works within the transaction).
   - Adds `Commit() error` and `Rollback() error`.

2. Define `TxAdapter` interface:
   - Single method `BeginTx() (TxBound, error)`.
   - Adapters implement this **only if** their engine supports transactions.

3. Implement `DB.Tx(fn func(tx *DB) error) error`:
   - Type-assert `db.adapter` to `TxAdapter`. If not satisfied → return `ErrNoTxSupport`.
   - Call `BeginTx()` → get `bound TxBound`.
   - Create `txDB := &DB{adapter: bound}`.
   - Call `fn(txDB)`. If error → call `bound.Rollback()`, return original error.
   - If nil → call `bound.Commit()`, return its error.

---

### Phase 6: Tests (`tests/`)
**Goal:** 100% coverage of ORM logic using Mocks — no real databases.

1. **`tests/setup_test.go`** — shared setup:
   - `MockAdapter` struct implementing `Adapter`. Stores the last received `Query`, `m`, `factory`, and `each` in public fields. Configurable to return a specific error.
   - `MockModel` struct implementing `Model` (e.g., `User` with ID, Name, Age). Returns deterministic values.

2. **`tests/orm_stlib_test.go`** (`//go:build !wasm`) and **`tests/orm_wasm_test.go`** (`//go:build wasm`):
   - Both call a shared `RunCoreTests(t *testing.T)` function.

3. **`RunCoreTests` must cover:**
   - `Create` → `MockAdapter.LastQuery.Action == ActionCreate`, correct Table/Columns/Values.
   - `Update` with conditions → correct Action, Conditions slice populated.
   - `Delete` with conditions → Action is `ActionDelete`, Values are empty.
   - `Query(m).Where(...).Limit(5).OrderBy("name","ASC").ReadOne()` → Action is `ActionReadOne`, Limit=1 (ReadOne() forces limit 1), OrderBy populated.
   - `Query(m).ReadAll(factory, each)` → Action is `ActionReadAll`, factory/each are non-nil in MockAdapter call.
   - `ErrValidation` is returned when `Columns()` and `Values()` lengths differ on Create.
   - `ErrEmptyTable` is returned when `TableName()` returns `""`.
   - `Or(Eq(...))` sets `Logic = "OR"` on the condition.
   - `db.Tx(fn)` with a `MockTxAdapter` → fn receives a `*DB` backed by `MockTxBound`; on fn success `Commit` is called; on fn error `Rollback` is called.
   - `db.Tx(fn)` with a plain `MockAdapter` (no `TxAdapter`) → returns `ErrNoTxSupport`.

   **Additional mocks needed in `setup_test.go`:**
   - `MockTxBound` struct implementing `TxBound`: records whether `Commit` or `Rollback` was called.
   - `MockTxAdapter` struct implementing both `Adapter` and `TxAdapter`: `BeginTx()` returns a `MockTxBound`.

---

## Verification Workflow

1. After Phase 1–5, verify pure TinyGo compilation: `tinygo build -o test.wasm -target wasm ./...` — no native library errors.
2. Run tests: `gotest`.
3. Ensure module is initialized: `go mod init github.com/tinywasm/orm` (if not already configured).

---

> [!IMPORTANT]
> Never inject literal SQL or IndexedDB-specific components into this directory (`orm/`).
> Any coupling to Postgres (e.g., `SELECT * FROM`, `$1`, `RETURNING id`) or SQLite (`?`) or IndexedDB (JS API calls) belongs **exclusively** in the Adapters (`tinywasm/postgre`, `tinywasm/sqlite`, `tinywasm/indexdb`), which translate the `Query` and live in **separate repositories**.
