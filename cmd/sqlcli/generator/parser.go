package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strings"
)

// GenConfig holds parsed configuration from config.go
type GenConfig struct {
	OutPath        string
	IncludeStructs []string
	ExcludeStructs []string
	FieldTypeMap   map[string]string
}

// ParseConfig parses config.go in the given directory for gen.Config
func ParseConfig(dir string) (*GenConfig, error) {
	configFile := filepath.Join(dir, "config.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, configFile, nil, parser.ParseComments)
	if err != nil {
		// No config.go found, return nil (use defaults)
		return nil, nil
	}

	cfg := &GenConfig{
		OutPath:      "generated", // default
		FieldTypeMap: make(map[string]string),
	}

	// Look for var _ = gen.Config{...}
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || len(valueSpec.Values) == 0 {
				continue
			}

			// Check if type is gen.Config or Config
			compLit, ok := valueSpec.Values[0].(*ast.CompositeLit)
			if !ok {
				continue
			}

			typeName := ""
			if sel, ok := compLit.Type.(*ast.SelectorExpr); ok {
				// gen.Config
				if ident, ok := sel.X.(*ast.Ident); ok {
					typeName = ident.Name + "." + sel.Sel.Name
				}
			} else if ident, ok := compLit.Type.(*ast.Ident); ok {
				// Config (local)
				typeName = ident.Name
			}

			if typeName != "gen.Config" && typeName != "Config" {
				continue
			}

			// Parse struct fields
			for _, elt := range compLit.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}

				key, ok := kv.Key.(*ast.Ident)
				if !ok {
					continue
				}

				switch key.Name {
				case "OutPath":
					if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
						cfg.OutPath = strings.Trim(lit.Value, "\"")
					}
				case "IncludeStructs":
					cfg.IncludeStructs = parseStringSlice(kv.Value)
				case "ExcludeStructs":
					cfg.ExcludeStructs = parseStringSlice(kv.Value)
				case "FieldTypeMap":
					cfg.FieldTypeMap = parseStringMap(kv.Value)
				}
			}
			return cfg, nil
		}
	}

	return cfg, nil
}

// parseStringSlice extracts string values from []any{...}
func parseStringSlice(expr ast.Expr) []string {
	var result []string
	compLit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return result
	}

	for _, elt := range compLit.Elts {
		switch v := elt.(type) {
		case *ast.BasicLit:
			if v.Kind == token.STRING {
				result = append(result, strings.Trim(v.Value, "\""))
			}
		case *ast.CompositeLit:
			// Type literal like models.User{}
			if sel, ok := v.Type.(*ast.SelectorExpr); ok {
				result = append(result, sel.Sel.Name)
			} else if ident, ok := v.Type.(*ast.Ident); ok {
				result = append(result, ident.Name)
			}
		case *ast.UnaryExpr:
			// &models.User{}
			if comp, ok := v.X.(*ast.CompositeLit); ok {
				if sel, ok := comp.Type.(*ast.SelectorExpr); ok {
					result = append(result, sel.Sel.Name)
				} else if ident, ok := comp.Type.(*ast.Ident); ok {
					result = append(result, ident.Name)
				}
			}
		}
	}
	return result
}

// parseStringMap extracts map[string]string from map literals
func parseStringMap(expr ast.Expr) map[string]string {
	result := make(map[string]string)
	compLit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return result
	}

	for _, elt := range compLit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		key := ""
		if lit, ok := kv.Key.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			key = strings.Trim(lit.Value, "\"")
		}

		val := ""
		if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			val = strings.Trim(lit.Value, "\"")
		}

		if key != "" && val != "" {
			result[key] = val
		}
	}
	return result
}

