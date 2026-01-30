package gen

// Config defines the code generation configuration.
// Place this in a file named `config.go` in your model directory.
type Config struct {
	// OutPath specifies the output directory for generated files.
	// Relative to the model directory. Default: "generated"
	OutPath string

	// IncludeStructs specifies which structs to generate.
	// Supports string names or type instances: []any{"User", &Post{}}
	// If empty, all structs with db tags are generated.
	IncludeStructs []any

	// ExcludeStructs specifies which structs to skip.
	// Supports string names: []any{"BaseModel", "Internal*"}
	ExcludeStructs []any

	// FieldTypeMap maps Go types to field types.
	// Example: map[string]string{"sql.NullTime": "field.Time"}
	FieldTypeMap map[string]string
}

// ConfigFileName is the convention filename for configuration.
const ConfigFileName = "config.go"
