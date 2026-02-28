package orm

// Plan describes how the Executor should run the operation.
type Plan struct {
	Mode   Action
	Query  string
	Args   []any
}
