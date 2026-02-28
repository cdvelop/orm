# Implementation Plan: Eliminate Global State from Database Adapters

## Development Rules
- **WASM Environment (`tinywasm`):** Frontend Go Compatibility requires standard library replacements (`tinywasm/fmt`).
- **Single Responsibility Principle (SRP):** Every file must have a single, well-defined purpose.
- **Mandatory Dependency Injection (DI):** No global state. Interfaces for external dependencies.
- **Testing Runner (`gotest`):** ALWAYS use the globally installed `gotest` CLI command. If it is not installed, install it via: `go install github.com/tinywasm/devflow/cmd/gotest@latest`.
- **Documentation First:** Update docs before coding.

## Goal
The adapters (`sqlite`, `postgres`, `indexdb`) currently use a global registry (`dbRegistry` and `dbMu`) to associate an `*orm.DB` instance with its underlying driver connection. This is necessary because they need to be able to close the connection or execute raw SQL, but `*orm.DB` encapsulates its `Executor` privately without exposing a `Close()` or `RawExecutor()` method.

The goal is to update the `tinywasm/orm` library to natively support closing connections and exposing the raw executor, thereby allowing the adapters to delete their global registry variables and strictly follow the "No Global State" rule.

## Execution Steps

### 1. Update `orm.Executor` Interface
- Modify `github.com/tinywasm/orm/executor.go`.
- Add a `Close() error` method to the `Executor` interface.
- Ensure any mock executors used in tests within `tinywasm/orm` are updated to implement `Close() error` (returning `nil`).

### 2. Update `orm.DB` Struct
- Modify `github.com/tinywasm/orm/db.go`.
- Add a `Close() error` method to `*orm.DB` that simply calls `db.exec.Close()`.
- Add a `RawExecutor() Executor` method to `*orm.DB` that returns `db.exec`. This will allow adapters to perform `ExecSQL` operations directly on the instance without needing a global map.

### 3. Verify `tinywasm/orm` Tests
- Run `gotest` inside `tinywasm/orm`.
- Ensure all internal tests pass after the interface extensions. Fix any broken mocks.
- Ensure test coverage remains >90%.

### 4. Update Documentation
- Check `README.md` and any API documentation in `docs/` or `fmt` directives for the `orm` module.
- Add documentation explaining the new `Close()` and `RawExecutor()` methods.

### 5. Refactor Adapters (Subsequent Task)
- **Note:** The actual removal of global state from `sqlite`, `postgres`, and `indexdb` will be a follow-up action once this `orm` API update is tested, tagged, and published.

## Verification Plan
### Automated Tests
- Run `gotest` in `tinywasm/orm` to verify no regressions in the base ORM logic.
- Ensure `Clean()` or dummy `Close()` methods in ORM tests report correctly.
