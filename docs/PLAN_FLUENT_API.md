# ORM Plan: Fluent Query Builder API

## Goal
Refactor the `QB` and conditionally the `Condition` structure to allow natural language chaining instead of deeply nested function calls (e.g. `orm.Or(orm.Eq(...))`).

## Implementation Details

1. **Current Friction:**
   ```go
   qb.Where(orm.Or(orm.Eq("age", 18), orm.Gt("score", 100)))
   ```

2. **Proposed API Pattern:**
   ```go
   qb.Where("age").Eq(18).Or().Where("score").Gt(100)
   ```

3. **Internal Refactoring:**
   - Modify `(&QB).Where(column string) *Clause` to return an intermediate type (e.g., `*Clause` or simply `*QB` with temporary internal state waiting for an operator).
   - Add operator methods to complete the condition mapping directly onto the Clause/QB: `Eq(val any)`, `Gt(val any)`, `Like(val any)`.
   - Add logical chaining methods on `QB`: `And()`, `Or()`.
   - Ensure the sequence of fluent calls deterministically compiles into the exact same `Condition` array structure as before, validating it via unit tests against mock parameters.

4. **Testing:**
   - Verify that nested logic (like parentheses in SQL) can either be supported or is intentionally constrained for simplicity to linear evaluation. 
