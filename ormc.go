//go:build !wasm

package orm

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	. "github.com/tinywasm/fmt"
)

type FieldInfo struct {
	Name       string
	ColumnName string
	Type       FieldType
	PK         bool
	Unique     bool
	NotNull    bool
	AutoInc    bool
	Ref        string
	RefColumn  string
	IsPK       bool
	GoType     string
	Input      string
	JSON       string
}

// SliceFieldInfo records a slice-of-struct field found in a parent struct.
// Not DB-mapped; used only for relation resolution.
type SliceFieldInfo struct {
	Name     string // e.g. "Roles"
	ElemType string // e.g. "Role"
}

type StructInfo struct {
	Name              string
	TableName         string
	PackageName       string
	Fields            []FieldInfo
	TableNameDeclared bool
	FormOnly          bool
	SourceFile        string
	SliceFields       []SliceFieldInfo // populated by ParseStruct; used by ResolveRelations
	Relations         []RelationInfo   // populated by ResolveRelations; used by GenerateForFile
}

// detectTableName scans the AST for func (X) TableName() string on structName.
// Returns the literal return value if found, "" otherwise.
func detectTableName(node *ast.File, structName string) string {
	for _, decl := range node.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
			continue
		}
		if funcDecl.Name.Name != "TableName" {
			continue
		}
		recv := funcDecl.Recv.List[0].Type
		recvName := ""
		if ident, ok := recv.(*ast.Ident); ok {
			recvName = ident.Name
		} else if star, ok := recv.(*ast.StarExpr); ok {
			if ident, ok := star.X.(*ast.Ident); ok {
				recvName = ident.Name
			}
		}
		if recvName != structName {
			continue
		}
		if funcDecl.Body != nil && len(funcDecl.Body.List) == 1 {
			if ret, ok := funcDecl.Body.List[0].(*ast.ReturnStmt); ok && len(ret.Results) == 1 {
				if lit, ok := ret.Results[0].(*ast.BasicLit); ok {
					return Convert(lit.Value).TrimPrefix(`"`).TrimSuffix(`"`).String()
				}
			}
		}
	}
	return ""
}

