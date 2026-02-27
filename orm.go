package orm

// Action represents the type of database operation.
type Action int

const (
	ActionCreate Action = iota
	ActionReadOne
	ActionUpdate
	ActionDelete
	ActionReadAll
)

// Condition represents a filter for a query.
// It is a sealed value type constructed via helper functions.
type Condition struct {
	field    string
	operator string
	value    any
	logic    string
}

func (c Condition) Field() string    { return c.field }
func (c Condition) Operator() string { return c.operator }
func (c Condition) Value() any       { return c.value }
func (c Condition) Logic() string    { return c.logic }

// Order represents a sort order for a query.
// It is a sealed value type constructed via QB.OrderBy().
type Order struct {
	column string
	dir    string
}

func (o Order) Column() string { return o.column }
func (o Order) Dir() string    { return o.dir }

// Query represents a database query to be executed by an Adapter.
// Adapters read these fields to build native queries.
type Query struct {
	Action     Action
	Table      string
	Columns    []string
	Values     []any
	Conditions []Condition
	OrderBy    []Order
	GroupBy    []string
	Limit      int
	Offset     int
}

// Model represents a database model.
// Consumers implement this interface.
type Model interface {
	TableName() string
	Columns() []string
	Values() []any
	Pointers() []any
}

// DB represents a database connection.
// Consumers instantiate it via New().
type DB struct {
	adapter Adapter
}

// New creates a new DB instance.
func New(adapter Adapter) *DB {
	return &DB{adapter: adapter}
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
	return db.adapter.Exec(q)
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
	return db.adapter.Exec(q)
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
	return db.adapter.Exec(q)
}

// QB represents a query builder.
// Consumers hold a *QB reference in variables for incremental building.
type QB struct {
	model   Model
	conds   []Condition
	orderBy []Order
	groupBy []string
	limit   int
	offset  int
}

// Query creates a new QB instance.
func (db *DB) Query(m Model) *QB {
	return &QB{
		model: m,
	}
}

// Where adds conditions to the query.
func (qb *QB) Where(conds ...Condition) *QB {
	qb.conds = append(qb.conds, conds...)
	return qb
}

// Limit sets the limit for the query.
func (qb *QB) Limit(limit int) *QB {
	qb.limit = limit
	return qb
}

// Offset sets the offset for the query.
func (qb *QB) Offset(offset int) *QB {
	qb.offset = offset
	return qb
}

// OrderBy adds an order clause to the query.
func (qb *QB) OrderBy(column, dir string) *QB {
	qb.orderBy = append(qb.orderBy, Order{column: column, dir: dir})
	return qb
}

// GroupBy adds a group by clause to the query.
func (qb *QB) GroupBy(columns ...string) *QB {
	qb.groupBy = append(qb.groupBy, columns...)
	return qb
}

// ToQuery returns the constructed Query object.
func (qb *QB) ToQuery() Query {
	return Query{
		Table:      qb.model.TableName(),
		Conditions: qb.conds,
		OrderBy:    qb.orderBy,
		GroupBy:    qb.groupBy,
		Limit:      qb.limit,
		Offset:     qb.offset,
	}
}
