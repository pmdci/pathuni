package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func runDump() {
	if !isValidFormat(dumpFormat) {
		fmt.Fprintf(os.Stderr, "Error: Unsupported format '%s'. Supported formats: plain, json, yaml\n", dumpFormat)
		os.Exit(1)
	}
	if !isValidScope(dumpScope) {
		fmt.Fprintf(os.Stderr, "Error: Invalid scope option '%s'. Use 'system', 'pathuni', or 'full'\n", dumpScope)
		os.Exit(1)
	}

	var paths []string
	var err error

	switch dumpScope {
	case "system":
		paths, err = getCurrentPath()
	case "pathuni":
		paths, err = getPathUniPaths()
	case "full":
		paths, err = getAllPathsWithTagFiltering()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	output, err := formatPaths(paths, dumpFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(output)
}

func isValidFormat(format string) bool {
	return format == "plain" || format == "json" || format == "yaml"
}

func isValidScope(scope string) bool {
	return scope == "system" || scope == "pathuni" || scope == "full"
}

func getCurrentPath() ([]string, error) {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return []string{}, nil
	}
	return strings.Split(pathEnv, ":"), nil
}

func getAllPathsWithTagFiltering() ([]string, error) {
	// Get current system PATH (no filtering - these paths have no tags)
	systemPaths, err := getCurrentPath()
	if err != nil {
		return nil, err
	}
	
	// Get PathUni paths with tag filtering applied
	pathuniPaths, err := getPathUniPaths()
	if err != nil {
		return nil, err
	}
	
	// Create a map to avoid duplicates while preserving order
	seen := make(map[string]bool)
	var result []string
	
	// Add system paths first
	for _, path := range systemPaths {
		if !seen[path] {
			result = append(result, path)
			seen[path] = true
		}
	}
	
	// Add PathUni paths (filtered), avoiding duplicates
	for _, path := range pathuniPaths {
		if !seen[path] {
			result = append(result, path)
			seen[path] = true
		}
	}
	
	return result, nil
}

func getPathUniPaths() ([]string, error) {
	configPath := getConfigPath()
	osName, _ := getOSName()
	shellName, _ := getShellName()
	
	// Parse tag filters
	tagFilter, err := parseTagFlags(tagsInclude, tagsExclude)
	if err != nil {
		return nil, err
	}
	
	paths, _, err := collectValidPaths(configPath, osName, shellName, tagFilter)
	return paths, err
}

func formatPaths(paths []string, format string) (string, error) {
	switch format {
	case "plain":
		return strings.Join(paths, "\n") + "\n", nil
	case "json":
		pathData := map[string][]string{"PATH": paths}
		jsonBytes, err := json.Marshal(pathData)
		if err != nil {
			return "", err
		}
		return string(jsonBytes) + "\n", nil
	case "yaml":
		pathData := map[string][]string{"PATH": paths}
		yamlBytes, err := yaml.Marshal(pathData)
		if err != nil {
			return "", err
		}
		return string(yamlBytes), nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}