// ParseStruct parses a single struct from a Go file and returns its metadata.
func (o *Ormc) ParseStruct(structName string, goFile string) (StructInfo, error) {
	if structName == "" {
		return StructInfo{}, Err("Please provide a struct name")
	}

	if goFile == "" {
		return StructInfo{}, Err("goFile path cannot be empty")
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, goFile, nil, parser.ParseComments)
	if err != nil {
		return StructInfo{}, Err(err, "Failed to parse file")
	}

	var targetStruct *ast.StructType
	var structFound bool
	var formOnly bool

	ast.Inspect(node, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if typeSpec.Name.Name == structName {
						if structType, ok := typeSpec.Type.(*ast.StructType); ok {
							targetStruct = structType
							structFound = true
							if genDecl.Doc != nil {
								for _, comment := range genDecl.Doc.List {
									if strings.Contains(comment.Text, "ormc:formonly") {
										formOnly = true
										break
									}
								}
							}
							return false
						}
					}
				}
			}
		}
		return true
	})

	if !structFound {
		return StructInfo{}, Err("Struct not found in file")
	}

	tableName := detectTableName(node, structName)
	declared := tableName != ""
	if !declared {
		tableName = Convert(structName).SnakeLow().String()
	}

	info := StructInfo{
		Name:              structName,
		TableName:         tableName,
		PackageName:       node.Name.Name,
		TableNameDeclared: declared,
		FormOnly:          formOnly,
	}

	pkFound := false
	for _, field := range targetStruct.Fields.List {
		if len(field.Names) == 0 {
			continue // Anonymous field, skip for now
		}

		fieldName := field.Names[0].Name
		if !ast.IsExported(fieldName) {
			continue
		}

		dbTag := ""
		formTag := ""
		jsonTag := ""
		if field.Tag != nil {
			tagVal := Convert(field.Tag.Value).TrimPrefix("`").TrimSuffix("`").String()
			parts := Convert(tagVal).Split(" ")
			for _, p := range parts {
				if HasPrefix(p, "db:\"") {
					dbTag = Convert(p).TrimPrefix(`db:"`).TrimSuffix(`"`).String()
				} else if HasPrefix(p, "form:\"") {
					formTag = Convert(p).TrimPrefix(`form:"`).TrimSuffix(`"`).String()
				} else if HasPrefix(p, "json:\"") {
					jsonTag = Convert(p).TrimPrefix(`json:"`).TrimSuffix(`"`).String()
				}
			}
		}

		if dbTag == "-" {
			continue
		}

		// Detect []Struct fields for relation resolution (R8)
		if arr, ok := field.Type.(*ast.ArrayType); ok {
			if eltIdent, ok := arr.Elt.(*ast.Ident); ok && eltIdent.Name != "byte" {
				info.SliceFields = append(info.SliceFields, SliceFieldInfo{
					Name:     fieldName,
					ElemType: eltIdent.Name,
				})
				continue // never add to Fields — not DB-mappable
			}
		}

		// Field Type mapping
		var fieldType FieldType
		var typeStr string
		var isPointer bool

		fType := field.Type
		if star, ok := fType.(*ast.StarExpr); ok {
			isPointer = true
			fType = star.X
		}

		if ident, ok := fType.(*ast.Ident); ok {
			typeStr = ident.Name
		} else if sel, ok := fType.(*ast.SelectorExpr); ok {
			if pkgIdent, ok := sel.X.(*ast.Ident); ok {
				typeStr = pkgIdent.Name + "." + sel.Sel.Name
			}
		} else if arr, ok := fType.(*ast.ArrayType); ok {
			if eltIdent, ok := arr.Elt.(*ast.Ident); ok && eltIdent.Name == "byte" {
				typeStr = "[]byte"
			}
		}

		if typeStr == "time.Time" {
			o.log(Sprintf("Warning: time.Time not allowed for field %s.%s; use int64+tinywasm/time. Skipping.", structName, fieldName))
			continue
		}

		switch typeStr {
		case "string":
			fieldType = FieldText
		case "int", "int32", "int64", "uint", "uint32", "uint64":
			fieldType = FieldInt
		case "float32", "float64":
			fieldType = FieldFloat
		case "bool":
			fieldType = FieldBool
		case "[]byte":
			fieldType = FieldBlob
		default:
			// If it's a struct (but not time.Time, not slice, not chan), map to FieldStruct
			if typeStr != "" && !strings.Contains(typeStr, "[") && !strings.Contains(typeStr, "chan ") {
				fieldType = FieldStruct
			} else {
				o.log(Sprintf("Warning: unsupported type %s for field %s.%s; skipping. Add db:\"-\" to suppress.", typeStr, structName, fieldName))
				continue
			}
		}

		if isPointer && fieldType != FieldStruct {
			o.log(Sprintf("Warning: pointers to primitive types not supported for field %s.%s; skipping. Add db:\"-\" to suppress.", structName, fieldName))
			continue
		}

		colName := Convert(fieldName).SnakeLow().String()
		isID, isPK := IDorPrimaryKey(tableName, fieldName)

		var pk, unique, notNull, autoInc bool
		var ref, refCol string

		fieldIsPK := false
		if (isID || isPK) && !pkFound {
			fieldIsPK = true
			pkFound = true
			pk = true
		}

		if dbTag != "" {
			tagParts := Convert(dbTag).Split(",")
			for _, p := range tagParts {
				switch {
				case p == "pk":
					if !fieldIsPK {
						pk = true
						fieldIsPK = true
						pkFound = true
					}
				case p == "unique":
					unique = true
				case p == "not_null":
					notNull = true
				case p == "autoincrement":
					if fieldType == FieldText {
						return StructInfo{}, Err("autoincrement not allowed on FieldText")
					}
					autoInc = true
				case HasPrefix(p, "ref="):
					refVal := Convert(p).TrimPrefix("ref=").String()
					refParts := Convert(refVal).Split(":")
					ref = refParts[0]
					if len(refParts) > 1 {
						refCol = refParts[1]
					}
				}
			}
		}

		info.Fields = append(info.Fields, FieldInfo{
			Name:       fieldName,
			ColumnName: colName,
			Type:       fieldType,
			PK:         pk,
			Unique:     unique,
			NotNull:    notNull,
			AutoInc:    autoInc,
			Ref:        ref,
			RefColumn:  refCol,
			IsPK:       fieldIsPK,
			GoType:     typeStr,
			Input:      formTag,
			JSON:       jsonTag,
		})
	}

	return info, nil
}

// GenerateForStruct reads the Go File and generates the ORM implementations for a given struct name.
func (o *Ormc) GenerateForStruct(structName string, goFile string) error {
	info, err := o.ParseStruct(structName, goFile)
	if err != nil {
		return err
	}
	if len(info.Fields) == 0 {
		return nil
	}
	return o.GenerateForFile([]StructInfo{info}, goFile)
}

