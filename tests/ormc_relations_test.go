//go:build !wasm

package tests

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/tinywasm/orm"
)

func TestOrmc_RelationLoader(t *testing.T) {
	t.Run("ResolveRelations detects FK and sets LoaderName", func(t *testing.T) {
		o := orm.NewOrmc()

		parent, err := o.ParseStruct("MockParent", "mock_generator_model.go")
		if err != nil {
			t.Fatal(err)
		}
		child, err := o.ParseStruct("MockChild", "mock_generator_model.go")
		if err != nil {
			t.Fatal(err)
		}

		all := map[string]orm.StructInfo{
			"MockParent": parent,
			"MockChild":  child,
		}
		o.ResolveRelations(all)

		if len(all["MockChild"].Relations) != 1 {
			t.Fatalf("expected 1 relation on MockChild, got %d", len(all["MockChild"].Relations))
		}
		rel := all["MockChild"].Relations[0]
		if rel.LoaderName != "ReadAllMockChildByMockParentID" {
			t.Errorf("unexpected loader name: %s", rel.LoaderName)
		}
	})

	t.Run("GenerateForFile emits relation loader", func(t *testing.T) {
		o := orm.NewOrmc()

		parent, _ := o.ParseStruct("MockParent", "mock_generator_model.go")
		child, _ := o.ParseStruct("MockChild", "mock_generator_model.go")

		all := map[string]orm.StructInfo{
			"MockParent": parent,
			"MockChild":  child,
		}
		o.ResolveRelations(all)

		err := o.GenerateForFile([]orm.StructInfo{all["MockChild"]}, "mock_generator_model.go")
		if err != nil {
			t.Fatal(err)
		}

		outFile := "mock_generator_model_db.go"
		content, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(outFile)

		if !strings.Contains(string(content), "ReadAllMockChildByMockParentID") {
			t.Error("relation loader not found in generated output")
		}
	})

	t.Run("No FK in child → warning log, no relation generated", func(t *testing.T) {
		o := orm.NewOrmc()
		var logged []string
		o.SetLog(func(msgs ...any) {
			for _, m := range msgs {
				logged = append(logged, fmt.Sprint(m))
			}
		})

		// MultiA has no FK pointing to any parent
		parent, _ := o.ParseStruct("MockParent", "mock_generator_model.go")
		noFK, _ := o.ParseStruct("MultiA", "mock_generator_model.go")

		all := map[string]orm.StructInfo{
			"MockParent": parent,
			"MultiA":     noFK,
		}
		// Patch MockParent to pretend Kids is []MultiA
		p := all["MockParent"]
		p.SliceFields = []orm.SliceFieldInfo{{Name: "Kids", ElemType: "MultiA"}}
		all["MockParent"] = p

		o.ResolveRelations(all)

		if len(all["MultiA"].Relations) != 0 {
			t.Error("expected 0 relations when no FK found")
		}
		found := false
		for _, l := range logged {
			if strings.Contains(l, "skipping") || strings.Contains(l, "no") {
				found = true
			}
		}
		if !found {
			t.Error("expected a warning log for missing FK")
		}
	})
}
