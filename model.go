package orm

// Model represents a database model.
// Consumers implement this interface.
type Model interface {
	TableName() string
	Columns() []string
	Values() []any
	Pointers() []any
}
