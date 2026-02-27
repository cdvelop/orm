package orm

// Executor represents the database connection abstraction.
// It must remain compatible with sql.DB, sql.Tx, mocks, and WASM drivers.
type Executor interface {
	Exec(query string, args ...any) error
	QueryRow(query string, args ...any) Scanner
	Query(query string, args ...any) (Rows, error)
}

// Scanner represents a single row scanner.
type Scanner interface {
	Scan(dest ...any) error
}

// Rows represents an iterator over query results.
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
	Err() error
}
