package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// extractPathEntries converts []interface{} to []PathEntry structs with both paths and tags
func extractPathEntries(paths []interface{}, context string) ([]PathEntry, error) {
	var result []PathEntry
	for i, item := range paths {
		switch v := item.(type) {
		case string:
			// Plain string path - no tags
			result = append(result, PathEntry{Path: v, Tags: nil})
		case map[string]interface{}:
			// PathEntry format
			pathStr, hasPath := v["path"].(string)
			if !hasPath {
				return nil, fmt.Errorf("missing 'path' field in %s at index %d", context, i)
			}
			
			var tags []string
			if tagsInterface, hasTags := v["tags"]; hasTags {
				if tagsSlice, ok := tagsInterface.([]interface{}); ok {
					for _, tagInterface := range tagsSlice {
						if tagStr, ok := tagInterface.(string); ok {
							tags = append(tags, tagStr)
						} else {
							return nil, fmt.Errorf("invalid tag type in %s at index %d: expected string", context, i)
						}
					}
					
					// Validate the tags
					pathContext := fmt.Sprintf("%s at index %d (path: %s)", context, i, pathStr)
					if err := validateTags(tags, pathContext); err != nil {
						return nil, err
					}
				} else {
					return nil, fmt.Errorf("invalid tags format in %s at index %d: expected array", context, i)
				}
			}
			
			result = append(result, PathEntry{Path: pathStr, Tags: tags})
		default:
			return nil, fmt.Errorf("invalid path entry in %s at index %d: expected string or object", context, i)
		}
	}
	return result, nil
}

type ShellConfig struct {
	IncludeSystemPaths bool `yaml:"include_system_paths,omitempty"`
}

type PathEntry struct {
	Path string   `yaml:"path"`
	Tags []string `yaml:"tags,omitempty"`
}

type PlatformConfig struct {
	Paths      []interface{}           `yaml:"paths,omitempty"` // Can be string or PathEntry
	PowerShell *ShellConfig            `yaml:"powershell,omitempty"`
}

type Config struct {
	All   PlatformConfig   `yaml:"all,omitempty"`
	Linux PlatformConfig `yaml:"linux,omitempty"`
	MacOS PlatformConfig `yaml:"macos,omitempty"`
}

func collectValidPaths(configPath, platform, shell string, platformOnly bool, tagFilter TagFilter) ([]string, int, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, 0, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, 0, err
	}

	var rawPaths []string
	var totalSystemPaths int
	
	// Add All section paths unless platform-only is specified
	if !platformOnly {
		allEntries, err := extractPathEntries(cfg.All.Paths, "all section")
		if err != nil {
			return nil, 0, err
		}
		for _, entry := range allEntries {
			if shouldIncludePath(entry.Tags, tagFilter) {
				rawPaths = append(rawPaths, entry.Path)
			}
		}
	}
	
	// Get platform-specific paths
	switch platform {
	case "Linux":
		linuxEntries, err := extractPathEntries(cfg.Linux.Paths, "linux section")
		if err != nil {
			return nil, 0, err
		}
		for _, entry := range linuxEntries {
			if shouldIncludePath(entry.Tags, tagFilter) {
				rawPaths = append(rawPaths, entry.Path)
			}
		}
		shellPaths := getShellSpecificPaths(shell, cfg.Linux)
		rawPaths = append(rawPaths, shellPaths...)
		totalSystemPaths += countValidSystemPaths(shell, cfg.Linux)
	case "macOS":
		macosEntries, err := extractPathEntries(cfg.MacOS.Paths, "macos section")
		if err != nil {
			return nil, 0, err
		}
		for _, entry := range macosEntries {
			if shouldIncludePath(entry.Tags, tagFilter) {
				rawPaths = append(rawPaths, entry.Path)
			}
		}
		shellPaths := getShellSpecificPaths(shell, cfg.MacOS)
		rawPaths = append(rawPaths, shellPaths...)
		totalSystemPaths += countValidSystemPaths(shell, cfg.MacOS)
	}

	var paths []string
	for _, line := range rawPaths {
		expanded := os.ExpandEnv(line)
		if info, err := os.Stat(expanded); err == nil && info.IsDir() {
			paths = append(paths, expanded)
		}
	}
	return paths, totalSystemPaths, nil
}


// PathStatus represents the status of a path after evaluation
type PathStatus struct {
	Path   string
	Tags   []string
	Exists bool
	Included bool // true if should be included after tag filtering
}

