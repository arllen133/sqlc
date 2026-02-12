package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/arllen133/sqlc/cmd/sqlcli/generator"
)

func main() {
	inputDir := flag.String("i", ".", "input directory containing model files")
	outDir := flag.String("o", "", "output directory (overrides config.go)")
	modulePath := flag.String("module", "", "module path (e.g., github.com/user/project)")
	packagePath := flag.String("package", "", "package path relative to module (e.g., models)")
	recursive := flag.Bool("r", false, "recursively search subdirectories for config.go")
	flag.Parse()

	if !*recursive {
		// Single directory mode
		mod, pkg, err := resolveModuleInfo(*inputDir, *modulePath, *packagePath)
		if err != nil {
			log.Printf("warning: failed to resolve module info: %v", err)
		} else {
			if *modulePath == "" {
				*modulePath = mod
			}
			if *packagePath == "" {
				*packagePath = pkg
			}
		}
		processDir(*inputDir, *outDir, *modulePath, *packagePath)
	} else {
		// Recursive mode
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

			// Resolve module info for each directory
			mod, pkg, err := resolveModuleInfo(dir, *modulePath, *packagePath)
			if err != nil {
				log.Printf("warning: failed to resolve module info for %s: %v", dir, err)
			}

			// Use resolved values if flags are empty, otherwise use flags
			effMod := *modulePath
			if effMod == "" {
				effMod = mod
			}
			effPkg := *packagePath
			if effPkg == "" {
				effPkg = pkg
			}

			processDir(dir, *outDir, effMod, effPkg)
		}
	}

	fmt.Println("Done.")
}

// resolveModuleInfo attempts to determine the module path and package path
// by looking for go.mod in parent directories.
func resolveModuleInfo(dir, flagModule, flagPackage string) (string, string, error) {
	// If both flags are provided, no need to resolve
	if flagModule != "" && flagPackage != "" {
		return flagModule, flagPackage, nil
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}

	// Find go.mod
	modFile, err := findGoMod(absDir)
	if err != nil {
		return "", "", err
	}

	// Parse module name from go.mod
	modName, err := parseModuleName(modFile)
	if err != nil {
		return "", "", err
	}

	// Calculate relative package path
	modDir := filepath.Dir(modFile)
	relPath, err := filepath.Rel(modDir, absDir)
	if err != nil {
		return "", "", err
	}

	// Clean up relPath (e.g. "." -> "")
	if relPath == "." {
		relPath = ""
	}

	return modName, relPath, nil
}

// findGoMod searches for go.mod starting from startDir upwards
func findGoMod(startDir string) (string, error) {
	dir := startDir
	for {
		f := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(f); err == nil {
			return f, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

// parseModuleName reads the module name from go.mod file
func parseModuleName(modFile string) (string, error) {
	content, err := os.ReadFile(modFile)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("module name not found in %s", modFile)
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
