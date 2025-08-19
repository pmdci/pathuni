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
	if !isValidInclude(dumpInclude) {
		fmt.Fprintf(os.Stderr, "Error: Invalid include option '%s'. Use 'all' or 'pathuni'\n", dumpInclude)
		os.Exit(1)
	}

	var paths []string
	var err error

	if dumpInclude == "all" {
		paths, err = getCurrentPath()
	} else {
		paths, err = getPathUniPaths()
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

func isValidInclude(include string) bool {
	return include == "all" || include == "pathuni"
}

func getCurrentPath() ([]string, error) {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return []string{}, nil
	}
	return strings.Split(pathEnv, ":"), nil
}

func getPathUniPaths() ([]string, error) {
	configPath := getConfigPath()
	osName := getOSName()
	return collectValidPaths(configPath, osName, platformOnly)
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