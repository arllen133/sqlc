package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
)

type ModelMeta struct {
	PackageName       string
	ParentPackage     string // For generated code to reference parent package
	ModulePath        string // Module path like github.com/user/project
	PackagePath       string // Package path like models
	ModelName         string
	TableName         string
	Fields            []FieldMeta
	Doc               []string // Documentation comments
	GeneratedAt       string   // Timestamp
	HasJSON           bool     // Whether imported json package is needed
	PKFieldName       string   // Cached PK Field Name
	PKColumnName      string   // Cached PK Column Name
	PKFieldType       string   // Cached PK Field Type
	IsAutoIncrementPK bool     // Cached PK AutoIncrement status
	SchemaStructName  string   // e.g. userSchema
}

type FieldMeta struct {
	FieldName string
	Column    string
	Type      string
	IsPK      bool
	AutoIncr  bool
	Doc       []string // Documentation comments
}

func ParseModels(dir string) ([]ModelMeta, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var models []ModelMeta
	for pkgName, pkg := range pkgs {
		for filename, file := range pkg.Files {
			if strings.HasSuffix(filename, "_gen.go") {
				continue
			}
			ast.Inspect(file, func(n ast.Node) bool {
				ts, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}

				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					return true
				}

				// Extract struct comments
				var docComments []string
				if ts.Doc != nil {
					for _, comment := range ts.Doc.List {
						docComments = append(docComments, strings.TrimPrefix(comment.Text, "// "))
					}
				}

				modelName := ts.Name.Name
				schemaStructName := strings.ToLower(modelName[:1]) + modelName[1:] + "Schema"

				model := ModelMeta{
					PackageName:      "generated",
					ParentPackage:    pkgName,
					ModelName:        modelName,
					TableName:        toSnakeCase(modelName) + "s", // Default plural
					Doc:              docComments,
					SchemaStructName: schemaStructName,
				}

				for _, field := range st.Fields.List {
					if len(field.Names) == 0 {
						continue // Embedded fields not supported in MVP
					}

					fieldName := field.Names[0].Name
					fieldType := fmt.Sprintf("%s", field.Type)

					// Handle array types properly for string representation
					if arr, ok := field.Type.(*ast.ArrayType); ok {
						if ident, ok := arr.Elt.(*ast.Ident); ok {
							fieldType = "[]" + ident.Name
						}
					}
					// Handle selector expressions (e.g. json.RawMessage)
					if sel, ok := field.Type.(*ast.SelectorExpr); ok {
						if x, ok := sel.X.(*ast.Ident); ok {
							fieldType = x.Name + "." + sel.Sel.Name
						}
					}

					meta := FieldMeta{
						FieldName: fieldName,
						Column:    toSnakeCase(fieldName),
						Type:      fieldType,
					}

					// Extract field comments
					if field.Doc != nil {
						for _, comment := range field.Doc.List {
							meta.Doc = append(meta.Doc, strings.TrimPrefix(comment.Text, "// "))
						}
					} else if field.Comment != nil {
						for _, comment := range field.Comment.List {
							meta.Doc = append(meta.Doc, strings.TrimPrefix(comment.Text, "// "))
						}
					}

					if field.Tag != nil {
						tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
						ormTag := tag.Get("db")
						if ormTag == "" {
							ormTag = tag.Get("orm") // Fallback
						}

						if ormTag != "" {
							// Normalize separators: replace ; with ,
							ormTag = strings.ReplaceAll(ormTag, ";", ",")
							// Split by comma
							parts := strings.Split(ormTag, ",")

							// First part is column name (unless it's empty?)
							if len(parts) > 0 && parts[0] != "" {
								// Check if it's a KV like "table:xxx" or just "name"
								if !strings.Contains(parts[0], ":") {
									meta.Column = parts[0]
								}
							}

							for _, part := range parts {
								kv := strings.Split(part, ":")
								key := kv[0]

								// Handle flags
								switch key {
								case "primaryKey":
									meta.IsPK = true
								case "autoIncrement":
									meta.AutoIncr = true
								case "table":
									if len(kv) > 1 {
										model.TableName = kv[1]
									}
								case "column":
									// Legacy support or explicit "column:xxx"
									if len(kv) > 1 {
										meta.Column = kv[1]
									}
								}
							}
						}
					}
					model.Fields = append(model.Fields, meta)

					// Cache PK info if this is the PK
					if meta.IsPK {
						model.PKFieldName = meta.FieldName
						model.PKColumnName = meta.Column
						model.PKFieldType = meta.Type
						model.IsAutoIncrementPK = meta.AutoIncr
					}
				}
				models = append(models, model)
				return true
			})
		}
	}
	return models, nil
}

func toSnakeCase(s string) string {
	var res strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				res.WriteRune('_')
			}
			res.WriteRune(r + ('a' - 'A'))
		} else {
			res.WriteRune(r)
		}
	}
	return res.String()
}
