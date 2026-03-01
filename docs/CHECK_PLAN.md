# Master Plan: Fixing ORM to support `In` clause

This document is the **Master Orchestrator (Implementation Plan)** for adding `IN` clause support to the `tinywasm/orm` module and clarifying architectural decisions raised during the `tinywasm/user` module refactoring.

## Development Rules
- **No Global State:** Avoid direct system calls. Use interfaces and dependency injection.
- **Single Responsibility Principle (SRP):** Files under 500 lines, flat hierarchy.
- **Documentation-First:** Update architecture docs BEFORE coding.
- **Testing Runner (`gotest`):** ALWAYS use the globally installed `gotest` CLI command. Use standard library `testing`, no external assertion libraries.
- **Standard Library Only:** Never use external frameworks or testing libraries.
- **Language Protocol:** Plans and documentation must be generated in **English**.

---

## 🏗 Sequential Execution Phases

### Phase 1: Add `In` Operator Support to `tinywasm/orm` Core
**Target Files:**
- `conditions.go`
- `qb.go`

**Instructions:**
1. **[MODIFY] `conditions.go`**: Add a new function `In(field string, value any) Condition` that returns a `Condition` with `operator: "IN"`.
2. **[MODIFY] `qb.go`**: Add `In(value any) *QB` method to `*Clause`, which calls `c.qb.addCondition(In(c.field, value))`. This mirrors existing methods like `Eq` and `Neq`.
3. Ensure these additions are tested if there are unit tests for condition construction (run `gotest`).

### Phase 2: Final Documentation & Verification
**Target Files:**
- `README.md` (if API changes are documented there)

**Instructions:**
1. Run `gotest` to verify all `orm` tests pass successfully.
2. If `README.md` documents available query conditions, add `In` to the list.
3. Commit the changes using `gopush 'feat: add In condition to ORM and fluent builder'`.

> **Note on Adapters:** The implementers of `tinywasm/sqlite` and `tinywasm/postgres` will need independent plans to handle the query compilation logic for the `IN` operator (e.g., expanding slice values into `(?, ?, ?)` statements). This plan solely covers the Core `tinywasm/orm` API extension.
