package benchmarks

import (
	"testing"

	"github.com/arllen133/sqlc/clause"
)

// Global variable to prevent compiler optimizations
var resultString string

// 1. Baseline: Direct access to string slice (No assertions)
func BenchmarkDirectString(b *testing.B) {
	data := []string{"id", "name", "email", "created_at"}
	for b.Loop() {
		for _, v := range data {
			resultString = v
		}
	}
}

// 2. Type Assertion: []any -> v.(string)
func BenchmarkTypeAssertion_String(b *testing.B) {
	data := []any{"id", "name", "email", "created_at"}
	for b.Loop() {
		for _, v := range data {
			// Simulate what ResolveColumnNames does for string
			if s, ok := v.(string); ok {
				resultString = s
			}
		}
	}
}

// 3. Type Switch: The exact pattern used in ResolveColumnNames (minus allocation)
func BenchmarkTypeSwitch(b *testing.B) {
	data := []any{"id", "name", "email", "created_at"}
	for b.Loop() {
		for _, arg := range data {
			switch v := arg.(type) {
			case string:
				resultString = v
			case interface{ ColumnName() string }:
				resultString = v.ColumnName()
			default:
				resultString = ""
			}
		}
	}
}

// 4. Interface Call: Calling method on interface (Overhead of dynamic dispatch)
type Namer interface {
	ColumnName() string
}

func BenchmarkInterfaceMethodCall(b *testing.B) {
	data := []Namer{
		clause.Column{Name: "id"},
		clause.Column{Name: "name"},
		clause.Column{Name: "email"},
		clause.Column{Name: "created_at"},
	}
	for b.Loop() {
		for _, v := range data {
			resultString = v.ColumnName()
		}
	}
}
