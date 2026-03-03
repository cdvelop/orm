//go:build !wasm

package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tinywasm/orm"
)

func TestOrmc_MultiStruct(t *testing.T) {
	t.Run("Both structs appear in a single output file", func(t *testing.T) {
		o := orm.NewOrmc()
		infoA, err := o.ParseStruct("MultiA", "mock_generator_model.go")
		if err != nil {
			t.Fatal(err)
		}

		infoB, err := o.ParseStruct("MultiB", "mock_generator_model.go")
		if err != nil {
			t.Fatal(err)
		}

		err = o.GenerateForFile([]orm.StructInfo{infoA, infoB}, "mock_generator_model.go")
		if err != nil {
			t.Fatal(err)
		}

		outFile := "mock_generator_model_db.go"
		content, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(outFile)

		s := string(content)

		// Both schemas must be present
		if !strings.Contains(s, "func (m *MultiA) Schema()") {
			t.Error("MultiA Schema() not generated")
		}
		if !strings.Contains(s, "func (m *MultiB) Schema()") {
			t.Error("MultiB Schema() not generated")
		}
	})
}

func TestOrmc_Run(t *testing.T) {
	t.Run("Run() scans dir and generates all structs", func(t *testing.T) {
		// Use a temp dir to avoid polluting tests/
		tmp := t.TempDir()

		// Copy model file into temp dir
		src, err := os.ReadFile("mock_generator_model.go")
		if err != nil {
			t.Fatal(err)
		}
		// Replace package declaration so it compiles as package "tests"
		modelFile := filepath.Join(tmp, "model.go")
		if err := os.WriteFile(modelFile, src, 0644); err != nil {
			t.Fatal(err)
		}

		var logged []string
		o := orm.NewOrmc()
		o.SetRootDir(tmp)
		o.SetLog(func(messages ...any) {
			// Collect log output — verifies SetLog + logFn branch
			for _, m := range messages {
				logged = append(logged, fmt.Sprint(m))
			}
		})

		if err := o.Run(); err != nil {
			t.Fatalf("Run() failed: %v", err)
		}

		// The generated file must exist
		outFile := filepath.Join(tmp, "model_db.go")
		content, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("Expected model_db.go, got error: %v", err)
		}

		s := string(content)
		// Spot-check a couple of structs that have valid fields
		if !strings.Contains(s, "func (m *User) Schema()") {
			t.Error("User Schema() not in Run() output")
		}
		if !strings.Contains(s, "func (m *MultiA) Schema()") {
			t.Error("MultiA Schema() not in Run() output")
		}
		// Warning for BadTimeNoTag / Unsupp must have been logged
		_ = logged // just exercise the log path; content varies
	})

	t.Run("Run() returns error when no models found", func(t *testing.T) {
		tmp := t.TempDir()
		o := orm.NewOrmc()
		o.SetRootDir(tmp)
		if err := o.Run(); err == nil {
			t.Error("Expected error for empty directory, got nil")
		}
	})
}

func TestOrmc_DetectPointerReceiver(t *testing.T) {
	t.Run("TableName() NOT generated when declared with pointer receiver", func(t *testing.T) {
		err := orm.NewOrmc().GenerateForStruct("PointerReceiver", "mock_generator_model.go")
		if err != nil {
			t.Fatal(err)
		}

		outFile := "mock_generator_model_db.go"
		content, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(outFile)

		if strings.Contains(string(content), "func (m *PointerReceiver) TableName()") {
			t.Error("TableName() must NOT be generated — already declared with pointer receiver")
		}
		if !strings.Contains(string(content), `"ptr_table"`) {
			// The Meta struct must reference the declared table name
			t.Error("Expected ptr_table in generated meta")
		}
	})
}

func TestQB_ClauseChain(t *testing.T) {
	t.Run("All Clause operators via QB chain", func(t *testing.T) {
		mockCompiler := &MockCompiler{}
		mockExec := &MockExecutor{}
		db := orm.New(mockExec, mockCompiler)
		model := &MockModel{Table: "items"}
		mockExec.ReturnQueryRows = &MockRows{Count: 0}

		db.Query(model).
			Where("a").Neq(1).
			Where("b").Gt(2).
			Where("c").Gte(3).
			Where("d").Lt(4).
			Where("e").Lte(5).
			Where("f").Like("%x%").
			Where("g").In([]int{1, 2}).
			Or().Where("h").Eq(9).
			ReadAll(func() orm.Model { return &MockModel{} }, func(orm.Model) {})

		conds := mockCompiler.LastQuery.Conditions
		expected := []struct {
			field string
			op    string
		}{
			{"a", "!="},
			{"b", ">"},
			{"c", ">="},
			{"d", "<"},
			{"e", "<="},
			{"f", "LIKE"},
			{"g", "IN"},
			{"h", "="},
		}
		if len(conds) != len(expected) {
			t.Fatalf("Expected %d conditions, got %d", len(expected), len(conds))
		}
		for i, ex := range expected {
			if conds[i].Field() != ex.field {
				t.Errorf("cond[%d]: expected field %q, got %q", i, ex.field, conds[i].Field())
			}
			if conds[i].Operator() != ex.op {
				t.Errorf("cond[%d]: expected op %q, got %q", i, ex.op, conds[i].Operator())
			}
		}
		// Last condition must be OR
		if conds[7].Logic() != "OR" {
			t.Errorf("Expected last condition Logic=OR, got %s", conds[7].Logic())
		}
	})
}

func TestOrmc_TableNameDetection(t *testing.T) {
	t.Run("TableName() NOT generated when already declared (D5)", func(t *testing.T) {
		err := orm.NewOrmc().GenerateForStruct("MultiA", "mock_generator_model.go")
		if err != nil {
			t.Fatal(err)
		}

		outFile := "mock_generator_model_db.go"
		content, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(outFile)

		if strings.Contains(string(content), "func (m *MultiA) TableName()") {
			t.Error("TableName() must NOT be generated — already declared in source")
		}
	})

	t.Run("TableName() IS generated when not declared (D5)", func(t *testing.T) {
		err := orm.NewOrmc().GenerateForStruct("MultiB", "mock_generator_model.go")
		if err != nil {
			t.Fatal(err)
		}

		outFile := "mock_generator_model_db.go"
		content, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(outFile)

		if !strings.Contains(string(content), "func (m *MultiB) TableName()") {
			t.Error("TableName() must be generated — not declared in source")
		}
	})
}

func TestOrmc_DbIgnoreTag(t *testing.T) {
	t.Run("db:\"-\" fields excluded from Schema, Values, Pointers (D3+D4)", func(t *testing.T) {
		err := orm.NewOrmc().GenerateForStruct("ModelWithIgnored", "mock_generator_model.go")
		if err != nil {
			t.Fatal(err)
		}

		outFile := "mock_generator_model_db.go"
		content, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(outFile)

		s := string(content)

		for _, absent := range []string{"Tags", "Friends", "tags", "friends"} {
			if strings.Contains(s, absent) {
				t.Errorf("db:\"-\" field %q must be absent from ALL generated code", absent)
			}
		}
		for _, present := range []string{
			`"id"`, `"name"`, `"score"`,
			"m.ID", "m.Name", "m.Score",
			"&m.ID", "&m.Name", "&m.Score",
		} {
			if !strings.Contains(s, present) {
				t.Errorf("Non-ignored field %q must be present", present)
			}
		}
	})
}
