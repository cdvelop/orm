package tests

import (
	"github.com/tinywasm/orm"
)

// MockCompiler captures the query and returns a predefined plan.
type MockCompiler struct {
	LastQuery  orm.Query
	LastModel  orm.Model
	ReturnPlan orm.Plan
	ReturnErr  error
}

func (m *MockCompiler) Compile(q orm.Query, model orm.Model) (orm.Plan, error) {
	m.LastQuery = q
	m.LastModel = model
	if m.ReturnPlan.Query == "" {
		m.ReturnPlan.Query = "MOCK_QUERY"
	}
	return m.ReturnPlan, m.ReturnErr
}

// MockExecutor captures execution calls.
type MockExecutor struct {
	ExecutedQueries []string
	ExecutedArgs    [][]any
	ReturnExecErr   error
	ReturnQueryRow  orm.Scanner
	ReturnQueryRows orm.Rows
	ReturnQueryErr  error
	ReturnCloseErr  error
}

func (m *MockExecutor) Exec(query string, args ...any) error {
	m.ExecutedQueries = append(m.ExecutedQueries, query)
	m.ExecutedArgs = append(m.ExecutedArgs, args)
	return m.ReturnExecErr
}

func (m *MockExecutor) QueryRow(query string, args ...any) orm.Scanner {
	m.ExecutedQueries = append(m.ExecutedQueries, query)
	m.ExecutedArgs = append(m.ExecutedArgs, args)
	if m.ReturnQueryRow == nil {
		return &MockScanner{}
	}
	return m.ReturnQueryRow
}

func (m *MockExecutor) Query(query string, args ...any) (orm.Rows, error) {
	m.ExecutedQueries = append(m.ExecutedQueries, query)
	m.ExecutedArgs = append(m.ExecutedArgs, args)
	if m.ReturnQueryRows == nil {
		return &MockRows{}, m.ReturnQueryErr
	}
	return m.ReturnQueryRows, m.ReturnQueryErr
}

func (m *MockExecutor) Close() error {
	return m.ReturnCloseErr
}

type MockScanner struct {
	ScanErr error
}

func (m *MockScanner) Scan(dest ...any) error {
	return m.ScanErr
}

type MockRows struct {
	Count    int
	Current  int
	ScanErr  error
	CloseErr error
	ErrVal   error
}

func (m *MockRows) Next() bool {
	if m.Current < m.Count {
		m.Current++
		return true
	}
	return false
}

func (m *MockRows) Scan(dest ...any) error {
	return m.ScanErr
}

func (m *MockRows) Close() error {
	return m.CloseErr
}

func (m *MockRows) Err() error {
	return m.ErrVal
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

// MockTxExecutor ...
type MockTxExecutor struct {
	MockExecutor
	Bound      *MockTxBoundExecutor
	BeginTxErr error
}

func (m *MockTxExecutor) BeginTx() (orm.TxBoundExecutor, error) {
	if m.BeginTxErr != nil {
		return nil, m.BeginTxErr
	}
	if m.Bound == nil {
		m.Bound = &MockTxBoundExecutor{}
	}
	return m.Bound, nil
}

type MockTxBoundExecutor struct {
	MockExecutor
	CommitCalled   bool
	RollbackCalled bool
	CommitErr      error
	RollbackErr    error
}

func (m *MockTxBoundExecutor) Commit() error {
	m.CommitCalled = true
	return m.CommitErr
}

func (m *MockTxBoundExecutor) Rollback() error {
	m.RollbackCalled = true
	return m.RollbackErr
}
