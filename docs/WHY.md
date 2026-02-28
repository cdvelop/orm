# Why tinywasm/orm?

## What Problem This Library Solves

Modern applications often need to integrate a database before the final storage engine is known, or while keeping the project easily testable and portable across environments (backend, edge, or WebAssembly). Traditional ORMs and database libraries typically bind the application too early to a specific database driver, SQL dialect, or runtime environment.

`tinywasm/orm` solves this by providing a minimal, explicit, engine-agnostic ORM core that allows developers to add database capabilities to a project without committing to a specific database engine upfront.

Instead of coupling application logic directly to a database driver, the ORM separates responsibilities into clear layers:

`Model` → `Query` → `Compiler` → `Plan` → `Executor`

This architecture allows the same application code to run with different storage engines such as SQL databases, embedded databases, or browser storage systems like IndexedDB, simply by swapping the compiler and executor.

This makes the library particularly useful for:
- Projects where the database engine is undecided early in development.
- Applications that must run in multiple environments (backend, WASM, edge).
- Systems that prioritize testability and clean architecture.
- Lightweight services that do not need a heavy ORM.

---

## Key Advantages

- **Engine Agnostic:** Application logic does not depend on a specific database engine. The storage backend can change without rewriting queries or business logic.
- **Excellent for Testing:** A mock executor can be injected, allowing full testing of database logic without running a real database.
- **WASM Friendly:** No reflection and no dependency on `database/sql` in the core. This keeps binaries small and compatible with WebAssembly environments.
- **Explicit and Predictable:** The ORM avoids hidden behavior and runtime schema inspection. Queries are built explicitly and compiled deterministically.
- **Minimal and Lightweight:** Designed to stay small, fast, and easy to reason about, especially compared to traditional ORMs.
- **Multi-Engine Architecture:** The same ORM layer can support SQL databases, browser databases, or custom storage engines.
- **Streaming Query Results:** Large result sets can be processed without allocating large slices in memory.

---

## Trade-offs

This design intentionally avoids some features commonly found in traditional ORMs.

- **More Explicit Code:** Models must define their schema manually, which adds some boilerplate but improves performance and clarity.
- **No Automatic Schema Discovery:** The ORM does not inspect database schemas at runtime.
- **Less "Magic" Than Traditional ORMs:** The library favors transparency and control over convenience abstractions.

---

## When to Use This ORM

`tinywasm/orm` is a good fit when:
- You want to keep your application independent from a specific database.
- You need strong testability and clean dependency injection.
- Your project targets WebAssembly or constrained environments.
- You prefer explicit architecture over implicit ORM behavior.
- You are building a framework, platform, or reusable system.

It may not be ideal if:
- You want a fully automated ORM with migrations, schema discovery, and heavy automation.
- You are tightly coupled to a single SQL engine and prefer driver-level APIs.
