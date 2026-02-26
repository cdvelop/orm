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
		mockAdapter := &MockAdapter{}
		db := orm.New(mockAdapter)

		model := &MockModel{
			Table: "users",
			Cols:  []string{"name", "age"},
			Vals:  []any{"Alice", 30},
		}

		err := db.Create(model)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if mockAdapter.LastQuery.Action != orm.ActionCreate {
			t.Errorf("Expected ActionCreate, got %v", mockAdapter.LastQuery.Action)
		}
		if mockAdapter.LastQuery.Table != "users" {
			t.Errorf("Expected table 'users', got '%s'", mockAdapter.LastQuery.Table)
		}
		if len(mockAdapter.LastQuery.Columns) != 2 {
			t.Errorf("Expected 2 columns, got %d", len(mockAdapter.LastQuery.Columns))
		}
	})

	// 2. Test Update with Conditions
	t.Run("Update", func(t *testing.T) {
		mockAdapter := &MockAdapter{}
		db := orm.New(mockAdapter)

		model := &MockModel{
			Table: "users",
			Cols:  []string{"age"},
			Vals:  []any{31},
		}

		err := db.Update(model, orm.Eq("name", "Alice"))
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		if mockAdapter.LastQuery.Action != orm.ActionUpdate {
			t.Errorf("Expected ActionUpdate, got %v", mockAdapter.LastQuery.Action)
		}
		if len(mockAdapter.LastQuery.Conditions) != 1 {
			t.Errorf("Expected 1 condition, got %d", len(mockAdapter.LastQuery.Conditions))
		}
		if mockAdapter.LastQuery.Conditions[0].Field() != "name" {
			t.Errorf("Expected condition field 'name', got '%s'", mockAdapter.LastQuery.Conditions[0].Field())
		}
	})

	// 3. Test Delete
	t.Run("Delete", func(t *testing.T) {
		mockAdapter := &MockAdapter{}
		db := orm.New(mockAdapter)

		model := &MockModel{Table: "users"}

		err := db.Delete(model, orm.Gt("age", 100))
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		if mockAdapter.LastQuery.Action != orm.ActionDelete {
			t.Errorf("Expected ActionDelete, got %v", mockAdapter.LastQuery.Action)
		}
		if len(mockAdapter.LastQuery.Conditions) != 1 {
			t.Errorf("Expected 1 condition, got %d", len(mockAdapter.LastQuery.Conditions))
		}
	})

	// 4. Test Query Chain (ReadOne)
	t.Run("ReadOne", func(t *testing.T) {
		mockAdapter := &MockAdapter{}
		db := orm.New(mockAdapter)

		model := &MockModel{Table: "users"}

		err := db.Query(model).
			Where(orm.Eq("id", 1)).
			OrderBy("created_at", "DESC").
			ReadOne()

		if err != nil {
			t.Fatalf("ReadOne failed: %v", err)
		}

		if mockAdapter.LastQuery.Action != orm.ActionReadOne {
			t.Errorf("Expected ActionReadOne, got %v", mockAdapter.LastQuery.Action)
		}
		if mockAdapter.LastQuery.Limit != 1 {
			t.Errorf("Expected Limit 1, got %d", mockAdapter.LastQuery.Limit)
		}
		if len(mockAdapter.LastQuery.OrderBy) != 1 {
			t.Errorf("Expected 1 OrderBy, got %d", len(mockAdapter.LastQuery.OrderBy))
		}
	})

	// Test ReadOne Validation Error
	t.Run("ReadOne Validation Error", func(t *testing.T) {
		mockAdapter := &MockAdapter{}
		db := orm.New(mockAdapter)
		model := &MockModel{Table: ""} // Empty table

		err := db.Query(model).ReadOne()
		if !errors.Is(err, orm.ErrEmptyTable) {
			t.Errorf("Expected ErrEmptyTable, got %v", err)
		}
	})

	// 5. Test ReadAll
	t.Run("ReadAll", func(t *testing.T) {
		mockAdapter := &MockAdapter{}
		db := orm.New(mockAdapter)

		model := &MockModel{Table: "users"}

		factory := func() orm.Model { return &MockModel{} }
		each := func(m orm.Model) {}

		err := db.Query(model).ReadAll(factory, each)
		if err != nil {
			t.Fatalf("ReadAll failed: %v", err)
		}

		if mockAdapter.LastQuery.Action != orm.ActionReadAll {
			t.Errorf("Expected ActionReadAll, got %v", mockAdapter.LastQuery.Action)
		}
		if mockAdapter.LastFactory == nil {
			t.Error("Expected factory to be passed to adapter")
		}
		if mockAdapter.LastEach == nil {
			t.Error("Expected each callback to be passed to adapter")
		}
	})

	// Test ReadAll Validation Error
	t.Run("ReadAll Validation Error", func(t *testing.T) {
		mockAdapter := &MockAdapter{}
		db := orm.New(mockAdapter)
		model := &MockModel{Table: ""} // Empty table

		err := db.Query(model).ReadAll(nil, nil)
		if !errors.Is(err, orm.ErrEmptyTable) {
			t.Errorf("Expected ErrEmptyTable, got %v", err)
		}
	})

	// 6. Test Validation Error (Create)
	t.Run("Validation Error Create", func(t *testing.T) {
		db := orm.New(&MockAdapter{})
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
		db := orm.New(&MockAdapter{})
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
		db := orm.New(&MockAdapter{})
		model := &MockModel{Table: ""} // Empty table

		err := db.Delete(model)
		if !errors.Is(err, orm.ErrEmptyTable) {
			t.Errorf("Expected ErrEmptyTable, got %v", err)
		}
	})

	// 8. Test Empty Table Error
	t.Run("Empty Table Error", func(t *testing.T) {
		db := orm.New(&MockAdapter{})
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
		mockTxBound := &MockTxBound{}
		mockTxAdapter := &MockTxAdapter{Bound: mockTxBound}
		db := orm.New(mockTxAdapter)

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
		mockTxBound := &MockTxBound{}
		mockTxAdapter := &MockTxAdapter{Bound: mockTxBound}
		db := orm.New(mockTxAdapter)

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
		mockTxAdapter := &MockTxAdapter{BeginTxErr: errors.New("begin error")}
		db := orm.New(mockTxAdapter)

		err := db.Tx(func(tx *orm.DB) error {
			return nil
		})

		if err == nil || err.Error() != "begin error" {
			t.Errorf("Expected 'begin error', got %v", err)
		}
	})

	// 12. Test No Transaction Support
	t.Run("No Tx Support", func(t *testing.T) {
		db := orm.New(&MockAdapter{}) // Not a TxAdapter
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

		// Order Getters (using MockAdapter to capture Order object since we can't construct it directly)
		mockAdapter := &MockAdapter{}
		db := orm.New(mockAdapter)
		model := &MockModel{Table: "users"}

		db.Query(model).OrderBy("col", "ASC").ReadOne()

		if len(mockAdapter.LastQuery.OrderBy) != 1 {
			t.Fatalf("Expected 1 OrderBy, got %d", len(mockAdapter.LastQuery.OrderBy))
		}
		o := mockAdapter.LastQuery.OrderBy[0]

		if o.Column() != "col" {
			t.Errorf("Expected Column 'col', got '%s'", o.Column())
		}
		if o.Dir() != "ASC" {
			t.Errorf("Expected Dir 'ASC', got '%s'", o.Dir())
		}
	})

	// 15. Test Builder Chain (Offset, GroupBy, Limit)
	t.Run("Builder Chain", func(t *testing.T) {
		mockAdapter := &MockAdapter{}
		db := orm.New(mockAdapter)
		model := &MockModel{Table: "users"}

		// Test Offset and GroupBy
		db.Query(model).
			Offset(10).
			GroupBy("a", "b").
			ReadOne()

		if mockAdapter.LastQuery.Offset != 10 {
			t.Errorf("Expected Offset 10, got %d", mockAdapter.LastQuery.Offset)
		}
		if len(mockAdapter.LastQuery.GroupBy) != 2 {
			t.Errorf("Expected 2 GroupBy cols, got %d", len(mockAdapter.LastQuery.GroupBy))
		}

		// Test Limit with ReadAll
		db.Query(model).
			Limit(5).
			ReadAll(nil, nil)

		if mockAdapter.LastQuery.Limit != 5 {
			t.Errorf("Expected Limit 5, got %d", mockAdapter.LastQuery.Limit)
		}
	})
}
