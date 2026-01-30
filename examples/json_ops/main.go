package main

import (
	"context"
	"database/sql"
	"fmt"

	// _ "github.com/go-sql-driver/mysql"

	"github.com/arllen133/sqlc"
	"github.com/arllen133/sqlc/examples/blog/models"
	"github.com/arllen133/sqlc/examples/blog/models/generated"
)

func main() {
	// =================================================================
	// 1. SETUP
	// =================================================================
	db, _ := sql.Open("mysql", "user:pass@tcp(localhost:3306)/blog")

	// NewSession automatically sets json.DefaultDialect based on driver
	sess := sqlc.NewSession(db, &sqlc.MySQLDialect{})
	repo := sqlc.NewRepository[models.Post](sess)
	ctx := context.Background()

	// Shortcut for path accessor
	meta := generated.PostMetadata

	// (Optional) Explicitly set default dialect if needed globaly
	// json.SetDefaultDialect(json.MySQL)

	fmt.Println("--- START JSON DEMOS ---")

	// =================================================================
	// 2. QUERYING (JSON Filtering)
	// =================================================================

	// A. Basic Equality
	// Find posts where view_count = 1000
	posts, _ := repo.Query().Where(
		meta.ViewCount.Eq(1000),
	).Find(ctx)
	printCount("Eq(1000)", posts)

	// B. Not Equal
	// Find posts where category_name != 'Draft'
	posts, _ = repo.Query().Where(
		meta.CategoryName.Neq("Draft"),
	).Find(ctx)
	printCount("Neq('Draft')", posts)

	// C. Boolean Logic
	// Find posts that are featured (is_featured = true)
	posts, _ = repo.Query().Where(
		meta.IsFeatured.Eq(true),
	).Find(ctx)
	printCount("IsFeatured=true", posts)

	// D. Range Queries (Gt, Gte, Lt, Lte)
	// Find posts with 100 <= view_count <= 1000
	posts, _ = repo.Query().
		Where(meta.ViewCount.Gte(100)).
		Where(meta.ViewCount.Lte(1000)).
		Find(ctx)
	printCount("ViewCount [100, 1000]", posts)

	// E. Array Operations
	// Find posts where "tags" array contains "golang"
	// (Note: Uses JSON_CONTAINS in MySQL)
	posts, _ = repo.Query().Where(
		meta.Tags.Contains("golang"),
	).Find(ctx)
	printCount("Tags contains 'golang'", posts)

	// F. Multiple Conditions (Compound)
	// Featured posts AND Category='Tech'
	posts, _ = repo.Query().
		Where(meta.IsFeatured.Eq(true)).
		Where(meta.CategoryName.Eq("Tech")).
		Find(ctx)
	printCount("Featured + Tech", posts)

	// =================================================================
	// 3. UPDATING (JSON Modification)
	// =================================================================

	id := int64(1)

	// A. Single Field Update
	// Update "view_count" to 0
	_ = repo.UpdateColumns(ctx, id,
		meta.ViewCount.Set(0),
	)
	fmt.Println("Updated ViewCount -> 0")

	// B. Multiple Fields Update (Batch)
	// Sets "is_featured" = true AND "category_name" = "Hot" in one SQL statement
	_ = repo.UpdateColumns(ctx, id, generated.Post.Metadata.SetPaths(
		meta.IsFeatured.Arg(true),
		meta.CategoryName.Arg("Hot"),
	))
	fmt.Println("Batch Updated Featured & Category")

	// C. Removing a Field
	// Remove "like_count" key from JSON
	_ = repo.UpdateColumns(ctx, id,
		meta.LikeCount.Remove(),
	)
	fmt.Println("Removed LikeCount")

	// =================================================================
	// 4. MERGE OPERATIONS
	// =================================================================

	// A. Merge Patch (RFC 7396)
	// Replaces arrays, merges objects, deletes nulls.
	// Example: Tags array is REPLACED by new array.
	_ = repo.UpdateColumns(ctx, id,
		generated.Post.Metadata.MergePatch(map[string]any{
			"view_count": 5000,
			"tags":       []string{"Merge", "Patch"},
		}),
	)
	fmt.Println("MergePatch executed")

	// B. Merge Preserve (Legacy/Concatenation)
	// Appends to arrays (in MySQL/Postgres-simulated).
	// Example: "tags" will likely be appended (implementation dependent)
	_ = repo.UpdateColumns(ctx, id,
		generated.Post.Metadata.MergePreserve(map[string]any{
			"tags": []string{"AppendedTag"},
		}),
	)
	fmt.Println("MergePreserve executed")

	// =================================================================
	// 5. READING VALUES
	// =================================================================

	// Fetch updated post
	post, _ := repo.FindOne(ctx, id)
	if post != nil {
		fmt.Printf("Current Post Data:\n")
		fmt.Printf(" - ViewCount: %d\n", post.Metadata.Data.ViewCount)
		fmt.Printf(" - Tags: %v\n", post.Metadata.Data.Tags)
		fmt.Printf(" - Category: %s\n", post.Metadata.Data.CategoryName)
	}
}

func printCount(label string, posts []*models.Post) {
	fmt.Printf("[%s] Found: %d\n", label, len(posts))
}
