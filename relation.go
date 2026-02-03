package sqlc

import (
	"context"

	"github.com/arllen133/sqlc/clause"
)

// RelationType defines the type of relationship
type RelationType int

const (
	// RelationHasOne indicates a 1:1 relationship (parent has one child)
	RelationHasOne RelationType = iota
	// RelationHasMany indicates a 1:N relationship (parent has many children)
	RelationHasMany
)

// Relation defines a relationship between parent model P and child model C.
// This can be created by the code generator or manually defined.
type Relation[P, C any] struct {
	// Type is the relationship type (HasOne or HasMany)
	Type RelationType

	// ForeignKey is the column on the child table that references the parent
	ForeignKey clause.Column

	// LocalKey is the column on the parent table (usually primary key)
	LocalKey clause.Column

	// Setter sets the loaded children on the parent model
	// For HasOne: children slice will have 0 or 1 element
	// For HasMany: children slice contains all related records
	Setter func(parent *P, children []*C)

	// GetLocalKeyValue extracts the local key value from a parent model
	GetLocalKeyValue func(parent *P) any
}

// preloadConfig stores configuration for a single preload operation
type preloadConfig struct {
	execute func(ctx context.Context, session *Session, parentIDs []any, parents any) error
}

// HasOne creates a HasOne relation definition
func HasOne[P, C any](
	foreignKey clause.Column,
	localKey clause.Column,
	setter func(*P, *C),
	getLocalKey func(*P) any,
) Relation[P, C] {
	return Relation[P, C]{
		Type:       RelationHasOne,
		ForeignKey: foreignKey,
		LocalKey:   localKey,
		Setter: func(p *P, children []*C) {
			if len(children) > 0 {
				setter(p, children[0])
			}
		},
		GetLocalKeyValue: getLocalKey,
	}
}

// HasMany creates a HasMany relation definition
func HasMany[P, C any](
	foreignKey clause.Column,
	localKey clause.Column,
	setter func(*P, []*C),
	getLocalKey func(*P) any,
) Relation[P, C] {
	return Relation[P, C]{
		Type:             RelationHasMany,
		ForeignKey:       foreignKey,
		LocalKey:         localKey,
		Setter:           setter,
		GetLocalKeyValue: getLocalKey,
	}
}

// Preload creates a preload executor for the given relation.
// This is a standalone function that creates the preload function to be added to QueryBuilder.
func Preload[P, C any](rel Relation[P, C]) preloadExecutor[P] {
	return func(ctx context.Context, session *Session, parents []*P) error {
		if len(parents) == 0 {
			return nil
		}

		// Collect parent IDs
		parentIDs := make([]any, 0, len(parents))
		parentMap := make(map[int64][]*P) // Map normalized parent ID to parent(s)
		for _, p := range parents {
			id := rel.GetLocalKeyValue(p)
			parentIDs = append(parentIDs, id)
			normalizedID := normalizeToInt64(id)
			parentMap[normalizedID] = append(parentMap[normalizedID], p)
		}

		// Build IN query for children
		childSchema := LoadSchema[C]()
		query := Query[C](session).Where(clause.IN{
			Column: rel.ForeignKey,
			Values: parentIDs,
		})

		// Execute query
		children, err := query.Find(ctx)
		if err != nil {
			return err
		}

		// Build child map: FK value -> children
		childMap := make(map[int64][]*C)
		for _, child := range children {
			fkValue := getFieldValue(child, rel.ForeignKey.Name)
			normalizedFK := normalizeToInt64(fkValue)
			childMap[normalizedFK] = append(childMap[normalizedFK], child)
		}

		// Assign children to parents
		for _, p := range parents {
			id := rel.GetLocalKeyValue(p)
			normalizedID := normalizeToInt64(id)
			rel.Setter(p, childMap[normalizedID])
		}

		// Initialize empty slices for HasMany (avoid nil)
		if rel.Type == RelationHasMany {
			for _, p := range parents {
				id := rel.GetLocalKeyValue(p)
				normalizedID := normalizeToInt64(id)
				if _, ok := childMap[normalizedID]; !ok {
					rel.Setter(p, []*C{})
				}
			}
		}

		_ = childSchema // Ensure schema is loaded
		return nil
	}
}

// normalizeToInt64 converts common numeric types to int64 for consistent map key comparison
func normalizeToInt64(v any) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int8:
		return int64(val)
	case int16:
		return int64(val)
	case int32:
		return int64(val)
	case int64:
		return val
	case uint:
		return int64(val)
	case uint8:
		return int64(val)
	case uint16:
		return int64(val)
	case uint32:
		return int64(val)
	case uint64:
		return int64(val)
	case float32:
		return int64(val)
	case float64:
		return int64(val)
	default:
		return 0
	}
}
