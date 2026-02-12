package clause_test

import (
	"reflect"
	"testing"

	"github.com/arllen133/sqlc/clause"
)

func TestExpressions(t *testing.T) {
	tests := []struct {
		name     string
		expr     interface{ Build() (string, []any, error) }
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "Eq",
			expr:     clause.Eq{Column: clause.Column{Name: "name"}, Value: "alice"},
			wantSQL:  "name = ?",
			wantArgs: []any{"alice"},
		},
		{
			name:     "Gt",
			expr:     clause.Gt{Column: clause.Column{Name: "age"}, Value: 18},
			wantSQL:  "age > ?",
			wantArgs: []any{18},
		},
		{
			name:     "In",
			expr:     clause.IN{Column: clause.Column{Name: "status"}, Values: []any{"active", "pending"}},
			wantSQL:  "status IN (?, ?)",
			wantArgs: []any{"active", "pending"},
		},
		{
			name:     "In Empty",
			expr:     clause.IN{Column: clause.Column{Name: "status"}, Values: []any{}},
			wantSQL:  "1 = 0",
			wantArgs: nil,
		},
		{
			name:     "Between",
			expr:     clause.Between{Column: clause.Column{Name: "price"}, Min: 10, Max: 100},
			wantSQL:  "price BETWEEN ? AND ?",
			wantArgs: []any{10, 100},
		},
		{
			name: "And",
			expr: clause.And{
				clause.Gt{Column: clause.Column{Name: "age"}, Value: 18},
				clause.Eq{Column: clause.Column{Name: "status"}, Value: "active"},
			},
			wantSQL:  "(age > ?) AND (status = ?)",
			wantArgs: []any{18, "active"},
		},
		{
			name: "Or",
			expr: clause.Or{
				clause.Eq{Column: clause.Column{Name: "role"}, Value: "admin"},
				clause.Eq{Column: clause.Column{Name: "role"}, Value: "moderator"},
			},
			wantSQL:  "(role = ?) OR (role = ?)",
			wantArgs: []any{"admin", "moderator"},
		},
		{
			name:     "Not",
			expr:     clause.Not{Expr: clause.Eq{Column: clause.Column{Name: "deleted"}, Value: true}},
			wantSQL:  "NOT (deleted = ?)",
			wantArgs: []any{true},
		},
		{
			name: "Nested Logic",
			expr: clause.Or{
				clause.And{
					clause.Gt{Column: clause.Column{Name: "age"}, Value: 18},
					clause.Eq{Column: clause.Column{Name: "status"}, Value: "active"},
				},
				clause.Eq{Column: clause.Column{Name: "role"}, Value: "admin"},
			},
			wantSQL:  "((age > ?) AND (status = ?)) OR (role = ?)",
			wantArgs: []any{18, "active", "admin"},
		},
		{
			name:     "Column With Table",
			expr:     clause.Eq{Column: clause.Column{Table: "users", Name: "email"}, Value: "test@example.com"},
			wantSQL:  "users.email = ?",
			wantArgs: []any{"test@example.com"},
		},
		{
			name:     "Like",
			expr:     clause.Like{Column: clause.Column{Name: "title"}, Value: "%golang%"},
			wantSQL:  "title LIKE ?",
			wantArgs: []any{"%golang%"},
		},
		{
			name:     "Assignment",
			expr:     clause.Assignment{Column: clause.Column{Name: "email"}, Value: "new@example.com"},
			wantSQL:  "email = ?",
			wantArgs: []any{"new@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs, err := tt.expr.Build()
			if err != nil {
				t.Errorf("Build() error = %v", err)
				return
			}
			if gotSQL != tt.wantSQL {
				t.Errorf("SQL: want %q, got %q", tt.wantSQL, gotSQL)
			}
			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Errorf("Args: want %v, got %v", tt.wantArgs, gotArgs)
			}
		})
	}
}

func TestOrderBy(t *testing.T) {
	col := clause.Column{Name: "created_at"}
	tests := []struct {
		name string
		expr interface{ Build() (string, []any, error) }
		want string
	}{
		{
			name: "Asc",
			expr: clause.OrderByColumn{Column: col, Desc: false},
			want: "created_at",
		},
		{
			name: "Desc",
			expr: clause.OrderByColumn{Column: col, Desc: true},
			want: "created_at DESC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := tt.expr.Build()
			if err != nil {
				t.Errorf("Build() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		})
	}
}