type ModelMeta struct {
	PackageName         string
	ParentPackage       string // For generated code to reference parent package
	ModulePath          string // Module path like github.com/user/project
	PackagePath         string // Package path like models
	ModelName           string
	TableName           string
	Fields              []FieldMeta
	JSONFields          []JSONFieldMeta   // JSON field path definitions
	Relations           []RelationMeta    // Relation definitions
	Doc                 []string          // Documentation comments
	CliVersion          string            // SQLCLI Version
	HasJSON             bool              // Whether imported encoding/json package is needed
	HasJSONField        bool              // Whether any field has type:json tag
	PKFieldName         string            // Cached PK Field Name
	PKColumnName        string            // Cached PK Column Name
	PKFieldType         string            // Cached PK Field Type
	IsAutoIncrementPK   bool              // Cached PK AutoIncrement status
	SchemaStructName    string            // e.g. userSchema
	IsJSONOnly          bool              // True if struct is only used as JSON embed (no db tags/PK)
	HasDBTag            bool              // True if any field has a db tag
	SoftDeleteField     string            // Name of the soft delete field (e.g. "DeletedAt")
	SoftDeleteColumn    string            // Name of the soft delete column (e.g. "deleted_at")
	SoftDeleteFieldType string            // Type of the soft delete field (e.g. "*time.Time")
	TypeAliases         map[string]string // type A int → {"A": "int"}
	FieldTypeMap        map[string]string // User-defined type mappings from config
}

// RelationMeta holds information about a model relation
type RelationMeta struct {
	FieldName           string // Field name in parent model (e.g., "Posts")
	RelType             string // Relation type: "hasOne", "hasMany", "belongsTo"
	ForeignKey          string // Foreign key column (on child for hasOne/Many, on parent for belongsTo)
	LocalKey            string // Local key column (on parent for hasOne/Many[default id], on child for belongsTo[default id])
	TargetType          string // Target model type name (e.g., "Post")
	TargetSlice         bool   // True if field is a slice (hasMany)
	ForeignKeyField     string // Go field name of foreign key (on parent for belongsTo, on target for hasOne/hasMany)
	ForeignKeyFieldType string // Go type of FK field; set only if it differs from parent PK type (for type conversion)
	TargetPKField       string // Go field name of PK on target model (used for belongsTo getForeignKey)
}

// ResolveRelationFields resolves ForeignKeyField across models for hasOne/hasMany relations.
// For belongsTo, ForeignKeyField is on the parent model (resolved during parsing).
// For hasOne/hasMany, ForeignKeyField is on the target model and needs cross-model lookup.
func ResolveRelationFields(models []ModelMeta) {
	// Build a map of model name -> ModelMeta for quick lookup
	modelMap := make(map[string]*ModelMeta, len(models))
	for i := range models {
		modelMap[models[i].ModelName] = &models[i]
	}

	for i := range models {
		for j := range models[i].Relations {
			rel := &models[i].Relations[j]
			target := modelMap[rel.TargetType]
			if target == nil {
				continue
			}

			switch rel.RelType {
			case "hasOne", "hasMany":
				// ForeignKeyField = Go field on target model matching foreignKey column
				for _, f := range target.Fields {
					if f.Column == rel.ForeignKey {
						rel.ForeignKeyField = f.FieldName
						// If FK type differs from parent PK type, record it for type conversion
						if f.Type != models[i].PKFieldType {
							rel.ForeignKeyFieldType = models[i].PKFieldType
						}
						break
					}
				}
			case "belongsTo":
				// TargetPKField = Go field name of PK on target model
				rel.TargetPKField = target.PKFieldName
			}
		}
	}
}

type FieldMeta struct {
	FieldName    string
	Column       string
	Type         string
	IsPK         bool
	AutoIncr     bool
	IsJSON       bool     // Whether field is a JSON type
	JSONTypeName string   // Name of the JSON struct type (e.g. "UserMetadata")
	Doc          []string // Documentation comments
}

// JSONFieldMeta holds information about a JSON field's path structure
type JSONFieldMeta struct {
	FieldName  string         // Name of the field in parent model (e.g. "Metadata")
	TypeName   string         // Name of the JSON struct type (e.g. "UserMetadata")
	ColumnName string         // Database column name
	Paths      []JSONPathMeta // List of paths in this JSON field
}

// JSONPathMeta holds information about a single JSON path
type JSONPathMeta struct {
	GoName   string // Go field name (e.g. "Name")
	JSONPath string // JSON path (e.g. "$.name")
}

