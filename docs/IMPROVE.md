# Improvement Roadmap

This document outlines planned improvements and features for `tinywasm/orm`.

## Planned Features

### 1. Generic Helpers
- **`Collect[T]`**: A generic helper to simplify accumulating results from `ReadAll`.
  ```go
  users, err := orm.Collect[User](db.Query(&User{}))
  ```

### 2. Eager Loading (Preload)
- Support for preloading relations to avoid N+1 query problems.
  ```go
  db.Query(&User{}).Preload(User_.Roles).ReadAll(...)
  ```

### 3. Migration Support
- A lightweight, engine-agnostic migration system.

### 4. Many-to-Many Relations
- First-class support for junction tables and many-to-many relationship mapping in `ormc`.

### 5. Performance Optimizations
- Further reduction of allocations during query building and execution.
- Optimization of the `Compiler` for common SQL patterns.

## Developer Experience

### Improved `ormc` CLI
- Better error messages for invalid struct tags.
- Support for custom templates in code generation.
- Watching mode to automatically re-generate code on file changes.
