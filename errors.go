package orm

import "github.com/tinywasm/fmt"

// ErrNotFound is returned when ReadOne() finds no matching row.
var ErrNotFound = fmt.Err("record", "not", "found")

// ErrValidation is returned when validate() finds a mismatch.
var ErrValidation = fmt.Err("error", "validation")

// ErrEmptyTable is returned when TableName() returns an empty string.
var ErrEmptyTable = fmt.Err("name", "table", "empty")

// ErrNoTxSupport is returned by DB.Tx() when the executor does not implement TxExecutor.
var ErrNoTxSupport = fmt.Err("transaction", "not", "supported")
