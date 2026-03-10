package orm

import "github.com/tinywasm/fmt"

// FieldExt extends fmt.Field with database-specific metadata (foreign keys).
// Used internally by adapters that support FK constraints.
type FieldExt struct {
	fmt.Field
	Ref       string // FK: target table name. Empty = no FK.
	RefColumn string // FK: target column. Empty = auto-detect PK of Ref table.
}