func ParseModels(dir string) ([]ModelMeta, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var models []ModelMeta
	for pkgName, pkg := range pkgs {
		// First pass: collect type aliases (type A int)
		typeAliases := make(map[string]string)
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				ts, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				// Check if this is a type alias (not a struct)
				if _, isStruct := ts.Type.(*ast.StructType); !isStruct {
					typeName := ts.Name.Name
					underlyingType := exprToString(ts.Type)
					if underlyingType != "" {
						typeAliases[typeName] = underlyingType
					}
				}
				return true
			})
		}

		// Second pass: collect structs
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
					TypeAliases:      typeAliases,
				}

				for _, field := range st.Fields.List {
					if len(field.Names) == 0 {
						continue // Embedded fields not supported in MVP
					}

					fieldName := field.Names[0].Name
					fieldType := exprToString(field.Type)

					/* Handled by exprToString now
					// Handle array types properly for string representation
					if arr, ok := field.Type.(*ast.ArrayType); ok {
						if ident, ok := arr.Elt.(*ast.Ident); ok {
							fieldType = "[]" + ident.Name
						} else if star, ok := arr.Elt.(*ast.StarExpr); ok {
							// Handle []*Type
							if ident, ok := star.X.(*ast.Ident); ok {
								fieldType = "[]*" + ident.Name
							} else if sel, ok := star.X.(*ast.SelectorExpr); ok {
								// Handle []*pkg.Type
								if x, ok := sel.X.(*ast.Ident); ok {
									fieldType = "[]*" + x.Name + "." + sel.Sel.Name
								}
							}
						}
					}
					*/
					// Handle selector expressions (e.g. json.RawMessage)
					if sel, ok := field.Type.(*ast.SelectorExpr); ok {
						if x, ok := sel.X.(*ast.Ident); ok {
							fieldType = x.Name + "." + sel.Sel.Name
						}
					}

					// Handle generics (e.g. sqlc.JSON[Metadata])
					if idx, ok := field.Type.(*ast.IndexExpr); ok {
						typeStr := ""
						// Handle X (e.g. sqlc.JSON)
						if x, ok := idx.X.(*ast.Ident); ok {
							typeStr = x.Name
						} else if x, ok := idx.X.(*ast.SelectorExpr); ok {
							if xid, ok := x.X.(*ast.Ident); ok {
								typeStr = xid.Name + "." + x.Sel.Name
							}
						}
						// Handle Index (e.g. Metadata or models.Metadata)
						idxStr := ""
						if x, ok := idx.Index.(*ast.Ident); ok {
							idxStr = x.Name
						} else if x, ok := idx.Index.(*ast.SelectorExpr); ok {
							if xid, ok := x.X.(*ast.Ident); ok {
								idxStr = xid.Name + "." + x.Sel.Name
							}
						}

						if typeStr != "" && idxStr != "" {
							fieldType = fmt.Sprintf("%s[%s]", typeStr, idxStr)
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
							model.HasDBTag = true // Mark that this model has db tags
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
								case "type":
									if len(kv) > 1 && kv[1] == "json" {
										meta.IsJSON = true
										// Extract generic type argument if present
										if strings.Contains(meta.Type, "[") && strings.HasSuffix(meta.Type, "]") {
											start := strings.Index(meta.Type, "[")
											end := strings.LastIndex(meta.Type, "]")
											inner := meta.Type[start+1 : end]
											// Strip package prefix if present, assuming struct definition is in the parsed directory
											if lastDot := strings.LastIndex(inner, "."); lastDot != -1 {
												meta.JSONTypeName = inner[lastDot+1:]
											} else {
												meta.JSONTypeName = inner
											}
										} else {
											meta.JSONTypeName = meta.Type
										}
									}
								case "softDelete":
									model.SoftDeleteField = meta.FieldName
									model.SoftDeleteColumn = meta.Column
									model.SoftDeleteFieldType = meta.Type
								}
							}
						}
					}
					// Skip fields with db:"-" (they are not in the database)
					if meta.Column == "-" {
						// Still parse relation tag for this field before skipping
						if field.Tag != nil {
							tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
							relationTag := tag.Get("relation")
							if relationTag != "" {
								rel := parseRelationTag(fieldName, meta.Type, relationTag)
								if rel != nil {
									model.Relations = append(model.Relations, *rel)
								}
							}
						}
						continue
					}
					model.Fields = append(model.Fields, meta)

					// Cache PK info if this is the PK
					if meta.IsPK {
						model.PKFieldName = meta.FieldName
						model.PKColumnName = meta.Column
						model.PKFieldType = meta.Type
						model.IsAutoIncrementPK = meta.AutoIncr
					}

					// Check for Soft Delete field (DeletedAt *time.Time)
					if meta.FieldName == "DeletedAt" && (meta.Type == "*time.Time" || meta.Type == "sql.NullTime") {
						model.SoftDeleteField = meta.FieldName
						model.SoftDeleteColumn = meta.Column
						model.SoftDeleteFieldType = meta.Type
					}

					// Parse relation tag
					if field.Tag != nil {
						tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
						relationTag := tag.Get("relation")
						if relationTag != "" {
							rel := parseRelationTag(fieldName, meta.Type, relationTag)
							if rel != nil {
								model.Relations = append(model.Relations, *rel)
							}
						}
					}
				}
				// Mark as JSON-only if no db tags and no PK
				if !model.HasDBTag && model.PKFieldName == "" {
					model.IsJSONOnly = true
				}

				// Resolve ForeignKeyField for belongsTo relations
				for i, rel := range model.Relations {
					if rel.RelType == "belongsTo" {
						for _, f := range model.Fields {
							if f.Column == rel.ForeignKey {
								model.Relations[i].ForeignKeyField = f.FieldName
								break
							}
						}
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

// parseJSONStructPaths parses a directory for a struct type and extracts JSON paths
func parseJSONStructPaths(dir string, typeName string, prefix string) []JSONPathMeta {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	var paths []JSONPathMeta
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				ts, ok := n.(*ast.TypeSpec)
				if !ok || ts.Name.Name != typeName {
					return true
				}

				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					return true
				}

				for _, field := range st.Fields.List {
					if len(field.Names) == 0 {
						continue
					}
					fieldName := field.Names[0].Name

					// Get json tag for path name
					jsonName := toSnakeCase(fieldName)
					if field.Tag != nil {
						tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
						if jt := tag.Get("json"); jt != "" {
							parts := strings.Split(jt, ",")
							if parts[0] != "" && parts[0] != "-" {
								jsonName = parts[0]
							}
						}
					}

					fullPath := prefix + "." + jsonName
					if prefix == "" {
						fullPath = "$." + jsonName
					}

					paths = append(paths, JSONPathMeta{
						GoName:   fieldName,
						JSONPath: fullPath,
					})

					// TODO: Handle nested structs recursively if needed
				}
				return false
			})
		}
	}
	return paths
}

