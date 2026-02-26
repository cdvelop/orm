package orm

import "github.com/tinywasm/fmt"

var ErrNotFound = fmt.Err("record", "not", "found")
var ErrValidation = fmt.Err("error", "validation")
var ErrEmptyTable = fmt.Err("table", "name", "empty")
var ErrNoTxSupport = fmt.Err("transaction", "not", "supported")
