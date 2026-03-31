//go:build !wasm

package orm

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"github.com/tinywasm/fmt"
)

// RewriteModelTags performs in-place struct tag cleanup in the given file.
// It removes 'form' and 'validate' tags, and strips field names from 'json' tags.
func (o *Ormc) RewriteModelTags(path string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	type edit struct {
		start, end int
		newVal     string
	}
	var edits []edit

	ast.Inspect(node, func(n ast.Node) bool {
		field, ok := n.(*ast.Field)
		if !ok || field.Tag == nil {
			return true
		}

		tagValue := field.Tag.Value
		if !strings.HasPrefix(tagValue, "`") || !strings.HasSuffix(tagValue, "`") {
			return true
		}

		rawTag := strings.Trim(tagValue, "`")
		newTag := rewriteRawTag(rawTag)
		if newTag != rawTag {
			edits = append(edits, edit{
				start:  int(field.Tag.Pos()),
				end:    int(field.Tag.End()),
				newVal: "`" + newTag + "`",
			})
		}

		return true
	})

	if len(edits) == 0 {
		return nil
	}

	// Apply edits in reverse to keep offsets valid
	result := content
	for i := len(edits) - 1; i >= 0; i-- {
		e := edits[i]
		start := fset.Position(token.Pos(e.start)).Offset
		end := fset.Position(token.Pos(e.end)).Offset

		var buf bytes.Buffer
		buf.Write(result[:start])
		buf.WriteString(e.newVal)
		buf.Write(result[end:])
		result = buf.Bytes()
	}

	return os.WriteFile(path, result, 0644)
}

func rewriteRawTag(raw string) string {
	parts := fmt.Convert(raw).Split(" ")
	var kept []string

	for _, p := range parts {
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "form:") || strings.HasPrefix(p, "validate:") {
			continue
		}
		if strings.HasPrefix(p, "json:\"") {
			val := strings.TrimSuffix(strings.TrimPrefix(p, "json:\""), "\"")
			if val == "-" {
				kept = append(kept, p)
				continue
			}

			subParts := strings.Split(val, ",")
			hasOmit := false
			for _, sp := range subParts {
				if sp == "omitempty" {
					hasOmit = true
					break
				}
			}

			if hasOmit {
				kept = append(kept, "json:\",omitempty\"")
			}
			// if no omitempty and no -, we just drop the json tag
			continue
		}
		kept = append(kept, p)
	}

	return strings.Join(kept, " ")
}
