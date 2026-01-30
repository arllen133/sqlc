package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/arllen133/sqlc/cmd/orm-gen/generator"
)

func main() {
	modelDir := flag.String("model", ".", "directory containing model files")
	outDir := flag.String("output", ".", "directory to save generated files")
	modulePath := flag.String("module", "", "module path (e.g., github.com/user/project)")
	packagePath := flag.String("package", "", "package path relative to module (e.g., models)")
	flag.Parse()

	models, err := generator.ParseModels(*modelDir)
	if err != nil {
		log.Fatalf("failed to parse models: %v", err)
	}

	// Set module and package paths for each model
	for i := range models {
		models[i].ModulePath = *modulePath
		models[i].PackagePath = *packagePath
	}

	for _, m := range models {
		fmt.Printf("Generating schema for %s...\n", m.ModelName)
		if err := generator.GenerateFile(m, *outDir); err != nil {
			log.Fatalf("failed to generate file for %s: %v", m.ModelName, err)
		}
	}
	fmt.Println("Done.")
}
