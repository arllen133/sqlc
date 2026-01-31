package type_alias

import "github.com/arllen133/sqlc/gen"

// config.go - Optional: override type mappings
var _ = gen.Config{
	// IncludeStructs: []any{"User"}, // Generate only User

	// Optional: Override automatic type alias resolution
	// FieldTypeMap: map[string]string{
	// 	"Status": "field.String", // Force Status to use String instead of Number
	// },
}
