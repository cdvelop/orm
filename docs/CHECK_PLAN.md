# PLAN — ORM: Make `Update()` Condition-Safe at Compile Time

## Context

A critical data-safety bug was found: `db.Update(&model)` called without conditions
generates `UPDATE table SET ... ` with **no WHERE clause**, overwriting every row.

The root problem is not missing runtime validation — it is an **API design flaw**.
The variadic signature `Update(m Model, conds ...Condition)` allows calling the
function with zero conditions, and the Go compiler accepts it silently.

**Resolution:** change the signature so the compiler itself rejects unsafe calls.
No driver changes are needed. No auto-inference magic. No runtime errors to test.

Related plans (no code changes needed — just doc updates in companion libs):
- `tinywasm/sqlite/docs/PLAN.md`
- `tinywasm/postgres/docs/PLAN.md`

---

## Development Rules

- Standard Library only — no external dependencies.
- Max 500 lines per file.
- Run tests with `gotest` (never `go test` directly).
- Publish with `gopush 'message'` after all tests pass.
- **Documentation must be updated before touching code.**
- Prerequisites in isolated environments:
  ```bash
  go install github.com/tinywasm/devflow/cmd/gotest@latest
  ```

---

## The Fix

Change `Update` from variadic-only to **first-condition required**:

```go
// BEFORE — conds is optional; db.Update(&m) compiles and silently corrupts data
func (db *DB) Update(m Model, conds ...Condition) error

// AFTER — first Condition is mandatory; db.Update(&m) no longer compiles
func (db *DB) Update(m Model, cond Condition, rest ...Condition) error
```

Any existing `db.Update(&model)` call (zero conditions) **becomes a compile error**.
This is the desired behavior: every such call in the codebase is a latent data-corruption
bug that must be fixed explicitly. The compiler finds them all for free.

### Why this is strictly superior to runtime validation

| Approach | Detection | Coverage needed | Risk |
|---|---|---|---|
| Runtime `validateUpdate()` | Tests / Production | Must write tests | Missed if not covered |
| Driver auto-infer PK in WHERE | Never fails | — | Implicit magic, hides intent |
| **Mandatory first `Condition`** | **Compile time** | **None** | **Impossible to bypass** |

---

## Step 1 — Update documentation (BEFORE coding)

### 1a. `docs/ARQUITECTURE.md`

In **section 3.6 "Write Operations"**, replace:

```go
func (db *DB) Update(m Model, conds ...Condition) error
func (db *DB) Delete(m Model, conds ...Condition) error
```

with:

```go
// Update modifies an existing row. At least one Condition is required.
// Providing zero conditions is a compile-time error, preventing accidental
// full-table UPDATE statements.
func (db *DB) Update(m Model, cond Condition, rest ...Condition) error

// Delete removes rows matching the given conditions.
// At least one Condition is required to prevent accidental full-table DELETE.
func (db *DB) Delete(m Model, cond Condition, rest ...Condition) error
```

> Also update the sentinel errors list in section 3.7 — remove any mention of
> `ErrNoWhereOnUpdate` if it was previously added.

### 1b. `docs/SKILL.md`

In **"Core Structs"** section, update the `DB` method list:

```
- `DB`: `New(Executor, Compiler)`, `Create`, `Update(m, cond, rest...)`,
        `Delete(m, cond, rest...)`, `Query`, `Tx`, `Close`, `RawExecutor`,
        `CreateTable`, `DropTable`, `CreateDatabase`
```

Add a new **"API Safety Contract"** section after "Core Structs":

```markdown
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
No test coverage is required to guarantee this property.
```

### 1c. `README.md`

Add or update the example in the README that shows `db.Update` usage to always
include at least one explicit condition. Verify no example uses the zero-arg form.

---

## Step 2 — Change `db.go`

In the `Update` method, change the signature and build the conditions slice:

```go
// Update modifies an existing row. At least one Condition is required.
// Providing zero conditions is a compile-time error — there is no variadic
// fallback — preventing accidental full-table UPDATE statements.
func (db *DB) Update(m Model, cond Condition, rest ...Condition) error {
	if err := validate(ActionUpdate, m); err != nil {
		return err
	}
	conds := append([]Condition{cond}, rest...)
	schema := m.Schema()
	columns := make([]string, len(schema))
	for i, f := range schema {
		columns[i] = f.Name
	}
	q := Query{
		Action:     ActionUpdate,
		Table:      m.TableName(),
		Columns:    columns,
		Values:     m.Values(),
		Conditions: conds,
	}
	plan, err := db.compiler.Compile(q, m)
	if err != nil {
		return err
	}
	return db.exec.Exec(plan.Query, plan.Args...)
}
```

> **Note:** Apply the same change to `Delete` for consistency and safety:
> `func (db *DB) Delete(m Model, cond Condition, rest ...Condition) error`

---

## Step 3 — Update tests (`tests/core_test.go`)

1. Remove the old `"Validation Error Update"` sub-test that tested zero-condition
   behavior (it no longer compiles, so the test itself becomes invalid).

2. Ensure all existing `db.Update(model, orm.Eq(...))` calls compile — they already
   provide one condition so they are unchanged.

3. Add one regression comment:

```go
// Compile-time guarantee: db.Update(&m) with zero conditions no longer compiles.
// No test case needed — the Go compiler enforces this contract.
// See: docs/ARQUITECTURE.md section 3.6
```

4. Verify `gotest` passes all existing tests unchanged.

---

## Step 4 — Update downstream call sites

Search across the monorepo for any `db.Update(` or `tx.Update(` with zero conditions:

```bash
grep -rn "\.Update(" . --include="*.go" | grep -v "_test.go" | grep -v "func.*Update"
```

For each occurrence, add the explicit PK condition:

```go
// Before (broken)
tx.Update(got)

// After (correct)
tx.Update(got, orm.Eq(Reservation_.ID, got.ID))
```

Specifically, `tinywasm/velty_modules/appointment-booking`:
- `repository.go` → `UpdateReservationStatus`: change `tx.Update(got)` to
  `tx.Update(got, orm.Eq(Reservation_.ID, got.ID))`
- `PLAN_STAGE_2_ORM.md` → update the transaction example in "Section D — Transactions"

---

## Acceptance Criteria

- [ ] `docs/ARQUITECTURE.md` — `Update` and `Delete` signatures updated
- [ ] `docs/SKILL.md` — "API Safety Contract" section added
- [ ] `README.md` — no example shows zero-condition Update
- [ ] `db.go` — `Update(m, cond, rest...)` compiles cleanly
- [ ] `db.go` — `Delete(m, cond, rest...)` compiles cleanly
- [ ] `gotest` passes with zero failures
- [ ] No `db.Update(&m)` or `tx.Update(&m)` zero-condition calls remain in the repo
- [ ] `gopush 'fix: require at least one Condition in Update/Delete to prevent full-table writes'`
