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
    if !isValidScope(scope) {
        fmt.Fprintf(os.Stderr, "Error: Invalid scope option '%s'. Use 'system', 'pathuni', or 'full'\n", scope)
        os.Exit(1)
    }
    if !isValidPrune(prune) {
        fmt.Fprintf(os.Stderr, "Error: Invalid prune option '%s'. Use 'none', 'pathuni', 'system', or 'all'\n", prune)
        os.Exit(1)
    }

	var paths []string
	var err error

    switch scope {
    case "system":
        paths, err = resolveSystemPaths()
        if err == nil && (prune == "system" || prune == "all") {
            paths = filterExisting(paths)
        }
    case "pathuni":
        paths, err = resolvePathuniPaths()
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

func isValidPrune(p string) bool {
    switch p {
    case "none", "pathuni", "system", "all":
        return true
    default:
        return false
    }
}

func getCurrentPath() ([]string, error) {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return []string{}, nil
	}
	return strings.Split(pathEnv, ":"), nil
}

func getAllPathsWithTagFiltering() ([]string, error) {
    // Resolve both sources with internal dedupe
    systemPaths, err := resolveSystemPaths()
    if err != nil {
        return nil, err
    }

    pathuniPaths, err := resolvePathuniPaths()
    if err != nil {
        return nil, err
    }

    // Apply prune to system side if requested
    if prune == "system" || prune == "all" {
        systemPaths = filterExisting(systemPaths)
    }

    // Merge with pathuni-first precedence to align with init
    return mergeFull(pathuniPaths, systemPaths, true), nil
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
