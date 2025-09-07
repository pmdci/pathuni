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
			
			var tags []string = nil  // Explicitly nil for inheritance
			if tagsInterface, hasTags := v["tags"]; hasTags {
				// Tags field exists (could be empty array or populated)
				if tagsSlice, ok := tagsInterface.([]interface{}); ok {
					tags = []string{}  // Initialize empty slice for explicit tags field
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
			// If no tags field exists, tags remains nil (for inheritance)
			
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

// GetEffectiveTags returns the effective tags for this path entry.
// If Tags field is explicitly set (even if empty), returns Tags.
// Otherwise, inherits platformTags.
func (pe *PathEntry) GetEffectiveTags(platformTags []string) []string {
	if pe.Tags != nil {  // Explicit tags field present (even if empty)
		return pe.Tags
	}
	return platformTags  // Inherit platform tags
}

// IsExplicitlyTagged returns true if this path entry has an explicit tags field,
// false if it should inherit platform tags. This is used to distinguish between
// untagged paths (immune to filtering) and explicitly empty-tagged paths.
func (pe *PathEntry) IsExplicitlyTagged() bool {
	return pe.Tags != nil
}

// validateConfig validates platform-level tags and other config constraints
func validateConfig(cfg *Config) error {
	// Validate platform-level tags for All section
	if err := validateTags(cfg.All.Tags, "all.tags"); err != nil {
		return err
	}
	
	// Validate platform-level tags for Linux section
	if err := validateTags(cfg.Linux.Tags, "linux.tags"); err != nil {
		return err
	}
	
	// Validate platform-level tags for macOS section
	if err := validateTags(cfg.MacOS.Tags, "macos.tags"); err != nil {
		return err
	}
	
	return nil
}

// SkipReason represents why a path was skipped in dry-run output
type SkipReason struct {
	Type   string // "tags", "hostname", "not_found"
	Detail string // "gaming = gaming", "mac,gaming (+1) != essential"
}

// SkippedPath represents a path that was skipped with reasons
type SkippedPath struct {
	Path    string
	Reasons []SkipReason
}

// EvaluationResult represents the comprehensive result of path evaluation for dry-run
type EvaluationResult struct {
	IncludedPaths []string      // Paths that will be included in PATH
	SkippedPaths  []SkippedPath // Paths that were skipped with reasons
	TotalPaths    int           // Total paths processed
}

// EvaluateConfigWithReasons returns detailed evaluation results with skip reasons for dry-run v2
func EvaluateConfigWithReasons(configPath, platform, shell string, tagFilter TagFilter) (*EvaluationResult, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}
	
	// Validate platform-level tags
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation error: %v", err)
	}

	result := &EvaluationResult{
		IncludedPaths: []string{},
		SkippedPaths:  []SkippedPath{},
		TotalPaths:    0,
	}

	// Helper function to process platform paths
	processPlatformPaths := func(platformConfig PlatformConfig, platformName string) error {
		entries, err := extractPathEntries(platformConfig.Paths, fmt.Sprintf("%s.paths", platformName))
		if err != nil {
			return err
		}

		for _, entry := range entries {
			result.TotalPaths++
			
			// Check if path exists
			if _, err := os.Stat(os.ExpandEnv(entry.Path)); os.IsNotExist(err) {
				result.SkippedPaths = append(result.SkippedPaths, SkippedPath{
					Path:    os.ExpandEnv(entry.Path),
					Reasons: []SkipReason{{Type: "not_found", Detail: "not found"}},
				})
				continue
			}

			// Check tag filtering using effective tags (with platform inheritance)
			effectiveTags := entry.GetEffectiveTags(platformConfig.Tags)
			if skipReasons := getPathSkipReasons(effectiveTags, entry.IsExplicitlyTagged(), tagFilter); skipReasons != nil {
				result.SkippedPaths = append(result.SkippedPaths, SkippedPath{
					Path:    os.ExpandEnv(entry.Path),
					Reasons: skipReasons,
				})
			} else {
				result.IncludedPaths = append(result.IncludedPaths, os.ExpandEnv(entry.Path))
			}
		}
		return nil
	}

	// Process "all" platform
	if err := processPlatformPaths(config.All, "all"); err != nil {
		return nil, err
	}

	// Process platform-specific paths
	switch platform {
	case "macOS":
		if err := processPlatformPaths(config.MacOS, "macos"); err != nil {
			return nil, err
		}
	case "Linux":
		if err := processPlatformPaths(config.Linux, "linux"); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// renderSkippedPath renders a single skipped path with tree structure for reasons
func renderSkippedPath(skipped SkippedPath) string {
	// Special case: not_found gets single-line format
	if len(skipped.Reasons) == 1 && skipped.Reasons[0].Type == "not_found" {
		return fmt.Sprintf("  [!] %s (not found)", skipped.Path)
	}
	
	// Determine icon character based on reason type
	iconChar := "-"
	if len(skipped.Reasons) > 0 && skipped.Reasons[0].Type == "not_found" {
		iconChar = "!"
	}
	
	var result strings.Builder
	result.WriteString(fmt.Sprintf("  [%s] %s\n", iconChar, skipped.Path))
	
	for i, reason := range skipped.Reasons {
		connector := "├"
		if i == len(skipped.Reasons)-1 {
			connector = "└"
		}
		result.WriteString(fmt.Sprintf("       %s%s\n", connector, reason.Detail))
	}
	
	return strings.TrimSuffix(result.String(), "\n")
}

// PrintEvaluationReportV2 prints the enhanced dry-run output with tree structure
func PrintEvaluationReportV2(configPath, platform, shell string, osInferred, shellInferred bool) error {
	// Parse tag filters  
	tagFilter, err := parseTagFlags(tagsInclude, tagsExclude)
	if err != nil {
		return err
	}
	
	result, err := EvaluateConfigWithReasons(configPath, platform, shell, tagFilter)
	if err != nil {
		return err
	}
	
	// Print header
	fmt.Printf("Evaluating: %s\n\n", configPath)
	
	// Print system info
	if osInferred {
		fmt.Printf("OS    : %s (detected)\n", platform)
	} else {
		fmt.Printf("OS    : %s (specified)\n", platform)
	}
	if shellInferred {
		fmt.Printf("Shell : %s (detected)\n", shell)
	} else {
		fmt.Printf("Shell : %s (specified)\n", shell)
	}
	
	fmt.Printf("\n")
	
	// Print included paths
	includedCount := len(result.IncludedPaths)
	if includedCount > 0 {
		fmt.Printf("%d Included Path", includedCount)
		if includedCount != 1 {
			fmt.Printf("s")
		}
		fmt.Printf(":\n")
		
		for _, path := range result.IncludedPaths {
			fmt.Printf("  [+] %s\n", path)
		}
		fmt.Printf("\n")
	}
	
	// Print skipped paths with tree structure
	skippedCount := len(result.SkippedPaths)
	if skippedCount > 0 {
		fmt.Printf("%d Skipped Path", skippedCount)
		if skippedCount != 1 {
			fmt.Printf("s")
		}
		fmt.Printf(":\n")
		
		for _, skipped := range result.SkippedPaths {
			fmt.Printf("%s\n", renderSkippedPath(skipped))
		}
		fmt.Printf("\n")
	}
	
	// Print summary
	fmt.Printf("%d paths included in total\n", includedCount)
	fmt.Printf("%d skipped in total\n", skippedCount)
	
	return nil
}

// includedEntry represents an included path along with its origin for dry-run
// scope handling. Origin should be "pathuni" or "system".
type includedEntry struct {
    Path   string
    Origin string
}

// PrintDryRunReport prints dry-run output respecting the global scope flag.
// It uses pathuni-first precedence when combining sources under scope=full.
func PrintDryRunReport(configPath, platform, shell string, osInferred, shellInferred bool, scope string) error {
    // Header
    fmt.Printf("Evaluating: %s\n\n", configPath)
    if osInferred {
        fmt.Printf("OS    : %s (detected)\n", platform)
    } else {
        fmt.Printf("OS    : %s (specified)\n", platform)
    }
    if shellInferred {
        fmt.Printf("Shell : %s (detected)\n", shell)
    } else {
        fmt.Printf("Shell : %s (specified)\n", shell)
    }
    fmt.Printf("Flags : scope=%s, prune=%s\n\n", scope, prune)

    // Parse tag filters
    tagFilter, err := parseTagFlags(tagsInclude, tagsExclude)
    if err != nil {
        return err
    }

    switch scope {
    case "pathuni":
        // Build included respecting prune for pathuni
        statuses, _, err := EvaluateConfigDetailed(configPath, platform, shell, tagFilter)
        if err != nil { return err }
        var includedPU []string
        if prune == "pathuni" || prune == "all" {
            for _, st := range statuses { if st.Included { includedPU = append(includedPU, st.Path) } }
        } else {
            for _, st := range statuses { if st.PassesFilter { includedPU = append(includedPU, st.Path) } }
        }
        if len(includedPU) > 0 {
            if len(includedPU) == 1 { fmt.Printf("1 Included Path:\n") } else { fmt.Printf("%d Included Paths:\n", len(includedPU)) }
            for _, p := range includedPU { fmt.Printf("  [+] %s\n", p) }
            fmt.Printf("\n")
        }
        // Show pathuni skipped reasons only when pruning pathuni side
        skippedTotal := 0
        if prune == "pathuni" || prune == "all" {
            result, err := EvaluateConfigWithReasons(configPath, platform, shell, tagFilter)
            if err != nil { return err }
            skippedTotal = len(result.SkippedPaths)
            if skippedTotal > 0 {
                if skippedTotal == 1 { fmt.Printf("1 Skipped Path:\n") } else { fmt.Printf("%d Skipped Paths:\n", skippedTotal) }
                for _, skipped := range result.SkippedPaths { fmt.Printf("%s\n", renderSkippedPath(skipped)) }
                fmt.Printf("\n")
            }
        }
        // Included summary (pathuni only) printed at the end
        if len(includedPU) == 1 {
            fmt.Printf("1 Pathuni path included in total\n")
        } else {
            fmt.Printf("%d Pathuni paths included in total\n", len(includedPU))
        }
        // Skipped summary (pathuni only when pruning pathuni)
        if skippedTotal == 0 {
            fmt.Printf("0 Skipped paths\n")
        } else if skippedTotal == 1 {
            fmt.Printf("1 Pathuni path skipped in total\n")
        } else {
            fmt.Printf("%d Pathuni paths skipped in total\n", skippedTotal)
        }
        return nil
    case "system":
        sys, err := resolveSystemPaths()
        if err != nil { return err }
        original := append([]string{}, sys...)
        var skippedSys []string
        if prune == "system" || prune == "all" {
            filtered := filterExisting(sys)
            m := make(map[string]bool); for _, p := range filtered { m[p] = true }
            for _, p := range original { if !m[p] { skippedSys = append(skippedSys, p) } }
            sys = filtered
        }
        if len(sys) > 0 {
            if len(sys) == 1 { fmt.Printf("1 Included Path:\n") } else { fmt.Printf("%d Included Paths:\n", len(sys)) }
            for _, p := range sys { fmt.Printf("  [.] %s\n", p) }
            fmt.Printf("\n")
        }
        // Print skipped block first (details), then summaries at the end
        if len(skippedSys) > 0 {
            if len(skippedSys) == 1 { fmt.Printf("1 Skipped Path:\n") } else { fmt.Printf("%d Skipped Paths:\n", len(skippedSys)) }
            for _, p := range skippedSys { fmt.Printf("  [?] %s (not found)\n", p) }
            fmt.Printf("\n")
        }
        // Summaries at the end
        if len(sys) == 1 { fmt.Printf("1 System path included in total\n") } else { fmt.Printf("%d System paths included in total\n", len(sys)) }
        if len(skippedSys) == 0 {
            fmt.Printf("0 Skipped paths\n")
        } else if len(skippedSys) == 1 {
            fmt.Printf("1 System path skipped in total\n")
        } else {
            fmt.Printf("%d System paths skipped in total\n", len(skippedSys))
        }
        return nil
    case "full":
        statuses, _, err := EvaluateConfigDetailed(configPath, platform, shell, tagFilter)
        if err != nil { return err }
        result, err := EvaluateConfigWithReasons(configPath, platform, shell, tagFilter)
        if err != nil { return err }
        sys, err := resolveSystemPaths()
        if err != nil { return err }
        originalSys := append([]string{}, sys...)
        var skippedSys []string
        if prune == "system" || prune == "all" {
            filtered := filterExisting(sys)
            m := make(map[string]bool); for _, p := range filtered { m[p] = true }
            for _, p := range originalSys { if !m[p] { skippedSys = append(skippedSys, p) } }
            sys = filtered
        }
        seen := make(map[string]bool)
        var included []includedEntry
        if prune == "pathuni" || prune == "all" {
            for _, st := range statuses { if st.Included { if !seen[st.Path] { seen[st.Path] = true; included = append(included, includedEntry{Path: st.Path, Origin: "pathuni"}) } } }
        } else {
            for _, st := range statuses { if st.PassesFilter { if !seen[st.Path] { seen[st.Path] = true; included = append(included, includedEntry{Path: st.Path, Origin: "pathuni"}) } } }
        }
        for _, p := range sys { if !seen[p] { seen[p] = true; included = append(included, includedEntry{Path: p, Origin: "system"}) } }
        if len(included) > 0 {
            if len(included) == 1 {
                fmt.Printf("1 Included Path:\n")
            } else {
                fmt.Printf("%d Included Paths:\n", len(included))
            }
            for _, e := range included {
                marker := "."
                if e.Origin == "pathuni" { marker = "+" }
                fmt.Printf("  [%s] %s\n", marker, e.Path)
            }
            fmt.Printf("\n")
        }
        // Include pathuni skipped reasons only when pruning pathuni
        var pathuniSkipped []SkippedPath
        if prune == "pathuni" || prune == "all" {
            pathuniSkipped = result.SkippedPaths
        } else {
            pathuniSkipped = nil
        }
        skippedCount := len(pathuniSkipped) + len(skippedSys)
        if skippedCount > 0 {
            if skippedCount == 1 { fmt.Printf("1 Skipped Path:\n") } else { fmt.Printf("%d Skipped Paths:\n", skippedCount) }
            for _, skipped := range pathuniSkipped { fmt.Printf("%s\n", renderSkippedPath(skipped)) }
            for _, p := range skippedSys { fmt.Printf("  [?] %s (not found)\n", p) }
            fmt.Printf("\n")
        }
        var pathuniCount, systemCount int
        for _, e := range included { if e.Origin == "pathuni" { pathuniCount++ } else { systemCount++ } }
        // Included summary: single-line when only one origin present; tree when both
        if pathuniCount > 0 && systemCount > 0 {
            fmt.Printf("%d Paths included in total\n", len(included))
            fmt.Printf("  ├ %d Pathuni path", pathuniCount); if pathuniCount != 1 { fmt.Printf("s") }; fmt.Printf("\n")
            fmt.Printf("  └ %d System path", systemCount); if systemCount != 1 { fmt.Printf("s") }; fmt.Printf("\n")
        } else if pathuniCount > 0 {
            if pathuniCount == 1 { fmt.Printf("1 Pathuni path included in total\n") } else { fmt.Printf("%d Pathuni paths included in total\n", pathuniCount) }
        } else {
            if systemCount == 1 { fmt.Printf("1 System path included in total\n") } else { fmt.Printf("%d System paths included in total\n", systemCount) }
        }
        totalSkipped := skippedCount
        puCount := len(pathuniSkipped)
        sysCount := len(skippedSys)
        if totalSkipped == 0 {
            fmt.Printf("0 Skipped paths\n")
        } else if puCount > 0 && sysCount > 0 {
            if totalSkipped == 1 { fmt.Printf("1 Path skipped in total\n") } else { fmt.Printf("%d Paths skipped in total\n", totalSkipped) }
            fmt.Printf("  ├ %d Pathuni path", puCount); if puCount != 1 { fmt.Printf("s") }; fmt.Printf("\n")
            fmt.Printf("  └ %d System path", sysCount); if sysCount != 1 { fmt.Printf("s") }; fmt.Printf("\n")
        } else if puCount > 0 { // only pathuni skipped
            if puCount == 1 { fmt.Printf("1 Pathuni path skipped in total\n") } else { fmt.Printf("%d Pathuni paths skipped in total\n", puCount) }
        } else { // only system skipped
            if sysCount == 1 { fmt.Printf("1 System path skipped in total\n") } else { fmt.Printf("%d System paths skipped in total\n", sysCount) }
        }
        return nil
    }
    return fmt.Errorf("invalid scope: %s", scope)
}

type PlatformConfig struct {
	Tags       []string                `yaml:"tags,omitempty"`      // Platform-level tags for inheritance
	Paths      []interface{}           `yaml:"paths,omitempty"`     // Can be string or PathEntry
	PowerShell *ShellConfig            `yaml:"powershell,omitempty"`
}

type Config struct {
	All   PlatformConfig   `yaml:"all,omitempty"`
	Linux PlatformConfig `yaml:"linux,omitempty"`
	MacOS PlatformConfig `yaml:"macos,omitempty"`
}

func collectValidPaths(configPath, platform, shell string, tagFilter TagFilter) ([]string, int, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, 0, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, 0, err
	}
	
	// Validate platform-level tags
	if err := validateConfig(&cfg); err != nil {
		return nil, 0, fmt.Errorf("config validation error: %v", err)
	}

	var rawPaths []string
	var totalSystemPaths int
	
	// Add All section paths
	allEntries, err := extractPathEntries(cfg.All.Paths, "all section")
	if err != nil {
		return nil, 0, err
	}
	for _, entry := range allEntries {
		effectiveTags := entry.GetEffectiveTags(cfg.All.Tags)
		if shouldIncludePath(effectiveTags, entry.IsExplicitlyTagged(), tagFilter) {
			rawPaths = append(rawPaths, entry.Path)
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
			effectiveTags := entry.GetEffectiveTags(cfg.Linux.Tags)
			if shouldIncludePath(effectiveTags, entry.IsExplicitlyTagged(), tagFilter) {
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
			effectiveTags := entry.GetEffectiveTags(cfg.MacOS.Tags)
			if shouldIncludePath(effectiveTags, entry.IsExplicitlyTagged(), tagFilter) {
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
    // PassesFilter indicates whether tag filtering passes regardless of existence
    PassesFilter bool
}

// EvaluateConfigDetailed returns detailed path status for improved dry-run output
func EvaluateConfigDetailed(configPath, platform, shell string, tagFilter TagFilter) (pathStatuses []PathStatus, systemPathsCount int, err error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, 0, fmt.Errorf("failed to parse yaml: %w", err)
	}
	
	// Validate platform-level tags
	if err := validateConfig(&cfg); err != nil {
		return nil, 0, fmt.Errorf("config validation error: %w", err)
	}

	var totalSystemPaths int
	
	// Helper function to process entries with platform tags
	processEntries := func(entries []PathEntry, platformTags []string) {
		for _, entry := range entries {
			expanded := os.ExpandEnv(entry.Path)
			resolved := filepath.Clean(expanded)
			
            // Check existence
            info, err := os.Stat(resolved)
            exists := (err == nil && info.IsDir())
            
            // Get effective tags (with platform inheritance)
            effectiveTags := entry.GetEffectiveTags(platformTags)
            
            // Evaluate tag filtering independently from existence
            passes := shouldIncludePath(effectiveTags, entry.IsExplicitlyTagged(), tagFilter)
            included := exists && passes
            
            pathStatuses = append(pathStatuses, PathStatus{
                Path:     resolved,
                Tags:     effectiveTags,  // Store effective tags, not original
                Exists:   exists,
                Included: included,
                PassesFilter: passes,
            })
		}
	}
	
	// Add All section paths
	entries, pathErr := extractPathEntries(cfg.All.Paths, "all section")
	if pathErr != nil {
		return nil, 0, fmt.Errorf("failed to parse config: %w", pathErr)
	}
	processEntries(entries, cfg.All.Tags)
	
	// Get platform-specific paths
	switch platform {
	case "Linux":
		entries, pathErr := extractPathEntries(cfg.Linux.Paths, "linux section")
		if pathErr != nil {
			return nil, 0, fmt.Errorf("failed to parse config: %w", pathErr)
		}
		processEntries(entries, cfg.Linux.Tags)
		
		// Add shell-specific paths (these are always untagged and included)
		shellPaths := getShellSpecificPaths(shell, cfg.Linux)
		shellEntries := make([]PathEntry, len(shellPaths))
		for i, shellPath := range shellPaths {
			shellEntries[i] = PathEntry{Path: shellPath, Tags: nil}
		}
		processEntries(shellEntries, cfg.Linux.Tags)
		totalSystemPaths += countValidSystemPaths(shell, cfg.Linux)
		
	case "macOS":
		entries, pathErr := extractPathEntries(cfg.MacOS.Paths, "macos section")
		if pathErr != nil {
			return nil, 0, fmt.Errorf("failed to parse config: %w", pathErr)
		}
		processEntries(entries, cfg.MacOS.Tags)
		
		// Add shell-specific paths (these are always untagged and included)
		shellPaths := getShellSpecificPaths(shell, cfg.MacOS)
		shellEntries := make([]PathEntry, len(shellPaths))
		for i, shellPath := range shellPaths {
			shellEntries[i] = PathEntry{Path: shellPath, Tags: nil}
		}
		processEntries(shellEntries, cfg.MacOS.Tags)
		totalSystemPaths += countValidSystemPaths(shell, cfg.MacOS)
	}
	
	return pathStatuses, totalSystemPaths, nil
}

func EvaluateConfig(configPath, platform, shell string, tagFilter TagFilter) (validPaths []string, skippedPaths []string, systemPathsCount int, err error) {
	pathStatuses, systemPathsCount, err := EvaluateConfigDetailed(configPath, platform, shell, tagFilter)
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

func PrintEvaluationReport(configPath, platform, shell string, inferred bool) error {
	// Parse tag filters
	tagFilter, err := parseTagFlags(tagsInclude, tagsExclude)
	if err != nil {
		return err
	}
	
	pathStatuses, systemPathsCount, err := EvaluateConfigDetailed(configPath, platform, shell, tagFilter)
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
		label = "detected"
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
	
	fmt.Printf("\n")
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
    osName, osInferred := getOSName()
    shellName, shellInferred := getShellName()

	if !osIsValid(osName) {
		fmt.Fprintf(os.Stderr, "Unsupported OS '%s'. Supported OS: %s\n", osName, strings.Join(osNames(), ", "))
		os.Exit(1)
	}

	if !shellIsValid(shellName) {
		fmt.Fprintf(os.Stderr, "Unsupported shell '%s'. Supported shells: %s\n", shellName, strings.Join(shellNames(), ", "))
		os.Exit(1)
	}

    err := PrintDryRunReport(configPath, osName, shellName, osInferred, shellInferred, scope)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
