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

// Order represents a sort order for a query.
// It is a sealed value type constructed via QB.OrderBy().
type Order struct {
	column string
	dir    string
}

func (o Order) Column() string { return o.column }
func (o Order) Dir() string    { return o.dir }

// Query represents a database query to be executed by an Executor.
// Planners read these fields to build Plans.
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
