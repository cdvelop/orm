package orm

// Planner converts ORM queries into engine instructions.
type Planner interface {
	Plan(q Query, m Model) (Plan, error)
}
