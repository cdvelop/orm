package orm

import "github.com/tinywasm/fmt"

// Compiler converts ORM queries into engine instructions.
type Compiler interface {
	Compile(q Query, m fmt.Model) (Plan, error)
}
