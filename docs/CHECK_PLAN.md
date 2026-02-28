# ORM Improvement Plan: Developer Experience (DX)

## Development Rules
- **No Reflection:** Zero use of `reflect`. Operations must be statically typed.
- **WASM Friendly:** Keep allocations low and binaries small. No `database/sql` in the core.
- **Dependency Injection:** Existing architecture (`Model`, `Compiler`, `Executor`) remains intact.
- **Language Protocol:** Plans and documentation must be in English.

## Overview
This plan outlines the steps to significantly improve the Developer Experience (DX) of `tinywasm/orm` by introducing code generation and adapting the public API without compromising the core architecture's performance and WASM compatibility constraints. 

Generics and `reflect` are strictly excluded from this solution. Instead, developer ergonomics are achieved via explicit code generation (`cmd/ormc/main.go`).

## Execution Steps

The improvements are separated into 4 independent steps. Each step has its own detailed plan document:

1. **Step 1: Code Generator for Boilerplate Elimination**
   - Goal: Create `cmd/ormc/main.go` to automatically implement the `Model` interface (`Values()`, `Pointers()`, `Columns()`).
   - Rule: No struct tags. Use `tinywasm/fmt` to convert Go struct field names naturally into `snake_case` column names.
   - Document: [PLAN_GENERATOR.md](PLAN_GENERATOR.md)

2. **Step 2: Table Descriptors for Type Safety (No Magic Strings)**
   - Goal: Extend the generator to produce metadata structs (e.g. `UserMeta`) providing strongly typed constants/variables for table and column names.
   - Document: [PLAN_METADATA.md](PLAN_METADATA.md)

3. **Step 3: Strongly Typed Collections (No Generics)**
   - Goal: Replace the complex `ReadAll(new, onRow)` callback hell for users by generating explicit read functions per struct (e.g. `ReadAllUser(qb *QB) ([]*User, error)`).
   - Document: [PLAN_COLLECTION.md](PLAN_COLLECTION.md)

4. **Step 4: Fluent Query Builder API**
   - Goal: Refactor `QB` and `Condition` to support a sequential, natural-language-like query construction without nesting `orm.Eq(X, Y)`.
   - Document: [PLAN_FLUENT_API.md](PLAN_FLUENT_API.md)

## Testing Strategy
- **Unit Testing:** Ensure the new CLI tool accurately parses ASTs and generates correct code.
- **Integration Testing:** Run the existing mock executors against the newly auto-generated models and the new fluent query builder.
- **Command:** `gotest`
