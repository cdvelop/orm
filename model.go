package orm

import (
	"github.com/tinywasm/fmt"
)

// Model is a type alias for fmt.Model to maintain package-level compatibility.
// We use a local interface if fmt.Model is already dot-imported.
type Model interface {
	ModelName() string
	fmt.Fielder
}