// EvaluateConfigDetailed returns detailed path status for improved dry-run output
func EvaluateConfigDetailed(configPath, platform, shell string, platformOnly bool, tagFilter TagFilter) (pathStatuses []PathStatus, systemPathsCount int, err error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, 0, fmt.Errorf("failed to parse yaml: %w", err)
	}

	var allEntries []PathEntry
	var totalSystemPaths int
	
	// Add All section paths unless platform-only is specified
	if !platformOnly {
		entries, pathErr := extractPathEntries(cfg.All.Paths, "all section")
		if pathErr != nil {
			return nil, 0, fmt.Errorf("failed to parse config: %w", pathErr)
		}
		allEntries = append(allEntries, entries...)
	}
	
	// Get platform-specific paths
	switch platform {
	case "Linux":
		entries, pathErr := extractPathEntries(cfg.Linux.Paths, "linux section")
		if pathErr != nil {
			return nil, 0, fmt.Errorf("failed to parse config: %w", pathErr)
		}
		allEntries = append(allEntries, entries...)
		
		// Add shell-specific paths (these are always untagged and included)
		shellPaths := getShellSpecificPaths(shell, cfg.Linux)
		for _, shellPath := range shellPaths {
			allEntries = append(allEntries, PathEntry{Path: shellPath, Tags: nil})
		}
		totalSystemPaths += countValidSystemPaths(shell, cfg.Linux)
		
	case "macOS":
		entries, pathErr := extractPathEntries(cfg.MacOS.Paths, "macos section")
		if pathErr != nil {
			return nil, 0, fmt.Errorf("failed to parse config: %w", pathErr)
		}
		allEntries = append(allEntries, entries...)
		
		// Add shell-specific paths (these are always untagged and included)
		shellPaths := getShellSpecificPaths(shell, cfg.MacOS)
		for _, shellPath := range shellPaths {
			allEntries = append(allEntries, PathEntry{Path: shellPath, Tags: nil})
		}
		totalSystemPaths += countValidSystemPaths(shell, cfg.MacOS)
	}

	// Process each path: check existence first, then filtering
	for _, entry := range allEntries {
		expanded := os.ExpandEnv(entry.Path)
		resolved := filepath.Clean(expanded)
		
		// Check existence first (existence wins over filtering)
		info, err := os.Stat(resolved)
		exists := (err == nil && info.IsDir())
		
		// Only apply tag filtering if path exists
		included := exists && shouldIncludePath(entry.Tags, tagFilter)
		
		pathStatuses = append(pathStatuses, PathStatus{
			Path:     resolved,
			Tags:     entry.Tags,
			Exists:   exists,
			Included: included,
		})
	}
	
	return pathStatuses, totalSystemPaths, nil
}

func EvaluateConfig(configPath, platform, shell string, platformOnly bool, tagFilter TagFilter) (validPaths []string, skippedPaths []string, systemPathsCount int, err error) {
	pathStatuses, systemPathsCount, err := EvaluateConfigDetailed(configPath, platform, shell, platformOnly, tagFilter)
	if err != nil {
		return nil, nil, 0, err
	}
	
	// Convert detailed results back to simple format for backwards compatibility
	for _, status := range pathStatuses {
		if status.Included {
			validPaths = append(validPaths, status.Path)
		} else {
			skippedPaths = append(skippedPaths, status.Path)
		}
	}
	
	return validPaths, skippedPaths, systemPathsCount, nil
}

func PrintEvaluationReport(configPath, platform, shell string, inferred bool, platformOnly bool) error {
	// Parse tag filters
	tagFilter, err := parseTagFlags(tagsInclude, tagsExclude)
	if err != nil {
		return err
	}
	
	pathStatuses, systemPathsCount, err := EvaluateConfigDetailed(configPath, platform, shell, platformOnly, tagFilter)
	if err != nil {
		return err
	}

	// Separate paths into categories based on existence and filtering
	var included []string
	var notFound []string
	var filteredByTags []string
	
	for _, status := range pathStatuses {
		if status.Included {
			included = append(included, status.Path)
		} else if !status.Exists {
			// Existence wins over filtering
			notFound = append(notFound, status.Path)
		} else {
			// Path exists but was filtered by tags
			filteredByTags = append(filteredByTags, status.Path)
		}
	}

	// Print header
	fmt.Printf("Evaluating: %s\n\n", configPath)
	fmt.Printf("OS    : %s\n", platform)
	label := "specified"
	if inferred {
		label = "inferred"
	}
	fmt.Printf("Shell : %s (%s)\n\n", shell, label)

	// Print included paths
	if len(included) > 0 {
		if len(included) == 1 {
			fmt.Println("1 Included Path:")
		} else {
			fmt.Printf("%d Included Paths:\n", len(included))
		}
		for _, p := range included {
			fmt.Printf("  [+] %s\n", p)
		}
		fmt.Println()
	}

	// Print not found paths
	if len(notFound) > 0 {
		if len(notFound) == 1 {
			fmt.Println("1 Skipped Path (not found):")
		} else {
			fmt.Printf("%d Skipped Paths (not found):\n", len(notFound))
		}
		for _, p := range notFound {
			fmt.Printf("  [!] %s\n", p)
		}
		fmt.Println()
	}

	// Print filtered paths
	if len(filteredByTags) > 0 {
		if len(filteredByTags) == 1 {
			fmt.Println("1 Skipped Path (filtered by tags):")
		} else {
			fmt.Printf("%d Skipped Paths (filtered by tags):\n", len(filteredByTags))
		}
		for _, p := range filteredByTags {
			fmt.Printf("  [-] %s\n", p)
		}
		fmt.Println()
	}

	// Print summary
	totalSkipped := len(notFound) + len(filteredByTags)
	if len(included) == 1 {
		fmt.Printf("1 path included in total\n")
	} else {
		fmt.Printf("%d paths included in total\n", len(included))
	}
	
	if systemPathsCount > 0 {
		fmt.Printf("* Including %d system paths due to include_system_paths setting\n", systemPathsCount)
	}
	
	if totalSkipped == 1 {
		fmt.Printf("1 skipped in total")
	} else {
		fmt.Printf("%d skipped in total", totalSkipped)
	}
	
	if platformOnly {
		fmt.Printf(" *\n* Not including paths from 'All' section due to --platform-only\n")
	} else {
		fmt.Printf("\n")
	}
	fmt.Printf("\n")
	
	// Print shell output
	if len(included) > 0 {
		fmt.Println("Output would be:")
		fmt.Printf("  %s\n", renderers[shell](included))
	}
	
	return nil
}

func runDryRun() {
	configPath := getConfigPath()
	osName := getOSName()
	shellName, inferred := getShellName()

	if !shellIsValid(shellName) {
		fmt.Fprintf(os.Stderr, "Unsupported shell '%s'. Supported shells: %s\n", shellName, strings.Join(shellNames(), ", "))
		os.Exit(1)
	}

	err := PrintEvaluationReport(configPath, osName, shellName, inferred, platformOnly)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}