// GenerateForFile writes ORM implementations for all infos into one file.
func (o *Ormc) GenerateForFile(infos []StructInfo, sourceFile string) error {
	if len(infos) == 0 {
		return nil
	}
	buf := Convert()

	// File Header
	buf.Write(Sprintf("//go:build !wasm\n"))
	buf.Write(Sprintf("// DO NOT EDIT. generated by github.com/tinywasm/orm\n\n"))
	buf.Write(Sprintf("package %s\n\n", infos[0].PackageName))

	hasModel := false
	for _, info := range infos {
		if !info.FormOnly {
			hasModel = true
			break
		}
	}

	buf.Write("import (\n")
	buf.Write("\t\"github.com/tinywasm/fmt\"\n")
	if hasModel {
		buf.Write("\t\"github.com/tinywasm/orm\"\n")
	}
	buf.Write(")\n\n")

	for _, info := range infos {
		if !info.FormOnly {
			// Model Interface Methods
			if !info.TableNameDeclared {
				buf.Write(Sprintf("func (m *%s) TableName() string {\n", info.Name))
				buf.Write(Sprintf("\treturn \"%s\"\n", info.TableName))
				buf.Write("}\n\n")
			}
		}

		buf.Write(Sprintf("func (m *%s) FormName() string {\n", info.Name))
		buf.Write(Sprintf("\treturn \"%s\"\n", Convert(info.Name).SnakeLow().String()))
		buf.Write("}\n\n")

		buf.Write(Sprintf("var _schema%s = []fmt.Field{\n", info.Name))
		for _, f := range info.Fields {
			typeStr := "fmt.FieldText"
			switch f.Type {
			case FieldInt:
				typeStr = "fmt.FieldInt"
			case FieldFloat:
				typeStr = "fmt.FieldFloat"
			case FieldBool:
				typeStr = "fmt.FieldBool"
			case FieldBlob:
				typeStr = "fmt.FieldBlob"
			case FieldStruct:
				typeStr = "fmt.FieldStruct"
			}

			buf.Write(Sprintf("\t\t{Name: \"%s\", Type: %s", f.ColumnName, typeStr))
			if f.PK {
				buf.Write(", PK: true")
			}
			if f.Unique {
				buf.Write(", Unique: true")
			}
			if f.NotNull {
				buf.Write(", NotNull: true")
			}
			if f.AutoInc {
				buf.Write(", AutoInc: true")
			}
			if f.Input != "" {
				buf.Write(Sprintf(", Input: \"%s\"", f.Input))
			}
			if f.JSON != "" {
				buf.Write(Sprintf(", JSON: \"%s\"", f.JSON))
			}
			buf.Write("},\n")
		}
		buf.Write("\t}\n\n")

		buf.Write(Sprintf("func (m *%s) Schema() []fmt.Field { return _schema%s }\n\n", info.Name, info.Name))

		buf.Write(Sprintf("func (m *%s) Pointers() []any {\n", info.Name))
		buf.Write("\treturn []any{\n")
		for _, f := range info.Fields {
			buf.Write(Sprintf("\t\t&m.%s,\n", f.Name))
		}
		buf.Write("\t}\n")
		buf.Write("}\n\n")

		if !info.FormOnly {
			// Metadata Descriptors
			buf.Write(Sprintf("var %s_ = struct {\n", info.Name))
			buf.Write("\tTableName string\n")
			for _, f := range info.Fields {
				buf.Write(Sprintf("\t%s string\n", f.Name))
			}
			buf.Write("}{\n")
			buf.Write(Sprintf("\tTableName: \"%s\",\n", info.TableName))
			for _, f := range info.Fields {
				buf.Write(Sprintf("\t%s: \"%s\",\n", f.Name, f.ColumnName))
			}
			buf.Write("}\n\n")

			// Typed Read Operations
			buf.Write(Sprintf("func ReadOne%s(qb *orm.QB, model *%s) (*%s, error) {\n", info.Name, info.Name, info.Name))
			buf.Write("\terr := qb.ReadOne()\n")
			buf.Write("\tif err != nil {\n")
			buf.Write("\t\treturn nil, err\n")
			buf.Write("\t}\n")
			buf.Write("\treturn model, nil\n")
			buf.Write("}\n\n")

			buf.Write(Sprintf("func ReadAll%s(qb *orm.QB) ([]*%s, error) {\n", info.Name, info.Name))
			buf.Write(Sprintf("\tvar results []*%s\n", info.Name))
			buf.Write("\terr := qb.ReadAll(\n")
			buf.Write(Sprintf("\t\tfunc() orm.Model { return &%s{} },\n", info.Name))
			buf.Write(Sprintf("\t\tfunc(m orm.Model) { results = append(results, m.(*%s)) },\n", info.Name))
			buf.Write("\t)\n")
			buf.Write("\treturn results, err\n")
			buf.Write("}\n\n")

			for _, rel := range info.Relations {
				buf.Write(Sprintf(
					"// ReadAll%sByParentID retrieves all %s records for a given parent ID.\n"+
						"// Auto-generated by ormc — relation detected via db:\"ref=%s\".\n"+
						"func ReadAll%sBy%s(db *orm.DB, parentID %s) ([]*%s, error) {\n"+
						"\treturn ReadAll%s(db.Query(&%s{}).Where(%s_.%s).Eq(parentID))\n"+
						"}\n\n",
					rel.ChildStruct,
					rel.ChildStruct,
					info.TableName, // parent table, for the comment
					rel.ChildStruct, rel.FKField, rel.FKFieldType,
					rel.ChildStruct,
					rel.ChildStruct, rel.ChildStruct, rel.ChildStruct, rel.FKField,
				))
			}
		}
	}

	outName := Convert(sourceFile).TrimSuffix(".go").String() + "_orm.go"
	return os.WriteFile(outName, buf.Bytes(), 0644)
}

