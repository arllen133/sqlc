package generator_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arllen133/sqlc/cmd/sqlcli/generator"
)

func TestParseConfig_NoConfigFile(t *testing.T) {
	// Create a temp directory without config.go
	dir := t.TempDir()

	cfg, err := generator.ParseConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return nil when no config.go exists
	if cfg != nil {
		t.Errorf("expected nil config, got: %+v", cfg)
	}
}

func TestParseConfig_EmptyConfig(t *testing.T) {
	dir := t.TempDir()

	// Create a config.go with empty Config
	configContent := `package test

import "github.com/arllen133/sqlc/gen"

var _ = gen.Config{}
`
	err := os.WriteFile(filepath.Join(dir, "config.go"), []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write config.go: %v", err)
	}

	cfg, err := generator.ParseConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	// Default values
	if cfg.OutPath != "generated" {
		t.Errorf("expected OutPath 'generated', got '%s'", cfg.OutPath)
	}
}

func TestParseConfig_WithOutPath(t *testing.T) {
	dir := t.TempDir()

	configContent := `package test

import "github.com/arllen133/sqlc/gen"

var _ = gen.Config{
	OutPath: "../output",
}
`
	err := os.WriteFile(filepath.Join(dir, "config.go"), []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write config.go: %v", err)
	}

	cfg, err := generator.ParseConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.OutPath != "../output" {
		t.Errorf("expected OutPath '../output', got '%s'", cfg.OutPath)
	}
}

func TestParseConfig_WithIncludeStructs(t *testing.T) {
	dir := t.TempDir()

	configContent := `package test

import "github.com/arllen133/sqlc/gen"

var _ = gen.Config{
	IncludeStructs: []any{"User", "Post"},
}
`
	err := os.WriteFile(filepath.Join(dir, "config.go"), []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write config.go: %v", err)
	}

	cfg, err := generator.ParseConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.IncludeStructs) != 2 {
		t.Fatalf("expected 2 IncludeStructs, got %d", len(cfg.IncludeStructs))
	}

	if cfg.IncludeStructs[0] != "User" || cfg.IncludeStructs[1] != "Post" {
		t.Errorf("expected ['User', 'Post'], got %v", cfg.IncludeStructs)
	}
}

func TestParseConfig_WithExcludeStructs(t *testing.T) {
	dir := t.TempDir()

	configContent := `package test

import "github.com/arllen133/sqlc/gen"

var _ = gen.Config{
	ExcludeStructs: []any{"BaseModel", "Internal"},
}
`
	err := os.WriteFile(filepath.Join(dir, "config.go"), []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write config.go: %v", err)
	}

	cfg, err := generator.ParseConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.ExcludeStructs) != 2 {
		t.Fatalf("expected 2 ExcludeStructs, got %d", len(cfg.ExcludeStructs))
	}

	if cfg.ExcludeStructs[0] != "BaseModel" || cfg.ExcludeStructs[1] != "Internal" {
		t.Errorf("expected ['BaseModel', 'Internal'], got %v", cfg.ExcludeStructs)
	}
}

func TestParseConfig_FullConfig(t *testing.T) {
	dir := t.TempDir()

	configContent := `package test

import "github.com/arllen133/sqlc/gen"

var _ = gen.Config{
	OutPath:        "custom_output",
	IncludeStructs: []any{"User"},
	ExcludeStructs: []any{"BaseModel"},
	FieldTypeMap: map[string]string{
		"sql.NullTime": "field.Time",
	},
}
`
	err := os.WriteFile(filepath.Join(dir, "config.go"), []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write config.go: %v", err)
	}

	cfg, err := generator.ParseConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.OutPath != "custom_output" {
		t.Errorf("expected OutPath 'custom_output', got '%s'", cfg.OutPath)
	}

	if len(cfg.IncludeStructs) != 1 || cfg.IncludeStructs[0] != "User" {
		t.Errorf("expected IncludeStructs ['User'], got %v", cfg.IncludeStructs)
	}

	if len(cfg.ExcludeStructs) != 1 || cfg.ExcludeStructs[0] != "BaseModel" {
		t.Errorf("expected ExcludeStructs ['BaseModel'], got %v", cfg.ExcludeStructs)
	}

	if cfg.FieldTypeMap["sql.NullTime"] != "field.Time" {
		t.Errorf("expected FieldTypeMap['sql.NullTime']='field.Time', got %v", cfg.FieldTypeMap)
	}
}
