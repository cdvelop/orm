# ORM Plan: Metadata Descriptors

## Goal
Eliminate "magic strings" in where clauses and queries by extending `cmd/ormgen/main.go` to generate a static Schema Descriptor.

## Implementation Details

1. **Descriptor Struct:**
   - For a struct `User`, generate a global or package-level variable `UserMeta` (or a struct type `_UserMeta`).
   - The descriptor will contain the table name and the literal string representations of every column automatically derived in `PLAN_GENERATOR.md`.

2. **Generated Output Example:**
   ```go
   var UserMeta = struct {
       TableName string
       // Columns
       ID       string
       Username string
   }{
       TableName: "users",
       ID:        "id",
       Username:  "username",
   }
   ```

3. **Usage:**
   - Developers will use `UserMeta.Username` instead of `"username"` in their queries.
   - If a developer renames the `Username` field in the `User` struct, the code generator updates `UserMeta.Username`. Any hardcoded queries relying on standard strings would pass compilation but fail in execution, whereas `UserMeta` ensures compile-time safety across the codebase.
