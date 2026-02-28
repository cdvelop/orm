package tests

import (
	"errors"
	"strings"
	"testing"

	"github.com/tinywasm/orm"
)

func RunCoreTests(t *testing.T) {
	// 1. Test Create
	t.Run("Create", func(t *testing.T) {
		mockCompiler := &MockCompiler{}
		mockExec := &MockExecutor{}
		db := orm.New(mockExec, mockCompiler)

		model := &MockModel{
			Table: "users",
			Cols:  []string{"name", "age"},
			Vals:  []any{"Alice", 30},
		}

		err := db.Create(model)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if mockCompiler.LastQuery.Action != orm.ActionCreate {
			t.Errorf("Expected ActionCreate, got %v", mockCompiler.LastQuery.Action)
		}
		if mockCompiler.LastQuery.Table != "users" {
			t.Errorf("Expected table 'users', got '%s'", mockCompiler.LastQuery.Table)
		}
		if len(mockCompiler.LastQuery.Columns) != 2 {
			t.Errorf("Expected 2 columns, got %d", len(mockCompiler.LastQuery.Columns))
		}
	})

	// 2. Test Update with Conditions
	t.Run("Update", func(t *testing.T) {
		mockCompiler := &MockCompiler{}
		mockExec := &MockExecutor{}
		db := orm.New(mockExec, mockCompiler)

		model := &MockModel{
			Table: "users",
			Cols:  []string{"age"},
			Vals:  []any{31},
		}

		err := db.Update(model, orm.Eq("name", "Alice"))
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		if mockCompiler.LastQuery.Action != orm.ActionUpdate {
			t.Errorf("Expected ActionUpdate, got %v", mockCompiler.LastQuery.Action)
		}
		if len(mockCompiler.LastQuery.Conditions) != 1 {
			t.Errorf("Expected 1 condition, got %d", len(mockCompiler.LastQuery.Conditions))
		}
		if mockCompiler.LastQuery.Conditions[0].Field() != "name" {
			t.Errorf("Expected condition field 'name', got '%s'", mockCompiler.LastQuery.Conditions[0].Field())
		}
	})

	// 3. Test Delete
	t.Run("Delete", func(t *testing.T) {
		mockCompiler := &MockCompiler{}
		mockExec := &MockExecutor{}
		db := orm.New(mockExec, mockCompiler)

		model := &MockModel{Table: "users"}

		err := db.Delete(model, orm.Gt("age", 100))
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		if mockCompiler.LastQuery.Action != orm.ActionDelete {
			t.Errorf("Expected ActionDelete, got %v", mockCompiler.LastQuery.Action)
		}
		if len(mockCompiler.LastQuery.Conditions) != 1 {
			t.Errorf("Expected 1 condition, got %d", len(mockCompiler.LastQuery.Conditions))
		}
	})

	// 4. Test Query Chain (ReadOne)
	t.Run("ReadOne", func(t *testing.T) {
		mockCompiler := &MockCompiler{}
		mockExec := &MockExecutor{}
		db := orm.New(mockExec, mockCompiler)

		model := &MockModel{Table: "users"}

		// Setup MockExecutor to return a scanner that succeeds
		mockExec.ReturnQueryRow = &MockScanner{}

		err := db.Query(model).
			Where(orm.Eq("id", 1)).
			OrderBy("created_at", "DESC").
			ReadOne()

		if err != nil {
			t.Fatalf("ReadOne failed: %v", err)
		}

		if mockCompiler.LastQuery.Action != orm.ActionReadOne {
			t.Errorf("Expected ActionReadOne, got %v", mockCompiler.LastQuery.Action)
		}
		if mockCompiler.LastQuery.Limit != 1 {
			t.Errorf("Expected Limit 1, got %d", mockCompiler.LastQuery.Limit)
		}
		if len(mockCompiler.LastQuery.OrderBy) != 1 {
			t.Errorf("Expected 1 OrderBy, got %d", len(mockCompiler.LastQuery.OrderBy))
		}
	})

	// Test ReadOne Validation Error
	t.Run("ReadOne Validation Error", func(t *testing.T) {
		mockCompiler := &MockCompiler{}
		mockExec := &MockExecutor{}
		db := orm.New(mockExec, mockCompiler)
		model := &MockModel{Table: ""} // Empty table

		err := db.Query(model).ReadOne()
		if !errors.Is(err, orm.ErrEmptyTable) {
			t.Errorf("Expected ErrEmptyTable, got %v", err)
		}
	})

	// 5. Test ReadAll
	t.Run("ReadAll", func(t *testing.T) {
		mockCompiler := &MockCompiler{}
		mockExec := &MockExecutor{}
		db := orm.New(mockExec, mockCompiler)

		model := &MockModel{Table: "users"}

		// Simulate 2 rows
		mockRows := &MockRows{Count: 2}
		mockExec.ReturnQueryRows = mockRows

		newCalled := 0
		onRowCalled := 0
		newFunc := func() orm.Model {
			newCalled++
			return &MockModel{}
		}
		onRow := func(m orm.Model) {
			onRowCalled++
		}

		err := db.Query(model).ReadAll(newFunc, onRow)
		if err != nil {
			t.Fatalf("ReadAll failed: %v", err)
		}

		if mockCompiler.LastQuery.Action != orm.ActionReadAll {
			t.Errorf("Expected ActionReadAll, got %v", mockCompiler.LastQuery.Action)
		}
		if newCalled != 2 {
			t.Errorf("Expected new called 2 times, got %d", newCalled)
		}
		if onRowCalled != 2 {
			t.Errorf("Expected onRow called 2 times, got %d", onRowCalled)
		}
	})

	// Test ReadAll Validation Error
	t.Run("ReadAll Validation Error", func(t *testing.T) {
		mockCompiler := &MockCompiler{}
		mockExec := &MockExecutor{}
		db := orm.New(mockExec, mockCompiler)
		model := &MockModel{Table: ""} // Empty table

		err := db.Query(model).ReadAll(nil, nil)
		if !errors.Is(err, orm.ErrEmptyTable) {
			t.Errorf("Expected ErrEmptyTable, got %v", err)
		}
	})

	// 6. Test Validation Error (Create)
	t.Run("Validation Error Create", func(t *testing.T) {
		db := orm.New(&MockExecutor{}, &MockCompiler{})
		model := &MockModel{
			Table: "users",
			Cols:  []string{"col1"},
			Vals:  []any{1, 2}, // Mismatch
		}

		err := db.Create(model)
		if err == nil {
			t.Error("Expected error, got nil")
		} else if !strings.Contains(err.Error(), orm.ErrValidation.Error()) {
			t.Errorf("Expected error containing '%s', got '%v'", orm.ErrValidation.Error(), err)
		}
	})

	// 7. Test Validation Error (Update)
	t.Run("Validation Error Update", func(t *testing.T) {
		db := orm.New(&MockExecutor{}, &MockCompiler{})
		model := &MockModel{
			Table: "users",
			Cols:  []string{"col1"},
			Vals:  []any{1, 2}, // Mismatch
		}

		err := db.Update(model)
		if err == nil {
			t.Error("Expected error, got nil")
		} else if !strings.Contains(err.Error(), orm.ErrValidation.Error()) {
			t.Errorf("Expected error containing '%s', got '%v'", orm.ErrValidation.Error(), err)
		}
	})

	// Test Validation Error (Delete)
	t.Run("Validation Error Delete", func(t *testing.T) {
		db := orm.New(&MockExecutor{}, &MockCompiler{})
		model := &MockModel{Table: ""} // Empty table

		err := db.Delete(model)
		if !errors.Is(err, orm.ErrEmptyTable) {
			t.Errorf("Expected ErrEmptyTable, got %v", err)
		}
	})

	// 8. Test Empty Table Error
	t.Run("Empty Table Error", func(t *testing.T) {
		db := orm.New(&MockExecutor{}, &MockCompiler{})
		model := &MockModel{Table: ""}

		err := db.Create(model)
		if !errors.Is(err, orm.ErrEmptyTable) {
			t.Errorf("Expected ErrEmptyTable, got %v", err)
		}
	})

	// 9. Test Or Condition
	t.Run("Or Condition", func(t *testing.T) {
		c := orm.Eq("a", 1)
		orC := orm.Or(c)

		if orC.Logic() != "OR" {
			t.Errorf("Expected Logic OR, got %s", orC.Logic())
		}
	})

	// 10. Test Transaction Support
	t.Run("Transaction", func(t *testing.T) {
		mockTxBound := &MockTxBoundExecutor{}
		mockTxExec := &MockTxExecutor{Bound: mockTxBound}
		mockCompiler := &MockCompiler{}
		db := orm.New(mockTxExec, mockCompiler)

		err := db.Tx(func(tx *orm.DB) error {
			// Perform operations inside tx
			return nil
		})

		if err != nil {
			t.Fatalf("Tx failed: %v", err)
		}

		if !mockTxBound.CommitCalled {
			t.Error("Expected Commit to be called")
		}
		if mockTxBound.RollbackCalled {
			t.Error("Expected Rollback NOT to be called")
		}
	})

	// 11. Test Transaction Rollback
	t.Run("Transaction Rollback", func(t *testing.T) {
		mockTxBound := &MockTxBoundExecutor{}
		mockTxExec := &MockTxExecutor{Bound: mockTxBound}
		mockCompiler := &MockCompiler{}
		db := orm.New(mockTxExec, mockCompiler)

		expectedErr := errors.New("oops")
		err := db.Tx(func(tx *orm.DB) error {
			return expectedErr
		})

		if !errors.Is(err, expectedErr) {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}

		if mockTxBound.CommitCalled {
			t.Error("Expected Commit NOT to be called")
		}
		if !mockTxBound.RollbackCalled {
			t.Error("Expected Rollback to be called")
		}
	})

	// Test Transaction Begin Error
	t.Run("Transaction Begin Error", func(t *testing.T) {
		mockTxExec := &MockTxExecutor{BeginTxErr: errors.New("begin error")}
		db := orm.New(mockTxExec, &MockCompiler{})

		err := db.Tx(func(tx *orm.DB) error {
			return nil
		})

		if err == nil || err.Error() != "begin error" {
			t.Errorf("Expected 'begin error', got %v", err)
		}
	})

	// 12. Test No Transaction Support
	t.Run("No Tx Support", func(t *testing.T) {
		db := orm.New(&MockExecutor{}, &MockCompiler{}) // Not a TxExecutor
		err := db.Tx(func(tx *orm.DB) error { return nil })
		if !errors.Is(err, orm.ErrNoTxSupport) {
			t.Errorf("Expected ErrNoTxSupport, got %v", err)
		}
	})

	// 13. Test Condition Helpers
	t.Run("Condition Helpers", func(t *testing.T) {
		tests := []struct {
			name     string
			cond     orm.Condition
			expected string
			val      any
		}{
			{"Neq", orm.Neq("a", 1), "!=", 1},
			{"Gte", orm.Gte("b", 2), ">=", 2},
			{"Lt", orm.Lt("c", 3), "<", 3},
			{"Lte", orm.Lte("d", 4), "<=", 4},
			{"Like", orm.Like("e", "%test%"), "LIKE", "%test%"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				if tc.cond.Operator() != tc.expected {
					t.Errorf("Expected operator %s, got %s", tc.expected, tc.cond.Operator())
				}
				if tc.cond.Value() != tc.val {
					t.Errorf("Expected value %v, got %v", tc.val, tc.cond.Value())
				}
				if tc.cond.Logic() != "AND" {
					t.Errorf("Expected default logic AND, got %s", tc.cond.Logic())
				}
			})
		}
	})

	// 14. Test Condition and Order Getters
	t.Run("Getters", func(t *testing.T) {
		// Condition Getters
		c := orm.Eq("field", "val")
		if c.Field() != "field" {
			t.Errorf("Expected Field 'field', got '%s'", c.Field())
		}
		if c.Operator() != "=" {
			t.Errorf("Expected Operator '=', got '%s'", c.Operator())
		}
		if c.Value() != "val" {
			t.Errorf("Expected Value 'val', got '%v'", c.Value())
		}
		if c.Logic() != "AND" {
			t.Errorf("Expected Logic 'AND', got '%s'", c.Logic())
		}

		// Order Getters
		mockCompiler := &MockCompiler{}
		mockExec := &MockExecutor{}
		db := orm.New(mockExec, mockCompiler)
		model := &MockModel{Table: "users"}
		mockExec.ReturnQueryRow = &MockScanner{}

		db.Query(model).OrderBy("col", "ASC").ReadOne()

		if len(mockCompiler.LastQuery.OrderBy) != 1 {
			t.Fatalf("Expected 1 OrderBy, got %d", len(mockCompiler.LastQuery.OrderBy))
		}
		o := mockCompiler.LastQuery.OrderBy[0]

		if o.Column() != "col" {
			t.Errorf("Expected Column 'col', got '%s'", o.Column())
		}
		if o.Dir() != "ASC" {
			t.Errorf("Expected Dir 'ASC', got '%s'", o.Dir())
		}
	})

	// 15. Test Builder Chain (Offset, GroupBy, Limit)
	t.Run("Builder Chain", func(t *testing.T) {
		mockCompiler := &MockCompiler{}
		mockExec := &MockExecutor{}
		db := orm.New(mockExec, mockCompiler)
		model := &MockModel{Table: "users"}
		mockExec.ReturnQueryRow = &MockScanner{}

		// Test Offset and GroupBy
		db.Query(model).
			Offset(10).
			GroupBy("a", "b").
			ReadOne()

		if mockCompiler.LastQuery.Offset != 10 {
			t.Errorf("Expected Offset 10, got %d", mockCompiler.LastQuery.Offset)
		}
		if len(mockCompiler.LastQuery.GroupBy) != 2 {
			t.Errorf("Expected 2 GroupBy cols, got %d", len(mockCompiler.LastQuery.GroupBy))
		}

		// Test Limit with ReadAll
		mockExec.ReturnQueryRows = &MockRows{Count: 0}
		db.Query(model).
			Limit(5).
			ReadAll(func() orm.Model { return nil }, func(orm.Model) {})

		if mockCompiler.LastQuery.Limit != 5 {
			t.Errorf("Expected Limit 5, got %d", mockCompiler.LastQuery.Limit)
		}
	})

	// 16. Errors coverage
	t.Run("Errors", func(t *testing.T) {
		model := &MockModel{Table: "t", Cols: []string{"a"}, Vals: []any{1}}

		// Create Plan Error
		db1 := orm.New(&MockExecutor{}, &MockCompiler{ReturnErr: errors.New("plan err")})
		if err := db1.Create(model); err == nil || err.Error() != "plan err" {
			t.Errorf("Expected plan err, got %v", err)
		}

		// Create Exec Error
		db2 := orm.New(&MockExecutor{ReturnExecErr: errors.New("exec err")}, &MockCompiler{})
		if err := db2.Create(model); err == nil || err.Error() != "exec err" {
			t.Errorf("Expected exec err, got %v", err)
		}

		// Update Plan Error
		if err := db1.Update(model); err == nil || err.Error() != "plan err" {
			t.Errorf("Expected plan err, got %v", err)
		}
		// Update Exec Error
		if err := db2.Update(model); err == nil || err.Error() != "exec err" {
			t.Errorf("Expected exec err, got %v", err)
		}

		// Delete Plan Error
		if err := db1.Delete(model); err == nil || err.Error() != "plan err" {
			t.Errorf("Expected plan err, got %v", err)
		}
		// Delete Exec Error
		if err := db2.Delete(model); err == nil || err.Error() != "exec err" {
			t.Errorf("Expected exec err, got %v", err)
		}

		// ReadOne Plan Error
		if err := db1.Query(model).ReadOne(); err == nil || err.Error() != "plan err" {
			t.Errorf("Expected plan err, got %v", err)
		}
		// ReadOne Scan Error
		db3 := orm.New(&MockExecutor{ReturnQueryRow: &MockScanner{ScanErr: errors.New("scan err")}}, &MockCompiler{})
		if err := db3.Query(model).ReadOne(); err == nil || err.Error() != "scan err" {
			t.Errorf("Expected scan err, got %v", err)
		}

		// ReadAll Plan Error
		if err := db1.Query(model).ReadAll(nil, nil); err == nil || err.Error() != "plan err" {
			t.Errorf("Expected plan err, got %v", err)
		}
		// ReadAll Query Error
		db4 := orm.New(&MockExecutor{ReturnQueryErr: errors.New("query err")}, &MockCompiler{})
		if err := db4.Query(model).ReadAll(nil, nil); err == nil || err.Error() != "query err" {
			t.Errorf("Expected query err, got %v", err)
		}
		// ReadAll Scan Error
		db5 := orm.New(&MockExecutor{ReturnQueryRows: &MockRows{Count: 1, ScanErr: errors.New("scan err")}}, &MockCompiler{})
		f := func() orm.Model { return &MockModel{} }
		e := func(m orm.Model) {}
		if err := db5.Query(model).ReadAll(f, e); err == nil || err.Error() != "scan err" {
			t.Errorf("Expected scan err, got %v", err)
		}
		// ReadAll Rows Err
		db6 := orm.New(&MockExecutor{ReturnQueryRows: &MockRows{Count: 0, ErrVal: errors.New("rows err")}}, &MockCompiler{})
		if err := db6.Query(model).ReadAll(f, e); err == nil || err.Error() != "rows err" {
			t.Errorf("Expected rows err, got %v", err)
		}
	})
}
