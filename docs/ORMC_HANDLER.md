# PLAN: `Ormc` Handler Refactor — `tinywasm/orm`

> **Prerequisite for:** [PLAN.md](PLAN.md) (multi-struct bug fix)
> **Must be completed FIRST before executing PLAN.md**

## Prerequisites

```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
```

---

## Goal

Replace the current package-level functions (`GenerateCodeForStruct`, `RunOrmcCLI`)
with an `Ormc` struct handler that:

1. Holds a configurable log function via `SetLog(func(messages ...any))`
2. Returns `error` from all methods — no `log.Fatal` / `os.Exit` inside the library
3. Keeps `log.Fatal` only in `cmd/ormc/main.go` (the CLI boundary)
4. Allows programmatic integration without any process-killing side effects

---

## Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| DA | `Ormc` is a struct handler with `New() *Ormc` constructor | DI principle: state is owned by the handler, not the package |
| DB | `SetLog(fn func(messages ...any))` on `*Ormc` — no default | No implicit behavior; the CLI sets its own; integrators set theirs |
| DC | `log.Printf` / `log.Fatal` removed from library entirely | A library must never kill the host process or use implicit global loggers |
| DD | All `Ormc` methods return `error` — no `log.Fatal` inside | Fatal conditions become `error` returns; CLI converts them to `log.Fatal` |
| DE | **Breaking change — no backward-compat shims** | Shims are dead code; callers must use `orm.New().<Method>(...)` |
| DF | Log signature: `func(messages ...any)` — simplest form | Caller formats with `tinywasm/fmt` before calling; no format string coupling |
| DG | `Ormc` handler lives in new file `ormc_handler.go` | SRP: handler struct separate from code-generation logic in `ormc.go` |
| DH | `SetRootDir(dir string)` sets the scan root; **default is `"."**` | Ideal for tests: no `os.Chdir` needed; CLI needs no `os.Getwd()` |

---

## New API

```go
// Ormc is the code generator handler.
// Use New() to create an instance, configure with SetLog/SetRootDir, then call Run.
type Ormc struct {
    logFn   func(messages ...any)
    rootDir string // default: "."
}

// New creates a new Ormc handler with rootDir defaulting to ".".
func New() *Ormc {
    return &Ormc{rootDir: "."}
}

// SetLog sets the function used for warnings and informational messages.
// If not set, messages are silently discarded.
func (o *Ormc) SetLog(fn func(messages ...any)) {
    o.logFn = fn
}

// SetRootDir sets the root directory that Run() will scan.
// Default is ".", which resolves to the current working directory.
// Useful in tests to point to a specific fixture directory.
func (o *Ormc) SetRootDir(dir string) {
    o.rootDir = dir
}

// log calls the configured log function if set.
func (o *Ormc) log(messages ...any) {
    if o.logFn != nil {
        o.logFn(messages...)
    }
}

// GenerateForStruct parses a single struct and generates its output file.
func (o *Ormc) GenerateForStruct(structName string, goFile string) error { ... }

// GenerateForFile writes ORM methods for all infos into a single output file.
func (o *Ormc) GenerateForFile(infos []StructInfo, sourceFile string) error { ... }

// ParseStruct parses a single struct from a Go file. Returns StructInfo.
func (o *Ormc) ParseStruct(structName string, goFile string) (StructInfo, error) { ... }

// Run scans o.rootDir for model.go/models.go files and generates _db.go files.
// Returns error instead of calling os.Exit.
func (o *Ormc) Run() error { ... }
```

---

## Updated CLI (`cmd/ormc/main.go`)

```go
//go:build !wasm

package main

import (
    "fmt"
    "log"
    "os"

    "github.com/tinywasm/orm"
)

func main() {
    o := orm.New()
    o.SetLog(func(messages ...any) {
        fmt.Fprintln(os.Stderr, messages...)
    })
    // rootDir defaults to "." — no os.Getwd() needed
    if err := o.Run(); err != nil {
        log.Fatalf("ormc: %v", err)
    }
}
```

> `log.Fatal` is only here, in the CLI boundary. Never inside the library.

---

## Affected Files

| File | Change |
|------|---------|
| `ormc.go` | All old functions **deleted**; logic moved to `*Ormc` methods; `Run()` uses `o.rootDir`; remove all `log.*` |
| `ormc_handler.go` | **New file**: `Ormc` struct, `New()`, `SetLog()`, `SetRootDir()`, `log()` |
| `cmd/ormc/main.go` | Replaced: `orm.New()` + `o.SetLog(...)` + `o.Run()` |
| `tests/ormc_test.go` | **Rewritten**: all calls updated to `orm.New().<Method>(...)` |

