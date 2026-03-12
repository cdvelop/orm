package orm

import "github.com/tinywasm/fmt"

// DB represents a database connection.
// Consumers instantiate it via New().
type DB struct {
	exec     Executor
	compiler Compiler
}

// New creates a new DB instance.
func New(exec Executor, compiler Compiler) *DB {
	return &DB{
		exec:     exec,
		compiler: compiler,
	}
}

// Create inserts a new model into the database.
func (db *DB) Create(m Model) error {
	if err := validate(ActionCreate, m); err != nil {
		return err
	}
	schema := m.Schema()
	columns := make([]string, len(schema))
	for i, f := range schema {
		columns[i] = f.Name
	}
	q := Query{
		Action:  ActionCreate,
		Table:   m.TableName(),
		Columns: columns,
		Values:  fmt.ReadValues(schema, m.Pointers()),
	}
	plan, err := db.compiler.Compile(q, m)
	if err != nil {
		return err
	}
	return db.exec.Exec(plan.Query, plan.Args...)
}

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
		Values:     fmt.ReadValues(schema, m.Pointers()),
		Conditions: conds,
	}
	plan, err := db.compiler.Compile(q, m)
	if err != nil {
		return err
	}
	return db.exec.Exec(plan.Query, plan.Args...)
}

// emptyModel is a private zero-value type used only for CreateDatabase.
type emptyModel struct{}

func (e emptyModel) TableName() string { return "" }
func (e emptyModel) Schema() []fmt.Field { return nil }
func (e emptyModel) Pointers() []any   { return nil }

// CreateTable creates a new table for the given model.
func (db *DB) CreateTable(m Model) error {
	if err := validate(ActionCreateTable, m); err != nil {
		return err
	}
	q := Query{
		Action: ActionCreateTable,
		Table:  m.TableName(),
	}
	plan, err := db.compiler.Compile(q, m)
	if err != nil {
		return err
	}
	return db.exec.Exec(plan.Query, plan.Args...)
}

// DropTable drops the table for the given model.
func (db *DB) DropTable(m Model) error {
	if err := validate(ActionDropTable, m); err != nil {
		return err
	}
	q := Query{
		Action: ActionDropTable,
		Table:  m.TableName(),
	}
	plan, err := db.compiler.Compile(q, m)
	if err != nil {
		return err
	}
	return db.exec.Exec(plan.Query, plan.Args...)
}

// CreateDatabase creates a new database.
func (db *DB) CreateDatabase(name string) error {
	m := emptyModel{}
	if err := validate(ActionCreateDatabase, m); err != nil {
		return err
	}
	q := Query{
		Action:   ActionCreateDatabase,
		Database: name,
	}
	plan, err := db.compiler.Compile(q, m)
	if err != nil {
		return err
	}
	return db.exec.Exec(plan.Query, plan.Args...)
}

// Delete deletes a model from the database.
// At least one Condition is required. Providing zero conditions is a compile-time
// error, preventing accidental full-table DELETE statements.
func (db *DB) Delete(m Model, cond Condition, rest ...Condition) error {
	if err := validate(ActionDelete, m); err != nil {
		return err
	}
	conds := append([]Condition{cond}, rest...)
	q := Query{
		Action:     ActionDelete,
		Table:      m.TableName(),
		Conditions: conds,
	}
	plan, err := db.compiler.Compile(q, m)
	if err != nil {
		return err
	}
	return db.exec.Exec(plan.Query, plan.Args...)
}

// Query creates a new QB instance.
func (db *DB) Query(m Model) *QB {
	return &QB{
		db:    db,
		model: m,
	}
}

// Close closes the underlying executor if it supports it.
func (db *DB) Close() error {
	return db.exec.Close()
}

// RawExecutor returns the underlying executor instance.
func (db *DB) RawExecutor() Executor {
	return db.exec
}
