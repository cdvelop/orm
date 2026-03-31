//go:build !wasm

package orm

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tinywasm/fmt"
)

type FieldInfo struct {
	Name       string
	ColumnName string
	Type       fmt.FieldType
	PK         bool
	Unique     bool
	NotNull    bool
	AutoInc    bool
	Ref        string
	RefColumn  string
	IsPK       bool
	GoType     string
	OmitEmpty  bool
	// Permitted config — populated from validate:"..." tag
	Letters bool
	Tilde   bool
	Numbers bool
	Spaces  bool
	Extra   []rune
	Minimum int
	Maximum int
	Format  string // "email", "phone", etc. (triggers validator call generation)
}

// SliceFieldInfo records a slice-of-struct field found in a parent struct.
// Not DB-mapped; used only for relation resolution.
type SliceFieldInfo struct {
	Name     string // e.g. "Roles"
	ElemType string // e.g. "Role"
}

type StructInfo struct {
	Name              string
	ModelName         string
	PackageName       string
	Fields            []FieldInfo
	ModelNameDeclared bool
	FormOnly          bool
	SourceFile        string
	SliceFields       []SliceFieldInfo // populated by ParseStruct; used by ResolveRelations
	Relations         []RelationInfo   // populated by ResolveRelations; used by GenerateForFile
}

// detectModelName scans the AST for func (X) ModelName() string on structName.
// Returns the literal return value if found, "" otherwise.
func detectModelName(node *ast.File, structName string) string {
	for _, decl := range node.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
			continue
		}
		if funcDecl.Name.Name != "ModelName" {
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
					return fmt.Convert(lit.Value).TrimPrefix(`"`).TrimSuffix(`"`).String()
				}
			}
		}
	}
	return ""
}

// ParseStruct parses a single struct from a Go file and returns its metadata.
func (o *Ormc) ParseStruct(structName string, goFile string) (StructInfo, error) {
	if structName == "" {
		return StructInfo{}, fmt.Err("Please provide a struct name")
	}

	if goFile == "" {
		return StructInfo{}, fmt.Err("goFile path cannot be empty")
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, goFile, nil, parser.ParseComments)
	if err != nil {
		return StructInfo{}, fmt.Err(err, "Failed to parse file")
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
		return StructInfo{}, fmt.Err("Struct not found in file")
	}

	modelName := detectModelName(node, structName)
	declared := modelName != ""
	if !declared {
		modelName = fmt.Convert(structName).SnakeLow().String()
	}

	info := StructInfo{
		Name:              structName,
		ModelName:         modelName,
		PackageName:       node.Name.Name,
		ModelNameDeclared: declared,
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
		jsonTag := ""
		validateTag := ""
		if field.Tag != nil {
			tagVal := fmt.Convert(field.Tag.Value).TrimPrefix("`").TrimSuffix("`").String()
			parts := fmt.Convert(tagVal).Split(" ")
			for _, p := range parts {
				if strings.HasPrefix(p, "db:\"") {
					dbTag = fmt.Convert(p).TrimPrefix(`db:"`).TrimSuffix(`"`).String()
				} else if strings.HasPrefix(p, "json:\"") {
					jsonTag = fmt.Convert(p).TrimPrefix(`json:"`).TrimSuffix(`"`).String()
				} else if strings.HasPrefix(p, "validate:\"") {
					validateTag = fmt.Convert(p).TrimPrefix(`validate:"`).TrimSuffix(`"`).String()
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
		var fieldType fmt.FieldType
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
			o.log(fmt.Sprintf("Warning: time.Time not allowed for field %s.%s; use int64+tinywasm/time. Skipping.", structName, fieldName))
			continue
		}

		switch typeStr {
		case "string":
			fieldType = fmt.FieldText
		case "int", "int32", "int64", "uint", "uint32", "uint64":
			fieldType = fmt.FieldInt
		case "float32", "float64":
			fieldType = fmt.FieldFloat
		case "bool":
			fieldType = fmt.FieldBool
		case "[]byte":
			fieldType = fmt.FieldBlob
		default:
			// If it's a struct (but not time.Time, not slice, not chan), map to FieldStruct
			if typeStr != "" && !strings.Contains(typeStr, "[") && !strings.Contains(typeStr, "chan ") {
				fieldType = fmt.FieldStruct
			} else {
				o.log(fmt.Sprintf("Warning: unsupported type %s for field %s.%s; skipping. Add db:\"-\" to suppress.", typeStr, structName, fieldName))
				continue
			}
		}

		if isPointer && fieldType != fmt.FieldStruct {
			o.log(fmt.Sprintf("Warning: pointers to primitive types not supported for field %s.%s; skipping. Add db:\"-\" to suppress.", structName, fieldName))
			continue
		}

		colName := fmt.Convert(fieldName).SnakeLow().String()
		isID, isPK := fmt.IDorPrimaryKey(modelName, fieldName)

		var pk, unique, notNull, autoInc bool
		var ref, refCol string

		fieldIsPK := false
		if (isID || isPK) && !pkFound {
			fieldIsPK = true
			pkFound = true
			pk = true
		}

		if dbTag != "" {
			tagParts := fmt.Convert(dbTag).Split(",")
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
					if fieldType == fmt.FieldText {
						return StructInfo{}, fmt.Err("autoincrement not allowed on FieldText")
					}
					autoInc = true
				case strings.HasPrefix(p, "ref="):
					refVal := fmt.Convert(p).TrimPrefix("ref=").String()
					refParts := fmt.Convert(refVal).Split(":")
					ref = refParts[0]
					if len(refParts) > 1 {
						refCol = refParts[1]
					}
				}
			}
		}

		omitEmpty := false
		if jsonTag != "" {
			parts := fmt.Convert(jsonTag).Split(",")
			for _, p := range parts {
				if p == "omitempty" {
					omitEmpty = true
				}
			}
		}

		fi := FieldInfo{
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
			OmitEmpty:  omitEmpty,
		}

		if validateTag != "" {
			parseValidateTag(validateTag, &fi)
		}

		info.Fields = append(info.Fields, fi)
	}

	return info, nil
}