// collectAllStructs walks rootDir and returns a map of all parsed StructInfo
// keyed by struct name. Used by Run() Pass 1.
func (o *Ormc) collectAllStructs() (map[string]StructInfo, []string, []string, error) {
	all := make(map[string]StructInfo)
	var structOrder []string
	var fileOrder []string
	fileSeen := make(map[string]bool)

	err := filepath.Walk(o.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			dirName := info.Name()
			if dirName == "vendor" || dirName == ".git" || dirName == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		fileName := info.Name()
		if fileName == "model.go" || fileName == "models.go" {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				return nil // Skip unparseable files
			}

			for _, decl := range node.Decls {
				if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
					for _, spec := range genDecl.Specs {
						if typeSpec, ok := spec.(*ast.TypeSpec); ok {
							if _, ok := typeSpec.Type.(*ast.StructType); ok {
								info, err := o.ParseStruct(typeSpec.Name.Name, path)
								if err != nil {
									o.log(Sprintf("Skipping %s in %s: %v", typeSpec.Name.Name, path, err))
									continue
								}
								if len(info.Fields) == 0 {
									o.log(Sprintf("Warning: %s has no mappable fields; skipping", typeSpec.Name.Name))
									continue
								}
								info.SourceFile = path
								all[info.Name] = info
								structOrder = append(structOrder, info.Name)
								if !fileSeen[path] {
									fileSeen[path] = true
									fileOrder = append(fileOrder, path)
								}
							}
						}
					}
				}
			}
		}

		return nil
	})

	return all, structOrder, fileOrder, err
}

// generateAll groups the enriched all map by source file path and calls
// GenerateForFile once per file.
func (o *Ormc) generateAll(all map[string]StructInfo, structOrder []string, fileOrder []string) error {
	byFile := make(map[string][]StructInfo)
	for _, structName := range structOrder {
		info := all[structName]
		byFile[info.SourceFile] = append(byFile[info.SourceFile], info)
	}

	for _, sourceFile := range fileOrder {
		infos := byFile[sourceFile]
		if len(infos) > 0 {
			if err := o.GenerateForFile(infos, sourceFile); err != nil {
				o.log(Sprintf("Failed to write output for %s: %v", sourceFile, err))
			}
		}
	}
	return nil
}

// Run is the entry point for the CLI tool.
func (o *Ormc) Run() error {
	// Pass 1: collect all structs across all model files
	all, structOrder, fileOrder, err := o.collectAllStructs()
	if err != nil {
		return Err(err, "error walking directory")
	}
	if len(all) == 0 {
		return Err("no models found")
	}

	// Pass 2: resolve cross-struct relations
	o.ResolveRelations(all)

	// Pass 3: generate (group by source file, call GenerateForFile once per file)
	return o.generateAll(all, structOrder, fileOrder)
}
