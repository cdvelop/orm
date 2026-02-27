package orm

type Scanner interface {
	Scan(dest ...any) error
}

type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
	Err() error
}

// Adapter receives a Query struct (intent), NOT a SQL string.
type Adapter interface {
	Exec(q Query) error
	QueryRow(q Query) Scanner
	Query(q Query) (Rows, error)
}