// parseValidateTag maps validate:"..." rules to FieldInfo Permitted fields.
func parseValidateTag(tag string, fi *FieldInfo) {
	parts := fmt.Convert(tag).Split(",")
	for _, v := range parts {
		switch {
		case v == "required":
			fi.NotNull = true
		case v == "email":
			fi.Format = "email"
		case v == "phone":
			fi.Format = "phone"
		case v == "ip":
			fi.Format = "ip"
		case v == "rut":
			fi.Format = "rut"
		case v == "date":
			fi.Format = "date"
		case v == "name":
			fi.Letters = true
			fi.Tilde = true
			fi.Spaces = true
		case v == "letters":
			fi.Letters = true
		case v == "numbers":
			fi.Numbers = true
		case v == "tilde":
			fi.Tilde = true
		case v == "spaces":
			fi.Spaces = true
		case strings.HasPrefix(v, "min="):
			n, _ := fmt.Convert(v).TrimPrefix("min=").Int64()
			fi.Minimum = int(n)
		case strings.HasPrefix(v, "max="):
			n, _ := fmt.Convert(v).TrimPrefix("max=").Int64()
			fi.Maximum = int(n)
		}
	}
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

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[0:1]) + s[1:]
}

func writePermittedFields(buf *fmt.Conv, f FieldInfo) {
	// Use nested Permitted literal
	hasPerm := f.Letters || f.Tilde || f.Numbers || f.Spaces ||
		len(f.Extra) > 0 || f.Minimum > 0 || f.Maximum > 0

	if !hasPerm {
		return
	}

	buf.Write(", Permitted: fmt.Permitted{")
	parts := []string{}
	if f.Letters {
		parts = append(parts, "Letters: true")
	}
	if f.Tilde {
		parts = append(parts, "Tilde: true")
	}
	if f.Numbers {
		parts = append(parts, "Numbers: true")
	}
	if f.Spaces {
		parts = append(parts, "Spaces: true")
	}
	if f.Minimum > 0 {
		parts = append(parts, fmt.Sprintf("Minimum: %d", f.Minimum))
	}
	if f.Maximum > 0 {
		parts = append(parts, fmt.Sprintf("Maximum: %d", f.Maximum))
	}
	if len(f.Extra) > 0 {
		buf2 := "Extra: []rune{"
		for i, r := range f.Extra {
			if i > 0 {
				buf2 += ", "
			}
			buf2 += fmt.Sprintf("'%s'", string(r))
		}
		buf2 += "}"
		parts = append(parts, buf2)
	}

	// Join parts
	for i, p := range parts {
		if i > 0 {
			buf.Write(", ")
		}
		buf.Write(p)
	}
	buf.Write("}")
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
									o.log(fmt.Sprintf("Skipping %s in %s: %v", typeSpec.Name.Name, path, err))
									continue
								}
								if len(info.Fields) == 0 {
									o.log(fmt.Sprintf("Warning: %s has no mappable fields; skipping", typeSpec.Name.Name))
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
				o.log(fmt.Sprintf("Failed to write output for %s: %v", sourceFile, err))
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
		return fmt.Err(err, "error walking directory")
	}
	if len(all) == 0 {
		return fmt.Err("no models found")
	}

	// Pass 2: resolve cross-struct relations
	o.ResolveRelations(all)

	// Pass 3: generate (group by source file, call GenerateForFile once per file)
	if err := o.generateAll(all, structOrder, fileOrder); err != nil {
		return err
	}

	// Pass 4: sync dependencies
	if _, err := os.Stat(filepath.Join(o.rootDir, "go.mod")); err == nil {
		o.log("Syncing dependencies...")
		if err := o.exec("go", "get", "github.com/tinywasm/fmt", "github.com/tinywasm/orm", "github.com/tinywasm/form"); err != nil {
			return fmt.Err(err, "failed to get dependencies")
		}
		if err := o.exec("go", "mod", "tidy"); err != nil {
			return fmt.Err(err, "failed to tidy module")
		}
	}

	return nil
}

func (o *Ormc) exec(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = o.rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
