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
		mockPlanner := &MockPlanner{}
		mockExec := &MockExecutor{}
		db := orm.New(mockExec, mockPlanner)

		model := &MockModel{
			Table: "users",
			Cols:  []string{"name", "age"},
			Vals:  []any{"Alice", 30},
		}

		err := db.Create(model)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if mockPlanner.LastQuery.Action != orm.ActionCreate {
			t.Errorf("Expected ActionCreate, got %v", mockPlanner.LastQuery.Action)
		}
		if mockPlanner.LastQuery.Table != "users" {
			t.Errorf("Expected table 'users', got '%s'", mockPlanner.LastQuery.Table)
		}
		if len(mockPlanner.LastQuery.Columns) != 2 {
			t.Errorf("Expected 2 columns, got %d", len(mockPlanner.LastQuery.Columns))
		}
	})

	// 2. Test Update with Conditions
	t.Run("Update", func(t *testing.T) {
		mockPlanner := &MockPlanner{}
		mockExec := &MockExecutor{}
		db := orm.New(mockExec, mockPlanner)

		model := &MockModel{
			Table: "users",
			Cols:  []string{"age"},
			Vals:  []any{31},
		}

		err := db.Update(model, orm.Eq("name", "Alice"))
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		if mockPlanner.LastQuery.Action != orm.ActionUpdate {
			t.Errorf("Expected ActionUpdate, got %v", mockPlanner.LastQuery.Action)
		}
		if len(mockPlanner.LastQuery.Conditions) != 1 {
			t.Errorf("Expected 1 condition, got %d", len(mockPlanner.LastQuery.Conditions))
		}
		if mockPlanner.LastQuery.Conditions[0].Field() != "name" {
			t.Errorf("Expected condition field 'name', got '%s'", mockPlanner.LastQuery.Conditions[0].Field())
		}
	})

	// 3. Test Delete
	t.Run("Delete", func(t *testing.T) {
		mockPlanner := &MockPlanner{}
		mockExec := &MockExecutor{}
		db := orm.New(mockExec, mockPlanner)

		model := &MockModel{Table: "users"}

		err := db.Delete(model, orm.Gt("age", 100))
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		if mockPlanner.LastQuery.Action != orm.ActionDelete {
			t.Errorf("Expected ActionDelete, got %v", mockPlanner.LastQuery.Action)
		}
		if len(mockPlanner.LastQuery.Conditions) != 1 {
			t.Errorf("Expected 1 condition, got %d", len(mockPlanner.LastQuery.Conditions))
		}
	})

	// 4. Test Query Chain (ReadOne)
	t.Run("ReadOne", func(t *testing.T) {
		mockPlanner := &MockPlanner{}
		mockExec := &MockExecutor{}
		db := orm.New(mockExec, mockPlanner)

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

		if mockPlanner.LastQuery.Action != orm.ActionReadOne {
			t.Errorf("Expected ActionReadOne, got %v", mockPlanner.LastQuery.Action)
		}
		if mockPlanner.LastQuery.Limit != 1 {
			t.Errorf("Expected Limit 1, got %d", mockPlanner.LastQuery.Limit)
		}
		if len(mockPlanner.LastQuery.OrderBy) != 1 {
			t.Errorf("Expected 1 OrderBy, got %d", len(mockPlanner.LastQuery.OrderBy))
		}
	})

	// Test ReadOne Validation Error
	t.Run("ReadOne Validation Error", func(t *testing.T) {
		mockPlanner := &MockPlanner{}
		mockExec := &MockExecutor{}
		db := orm.New(mockExec, mockPlanner)
		model := &MockModel{Table: ""} // Empty table

		err := db.Query(model).ReadOne()
		if !errors.Is(err, orm.ErrEmptyTable) {
			t.Errorf("Expected ErrEmptyTable, got %v", err)
		}
	})

	// 5. Test ReadAll
	t.Run("ReadAll", func(t *testing.T) {
		mockPlanner := &MockPlanner{}
		mockExec := &MockExecutor{}
		db := orm.New(mockExec, mockPlanner)

		model := &MockModel{Table: "users"}

		// Simulate 2 rows
		mockRows := &MockRows{Count: 2}
		mockExec.ReturnQueryRows = mockRows

		factoryCalled := 0
		eachCalled := 0
		factory := func() orm.Model {
			factoryCalled++
			return &MockModel{}
		}
		each := func(m orm.Model) {
			eachCalled++
		}

		err := db.Query(model).ReadAll(factory, each)
		if err != nil {
			t.Fatalf("ReadAll failed: %v", err)
		}

		if mockPlanner.LastQuery.Action != orm.ActionReadAll {
			t.Errorf("Expected ActionReadAll, got %v", mockPlanner.LastQuery.Action)
		}
		if factoryCalled != 2 {
			t.Errorf("Expected factory called 2 times, got %d", factoryCalled)
		}
		if eachCalled != 2 {
			t.Errorf("Expected each called 2 times, got %d", eachCalled)
		}
	})

	// Test ReadAll Validation Error
	t.Run("ReadAll Validation Error", func(t *testing.T) {
		mockPlanner := &MockPlanner{}
		mockExec := &MockExecutor{}
		db := orm.New(mockExec, mockPlanner)
		model := &MockModel{Table: ""} // Empty table

		err := db.Query(model).ReadAll(nil, nil)
		if !errors.Is(err, orm.ErrEmptyTable) {
			t.Errorf("Expected ErrEmptyTable, got %v", err)
		}
	})

	// 6. Test Validation Error (Create)
	t.Run("Validation Error Create", func(t *testing.T) {
		db := orm.New(&MockExecutor{}, &MockPlanner{})
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
		db := orm.New(&MockExecutor{}, &MockPlanner{})
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
		db := orm.New(&MockExecutor{}, &MockPlanner{})
		model := &MockModel{Table: ""} // Empty table

		err := db.Delete(model)
		if !errors.Is(err, orm.ErrEmptyTable) {
			t.Errorf("Expected ErrEmptyTable, got %v", err)
		}
	})

	// 8. Test Empty Table Error
	t.Run("Empty Table Error", func(t *testing.T) {
		db := orm.New(&MockExecutor{}, &MockPlanner{})
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
		mockPlanner := &MockPlanner{}
		db := orm.New(mockTxExec, mockPlanner)

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
		mockPlanner := &MockPlanner{}
		db := orm.New(mockTxExec, mockPlanner)

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
		db := orm.New(mockTxExec, &MockPlanner{})

		err := db.Tx(func(tx *orm.DB) error {
			return nil
		})

		if err == nil || err.Error() != "begin error" {
			t.Errorf("Expected 'begin error', got %v", err)
		}
	})

	// 12. Test No Transaction Support
	t.Run("No Tx Support", func(t *testing.T) {
		db := orm.New(&MockExecutor{}, &MockPlanner{}) // Not a TxExecutor
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
		mockPlanner := &MockPlanner{}
		mockExec := &MockExecutor{}
		db := orm.New(mockExec, mockPlanner)
		model := &MockModel{Table: "users"}
		mockExec.ReturnQueryRow = &MockScanner{}

		db.Query(model).OrderBy("col", "ASC").ReadOne()

		if len(mockPlanner.LastQuery.OrderBy) != 1 {
			t.Fatalf("Expected 1 OrderBy, got %d", len(mockPlanner.LastQuery.OrderBy))
		}
		o := mockPlanner.LastQuery.OrderBy[0]

		if o.Column() != "col" {
			t.Errorf("Expected Column 'col', got '%s'", o.Column())
		}
		if o.Dir() != "ASC" {
			t.Errorf("Expected Dir 'ASC', got '%s'", o.Dir())
		}
	})

	// 15. Test Builder Chain (Offset, GroupBy, Limit)
	t.Run("Builder Chain", func(t *testing.T) {
		mockPlanner := &MockPlanner{}
		mockExec := &MockExecutor{}
		db := orm.New(mockExec, mockPlanner)
		model := &MockModel{Table: "users"}
		mockExec.ReturnQueryRow = &MockScanner{}

		// Test Offset and GroupBy
		db.Query(model).
			Offset(10).
			GroupBy("a", "b").
			ReadOne()

		if mockPlanner.LastQuery.Offset != 10 {
			t.Errorf("Expected Offset 10, got %d", mockPlanner.LastQuery.Offset)
		}
		if len(mockPlanner.LastQuery.GroupBy) != 2 {
			t.Errorf("Expected 2 GroupBy cols, got %d", len(mockPlanner.LastQuery.GroupBy))
		}

		// Test Limit with ReadAll
		mockExec.ReturnQueryRows = &MockRows{Count: 0}
		db.Query(model).
			Limit(5).
			ReadAll(func() orm.Model { return nil }, func(orm.Model) {})

		if mockPlanner.LastQuery.Limit != 5 {
			t.Errorf("Expected Limit 5, got %d", mockPlanner.LastQuery.Limit)
		}
	})

	// 16. Errors coverage
	t.Run("Errors", func(t *testing.T) {
		model := &MockModel{Table: "t", Cols: []string{"a"}, Vals: []any{1}}

		// Create Plan Error
		db1 := orm.New(&MockExecutor{}, &MockPlanner{ReturnErr: errors.New("plan err")})
		if err := db1.Create(model); err == nil || err.Error() != "plan err" {
			t.Errorf("Expected plan err, got %v", err)
		}

		// Create Exec Error
		db2 := orm.New(&MockExecutor{ReturnExecErr: errors.New("exec err")}, &MockPlanner{})
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
		db3 := orm.New(&MockExecutor{ReturnQueryRow: &MockScanner{ScanErr: errors.New("scan err")}}, &MockPlanner{})
		if err := db3.Query(model).ReadOne(); err == nil || err.Error() != "scan err" {
			t.Errorf("Expected scan err, got %v", err)
		}

		// ReadAll Plan Error
		if err := db1.Query(model).ReadAll(nil, nil); err == nil || err.Error() != "plan err" {
			t.Errorf("Expected plan err, got %v", err)
		}
		// ReadAll Query Error
		db4 := orm.New(&MockExecutor{ReturnQueryErr: errors.New("query err")}, &MockPlanner{})
		if err := db4.Query(model).ReadAll(nil, nil); err == nil || err.Error() != "query err" {
			t.Errorf("Expected query err, got %v", err)
		}
		// ReadAll Scan Error
		db5 := orm.New(&MockExecutor{ReturnQueryRows: &MockRows{Count: 1, ScanErr: errors.New("scan err")}}, &MockPlanner{})
		f := func() orm.Model { return &MockModel{} }
		e := func(m orm.Model) {}
		if err := db5.Query(model).ReadAll(f, e); err == nil || err.Error() != "scan err" {
			t.Errorf("Expected scan err, got %v", err)
		}
		// ReadAll Rows Err
		db6 := orm.New(&MockExecutor{ReturnQueryRows: &MockRows{Count: 0, ErrVal: errors.New("rows err")}}, &MockPlanner{})
		if err := db6.Query(model).ReadAll(f, e); err == nil || err.Error() != "rows err" {
			t.Errorf("Expected rows err, got %v", err)
		}
	})
}
