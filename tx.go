package orm

// TxBoundExecutor represents an executor bound to a transaction.
type TxBoundExecutor interface {
	Executor
	Commit() error
	Rollback() error
}

// TxExecutor represents an executor that supports transactions.
type TxExecutor interface {
	Executor
	BeginTx() (TxBoundExecutor, error)
}

// Tx executes a function within a transaction.
func (db *DB) Tx(fn func(tx *DB) error) error {
	txExec, ok := db.exec.(TxExecutor)
	if !ok {
		return ErrNoTxSupport
	}

	bound, err := txExec.BeginTx()
	if err != nil {
		return err
	}

	txDB := &DB{
		exec:     bound,
		compiler: db.compiler,
	}

	if err := fn(txDB); err != nil {
		bound.Rollback()
		return err
	}

	return bound.Commit()
}