---

## Execution Steps

### Step 1 — Create `ormc_handler.go`

```go
//go:build !wasm

package orm

// Ormc is the code generator handler for the ormc tool.
type Ormc struct {
    logFn   func(messages ...any)
    rootDir string
}

// New creates a new Ormc handler with rootDir defaulting to ".".
func New() *Ormc {
    return &Ormc{rootDir: "."}
}

// SetLog sets the log function for warnings and informational messages.
// If not set, messages are silently discarded.
func (o *Ormc) SetLog(fn func(messages ...any)) {
    o.logFn = fn
}

// SetRootDir sets the root directory that Run() will scan.
// Defaults to ".". Useful in tests to point to a specific directory
// without needing os.Chdir.
func (o *Ormc) SetRootDir(dir string) {
    o.rootDir = dir
}

// log emits a message via the configured log function, if any.
func (o *Ormc) log(messages ...any) {
    if o.logFn != nil {
        o.logFn(messages...)
    }
}
```

### Step 2 — Migrate `ormc.go`: move functions to `*Ormc` methods

**Delete** the old package-level functions entirely. Move their logic to `*Ormc` methods:

| Old function (delete) | New method |
|----------------------|------------|
| `GenerateCodeForStruct(name, file) error` | `(o *Ormc) GenerateForStruct(name, file string) error` |
| `ParseStructInfo(name, file) (StructInfo, error)` | `(o *Ormc) ParseStruct(name, file string) (StructInfo, error)` |
| `generateCodeFile(infos, file) error` | `(o *Ormc) GenerateForFile(infos []StructInfo, file string) error` |
| `RunOrmcCLI()` | `(o *Ormc) Run() error` — uses `o.rootDir` (no parameter) |

Replace all `log.Printf(...)` with `o.log(...)`.
Replace all `log.Fatalf(...)` / `log.Fatal(...)` with `return Err(...)`.
Remove the `"log"` import from `ormc.go`.

### Step 3 — Update `cmd/ormc/main.go`

Replace `orm.RunOrmcCLI()` with the new handler. No `os.Getwd()` needed:

```go
func main() {
    o := orm.New()
    o.SetLog(func(messages ...any) {
        fmt.Fprintln(os.Stderr, messages...)
    })
    if err := o.Run(); err != nil {
        log.Fatalf("ormc: %v", err)
    }
}
```

This is the ONLY place in the codebase that calls `log.Fatal`.

### Step 4 — Rewrite `tests/ormc_test.go`

All calls to old package-level functions must be replaced with `orm.New()` calls:

| Old call | New call |
|----------|----------|
| `orm.GenerateCodeForStruct(name, file)` | `orm.New().GenerateForStruct(name, file)` |
| `orm.ParseStructInfo(name, file)` | `orm.New().ParseStruct(name, file)` |
| `orm.GenerateCodeForFile(infos, file)` | `orm.New().GenerateForFile(infos, file)` |

### Step 5 — Verify build

```bash
go build ./...
```

### Step 6 — Run tests

```bash
gotest
```

All tests must pass with the new API calls.


---

## Acceptance Criteria

| Criterion | Check |
|-----------|-------|
| `Ormc` struct has `logFn` and `rootDir` fields | ✅ |
| `New()` initializes `rootDir = "."` | ✅ |
| `SetLog(func(...any))` sets log function | ✅ |
| `SetRootDir(dir string)` sets scan root | ✅ |
| `Run()` uses `o.rootDir` (no parameter) | ✅ |
| No `log.*` import in `ormc.go` | ✅ |
| No `log.Fatal` / `os.Exit` anywhere in the library (only `cmd/`) | ✅ |
| Old package-level functions (`GenerateCodeForStruct`, `ParseStructInfo`, `RunOrmcCLI`) **deleted** | ✅ |
| `cmd/ormc/main.go` uses `o.Run()` without `os.Getwd()` | ✅ |
| All `Ormc` methods return `error` | ✅ |
| `tests/ormc_test.go` rewritten to use `orm.New().<Method>(...)` | ✅ |
| `go build ./...` passes | ✅ |
| `gotest` passes | ✅ |
