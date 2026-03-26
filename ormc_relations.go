//go:build !wasm

package orm

import (
	"sort"

	"github.com/tinywasm/fmt"
)

// RelationInfo describes a one-to-many relation loader to generate.
type RelationInfo struct {
	ChildStruct string // e.g. "Role"
	FKField     string // e.g. "UserID"  (Go field name)
	FKColumn    string // e.g. "user_id" (column name)
	LoaderName  string // e.g. "ReadAllRoleByUserID"
	FKFieldType string // e.g. "string", "int64"
}

// ResolveRelations (exported for testing) scans all parent SliceFields,
// finds the matching FK in the child struct, and appends RelationInfo
// to the child's entry in the map.
func (o *Ormc) ResolveRelations(all map[string]StructInfo) {
	// Sort parent names to ensure deterministic relation generation
	var parentNames []string
	for parentName := range all {
		parentNames = append(parentNames, parentName)
	}
	sort.Strings(parentNames)

	for _, parentName := range parentNames {
		parentInfo := all[parentName]
		for _, sliceField := range parentInfo.SliceFields {
			childStructName := sliceField.ElemType
			childInfo, ok := all[childStructName]
			if !ok {
				o.log(fmt.Sprintf("Warning: relation field %s.%s points to unknown struct %s; skipping", parentName, sliceField.Name, childStructName))
				continue
			}

			fkField := findFKField(childInfo, parentInfo.ModelName)
			if fkField == nil {
				o.log(fmt.Sprintf("Warning: no FK found in child %s pointing to parent model %s (from %s.%s); skipping relation loader", childStructName, parentInfo.ModelName, parentName, sliceField.Name))
				continue
			}

			rel := RelationInfo{
				ChildStruct: childStructName,
				FKField:     fkField.Name,
				FKColumn:    fkField.ColumnName,
				LoaderName:  fmt.Sprintf("ReadAll%sBy%s", childStructName, fkField.Name),
				FKFieldType: fkField.GoType,
			}
			childInfo.Relations = append(childInfo.Relations, rel)
			all[childStructName] = childInfo
		}
	}
}

// findFKField returns the first FieldInfo in child whose Ref matches parentTable,
// or nil if none found.
func findFKField(child StructInfo, parentTable string) *FieldInfo {
	for _, f := range child.Fields {
		if f.Ref == parentTable {
			return &f
		}
	}
	return nil
}
