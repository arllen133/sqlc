package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/arllen133/sqlc/cmd/sqlcli/generator"
)

func main() {
	inputDir := flag.String("i", ".", "input directory containing model files")
	outDir := flag.String("o", "", "output directory (overrides config.go)")
	modulePath := flag.String("module", "", "module path (e.g., github.com/user/project)")
	packagePath := flag.String("package", "", "package path relative to module (e.g., models)")
	recursive := flag.Bool("r", false, "recursively search subdirectories for config.go")
	flag.Parse()

	if *recursive {
		// Find all directories containing config.go
		dirs, err := findConfigDirs(*inputDir)
		if err != nil {
			log.Fatalf("failed to find config directories: %v", err)
		}

		if len(dirs) == 0 {
			fmt.Println("No config.go files found.")
			return
		}

		for _, dir := range dirs {
			fmt.Printf("\n=== Processing %s ===\n", dir)
			processDir(dir, *outDir, *modulePath, *packagePath)
		}
	} else {
		processDir(*inputDir, *outDir, *modulePath, *packagePath)
	}

	fmt.Println("Done.")
}

// findConfigDirs recursively finds all directories containing config.go
func findConfigDirs(root string) ([]string, error) {
	var dirs []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if info.Name() == "config.go" {
			dirs = append(dirs, filepath.Dir(path))
		}
		return nil
	})
	return dirs, err
}

// processDir processes a single directory
func processDir(modelDir, outDir, modulePath, packagePath string) {
	// Parse config.go for declarative configuration
	cfg, err := generator.ParseConfig(modelDir)
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	// Determine output directory: flag > config > default
	effectiveOutDir := modelDir
	if outDir != "" {
		effectiveOutDir = outDir
	} else if cfg != nil && cfg.OutPath != "" {
		// OutPath is relative to modelDir
		effectiveOutDir = filepath.Join(modelDir, cfg.OutPath)
	}

	models, err := generator.ParseModels(modelDir)
	if err != nil {
		log.Fatalf("failed to parse models: %v", err)
	}

	// Apply Include/Exclude filters from config
	if cfg != nil {
		models = filterModels(models, cfg)
	}

	// Set module and package paths for each model
	for i := range models {
		models[i].ModulePath = modulePath
		models[i].PackagePath = packagePath
		// Pass user-defined field type mappings from config
		if cfg != nil && cfg.FieldTypeMap != nil {
			models[i].FieldTypeMap = cfg.FieldTypeMap
		}
	}

	for _, m := range models {
		fmt.Printf("Generating schema for %s...\n", m.ModelName)
		if err := generator.GenerateFile(m, effectiveOutDir); err != nil {
			log.Fatalf("failed to generate file for %s: %v", m.ModelName, err)
		}
	}
}

// filterModels applies Include/Exclude filters from config
func filterModels(models []generator.ModelMeta, cfg *generator.GenConfig) []generator.ModelMeta {
	if len(cfg.IncludeStructs) == 0 && len(cfg.ExcludeStructs) == 0 {
		return models
	}

	includeSet := make(map[string]bool)
	for _, name := range cfg.IncludeStructs {
		includeSet[name] = true
	}

	excludeSet := make(map[string]bool)
	for _, name := range cfg.ExcludeStructs {
		excludeSet[name] = true
	}

	var result []generator.ModelMeta
	for _, m := range models {
		// Skip excluded
		if excludeSet[m.ModelName] {
			continue
		}
		// If include list exists, only include those
		if len(includeSet) > 0 && !includeSet[m.ModelName] {
			continue
		}
		result = append(result, m)
	}
	return result
}
