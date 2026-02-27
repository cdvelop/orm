package tests

import (
	"errors"
	"github.com/tinywasm/orm"
)

// MockScanner simulates reading a single row.
type MockScanner struct {
	Data []any
	Err  error
}

func (m *MockScanner) Scan(dest ...any) error {
	if m.Err != nil {
		return m.Err
	}
	if len(dest) != len(m.Data) {
		return errors.New("mock: column count mismatch")
	}

	for i, val := range m.Data {
		switch ptr := dest[i].(type) {
		case *string:
			*ptr = val.(string)
		case *int:
			*ptr = val.(int)
		case *bool:
			*ptr = val.(bool)
		default:
			return errors.New("mock: unsupported pointer type")
		}
	}
	return nil
}

// MockRows simulates iterating multiple rows.
type MockRows struct {
	RowsData [][]any
	Index    int
	ErrState error
}

func (m *MockRows) Next() bool {
	// Simple implementation: checks if Index is within bounds.
	// We increment Index in Scan. This assumes Scan is called for every Next()=true.
	return m.Index < len(m.RowsData)
}

func (m *MockRows) Scan(dest ...any) error {
	if m.ErrState != nil {
		return m.ErrState
	}
	if m.Index >= len(m.RowsData) {
		return errors.New("mock: no more rows")
	}

	row := m.RowsData[m.Index]
	m.Index++ // Advance here

	for i, val := range row {
		switch ptr := dest[i].(type) {
		case *string:
			*ptr = val.(string)
		case *int:
			*ptr = val.(int)
		case *bool:
			*ptr = val.(bool)
		default:
			return errors.New("mock: unsupported pointer type")
		}
	}
	return nil
}

func (m *MockRows) Close() error {
	return nil
}

func (m *MockRows) Err() error {
	return m.ErrState
}

// MockAdapter implements orm.Adapter for testing.
type MockAdapter struct {
	LastQuery  orm.Query
	ExecFn     func(q orm.Query) error
	QueryRowFn func(q orm.Query) orm.Scanner
	QueryFn    func(q orm.Query) (orm.Rows, error)
	ReturnErr  error
}

func (m *MockAdapter) Exec(q orm.Query) error {
	m.LastQuery = q
	if m.ExecFn != nil {
		return m.ExecFn(q)
	}
	return m.ReturnErr
}

func (m *MockAdapter) QueryRow(q orm.Query) orm.Scanner {
	m.LastQuery = q
	if m.QueryRowFn != nil {
		return m.QueryRowFn(q)
	}
	return &MockScanner{Err: orm.ErrNotFound}
}

func (m *MockAdapter) Query(q orm.Query) (orm.Rows, error) {
	m.LastQuery = q
	if m.QueryFn != nil {
		return m.QueryFn(q)
	}
	return &MockRows{RowsData: [][]any{}}, nil
}

// MockModel is a mock implementation of the Model interface.
type MockModel struct {
	Table string
	Cols  []string
	Vals  []any
}

func (m MockModel) TableName() string { return m.Table }
func (m MockModel) Columns() []string { return m.Cols }
func (m MockModel) Values() []any     { return m.Vals }
func (m MockModel) Pointers() []any   { return nil }

// MockUser is a concrete model for testing reads.
type MockUser struct {
	ID   int
	Name string
}

func (u *MockUser) TableName() string { return "users" }
func (u *MockUser) Columns() []string { return []string{"id", "name"} }
func (u *MockUser) Values() []any     { return []any{u.ID, u.Name} }
func (u *MockUser) Pointers() []any   { return []any{&u.ID, &u.Name} }

// MockTxBound ...
type MockTxBound struct {
	MockAdapter
	CommitCalled   bool
	RollbackCalled bool
	CommitErr      error
	RollbackErr    error
}

func (m *MockTxBound) Commit() error {
	m.CommitCalled = true
	return m.CommitErr
}

func (m *MockTxBound) Rollback() error {
	m.RollbackCalled = true
	return m.RollbackErr
}

// MockTxAdapter ...
type MockTxAdapter struct {
	MockAdapter
	Bound *MockTxBound
	BeginTxErr error
}

func (m *MockTxAdapter) BeginTx() (orm.TxBound, error) {
	if m.BeginTxErr != nil {
		return nil, m.BeginTxErr
	}
	if m.Bound == nil {
		m.Bound = &MockTxBound{}
	}
	return m.Bound, nil
}