// exprToString converts an AST expression to its string representation
func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + exprToString(t.Elt)
		}
	case *ast.MapType:
		return "map[" + exprToString(t.Key) + "]" + exprToString(t.Value)
	}
	return ""
}

// parseRelationTag parses a relation tag like "hasMany,foreignKey:user_id,localKey:id"
func parseRelationTag(fieldName, fieldType, tag string) *RelationMeta {
	rel := &RelationMeta{
		FieldName: fieldName,
		LocalKey:  "id", // Default local key
	}

	// Determine if it's a slice (hasMany)
	rel.TargetSlice = strings.HasPrefix(fieldType, "[]") || strings.HasPrefix(fieldType, "[]*")

	// Extract target type from field type
	targetType := fieldType
	targetType = strings.TrimPrefix(targetType, "[]")
	targetType = strings.TrimPrefix(targetType, "[]*")
	targetType = strings.TrimPrefix(targetType, "*")
	// Remove package prefix if present
	if lastDot := strings.LastIndex(targetType, "."); lastDot != -1 {
		targetType = targetType[lastDot+1:]
	}
	rel.TargetType = targetType

	// Parse tag parts
	parts := strings.SplitSeq(tag, ",")
	for part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, ":") {
			kv := strings.SplitN(part, ":", 2)
			key, val := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
			switch key {
			case "foreignKey":
				rel.ForeignKey = val
			case "localKey":
				rel.LocalKey = val
			}
		} else {
			// Relation type (hasOne, hasMany, belongsTo)
			switch strings.ToLower(part) {
			case "hasone":
				rel.RelType = "hasOne"
			case "hasmany":
				rel.RelType = "hasMany"
			case "belongsto":
				rel.RelType = "belongsTo"
			}
		}
	}

	// Auto-detect relation type from field type if not specified
	if rel.RelType == "" {
		if rel.TargetSlice {
			rel.RelType = "hasMany"
		} else {
			rel.RelType = "hasOne"
		}
	}

	// Validate: must have foreignKey
	if rel.ForeignKey == "" {
		return nil
	}

	return rel
}
