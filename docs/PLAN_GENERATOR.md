# ORM Plan: Boilerplate Generator (`ormgen`)

## Goal
Create a CLI tool at `cmd/ormgen/main.go` to parse Go source files using the `//go:generate ormgen` directive and output the code necessary to satisfy the `orm.Model` interface, without relying on struct tags.

## Implementation Details

1. **Parsing Ast:**
   - Use `go/parser` and `go/ast` to scan the file for the target struct.
   - Identify exported fields (ignoring unexported ones).
   
2. **Column Naming (Snake Case):**
   - Import `github.com/tinywasm/fmt`.
   - Convert struct field names (e.g., `UserID`) into database column names (e.g., `"user_id"`) using the `SnakeLow` method from `tinywasm/fmt`.
   - **Primary Key Selection:** Use `fmt.IDorPrimaryKey(tableName, fieldName)` from `tinywasm/fmt` to determine if a field is the Primary Key. The first field that returns `true` for `isPK` should be treated as the Model's primary key.
   - **Crucial:** No struct tags (`orm:"id"`) are parsed or required. The generation relies 100% on the Go struct definition and `tinywasm/fmt` conventions.

3. **Code Generation Output (`<file>_orm.go`):**
   - For a struct `User`, generate:
     - `func (m *User) TableName() string` (Defaulting to snake_case of the struct name + "s", e.g., "users").
     - `func (m *User) Columns() []string`
     - `func (m *User) Values() []any`
     - `func (m *User) Pointers() []any`

4. **Testing:**
   - Create a dummy module inside a test folder simulating a struct.
   - Assert the generated `_orm.go` file correctly compiles and matches expected string output.
