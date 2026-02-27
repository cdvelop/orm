package orm

// DB represents a database connection.
// Consumers instantiate it via New().
type DB struct {
	exec    Executor
	planner Planner
}

// New creates a new DB instance.
func New(exec Executor, planner Planner) *DB {
	return &DB{
		exec:    exec,
		planner: planner,
	}
}

// Create inserts a new model into the database.
func (db *DB) Create(m Model) error {
	if err := validate(ActionCreate, m); err != nil {
		return err
	}
	q := Query{
		Action:  ActionCreate,
		Table:   m.TableName(),
		Columns: m.Columns(),
		Values:  m.Values(),
	}
	plan, err := db.planner.Plan(q, m)
	if err != nil {
		return err
	}
	return db.exec.Exec(plan.Query, plan.Args...)
}

// Update updates a model in the database.
func (db *DB) Update(m Model, conds ...Condition) error {
	if err := validate(ActionUpdate, m); err != nil {
		return err
	}
	q := Query{
		Action:     ActionUpdate,
		Table:      m.TableName(),
		Columns:    m.Columns(),
		Values:     m.Values(),
		Conditions: conds,
	}
	plan, err := db.planner.Plan(q, m)
	if err != nil {
		return err
	}
	return db.exec.Exec(plan.Query, plan.Args...)
}

// Delete deletes a model from the database.
func (db *DB) Delete(m Model, conds ...Condition) error {
	if err := validate(ActionDelete, m); err != nil {
		return err
	}
	q := Query{
		Action:     ActionDelete,
		Table:      m.TableName(),
		Conditions: conds,
	}
	plan, err := db.planner.Plan(q, m)
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
