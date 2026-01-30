package gentest

import (
	"github.com/arllen133/sqlc/gen"
)

// Configuration for sqlcli code generation.
// Only the User struct will be generated; Settings struct is excluded.
var _ = gen.Config{
	// OutPath: "" means use default (./generated relative to model dir)
	// OutPath: "../output" would generate to ../output/generated
	IncludeStructs: []any{"User"},     // Only generate User schema
	ExcludeStructs: []any{"Settings"}, // Explicitly exclude Settings
}
