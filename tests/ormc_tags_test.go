package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tinywasm/orm"
)

func TestRewriteModelTags(t *testing.T) {
	// Create a temp model file
	tmpDir := t.TempDir()
	modelPath := filepath.Join(tmpDir, "model.go")

	input := `package test

type User struct {
	ID    int    ` + "`" + `db:"pk" json:"id"` + "`" + `
	Name  string ` + "`" + `json:"name" validate:"required"` + "`" + `
	Email string ` + "`" + `db:"unique" form:"email" json:"email,omitempty" validate:"email"` + "`" + `
	Age   int    ` + "`" + `json:"-"` + "`" + `
}
`
	err := os.WriteFile(modelPath, []byte(input), 0644)
	if err != nil {
		t.Fatal(err)
	}

	o := orm.NewOrmc()
	err = o.RewriteModelTags(modelPath)
	if err != nil {
		t.Fatalf("RewriteModelTags failed: %v", err)
	}

	output, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}

	outStr := string(output)

	// Verify ID: json:"id" removed
	if strings.Contains(outStr, `json:"id"`) {
		t.Errorf("Expected json:\"id\" to be removed, got: %s", outStr)
	}
	// Verify Name: json:"name" removed, validate:"required" removed
	if strings.Contains(outStr, `json:"name"`) || strings.Contains(outStr, `validate:"required"`) {
		t.Errorf("Expected json:\"name\" and validate:\"required\" to be removed, got: %s", outStr)
	}
	// Verify Email: form, validate removed, json rewritten to ,omitempty
	if strings.Contains(outStr, `form:"email"`) || strings.Contains(outStr, `validate:"email"`) {
		t.Errorf("Expected form and validate to be removed, got: %s", outStr)
	}
	if !strings.Contains(outStr, `json:",omitempty"`) {
		t.Errorf("Expected json:\",omitempty\" to be present, got: %s", outStr)
	}
	// Verify Age: json:"-" preserved
	if !strings.Contains(outStr, `json:"-"`) {
		t.Errorf("Expected json:\"-\" to be preserved, got: %s", outStr)
	}
}
