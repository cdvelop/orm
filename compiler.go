package orm

// Compiler converts ORM queries into engine instructions.
type Compiler interface {
	Compile(q Query, m Model) (Plan, error)
}
