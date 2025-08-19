package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	All   []string `yaml:"All"`
	Linux []string `yaml:"Linux"`
	MacOS []string `yaml:"macOS"`
}

func collectValidPaths(configPath, platform string) ([]string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	var rawPaths []string
	rawPaths = append(rawPaths, cfg.All...)
	switch platform {
	case "Linux":
		rawPaths = append(rawPaths, cfg.Linux...)
	case "macOS":
		rawPaths = append(rawPaths, cfg.MacOS...)
	}

	var paths []string
	for _, line := range rawPaths {
		expanded := os.ExpandEnv(line)
		if info, err := os.Stat(expanded); err == nil && info.IsDir() {
			paths = append(paths, expanded)
		}
	}
	return paths, nil
}

func EvaluateConfig(configPath, platform string) (validPaths []string, skippedPaths []string, err error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	var rawPaths []string
	rawPaths = append(rawPaths, cfg.All...)
	switch platform {
	case "Linux":
		rawPaths = append(rawPaths, cfg.Linux...)
	case "macOS":
		rawPaths = append(rawPaths, cfg.MacOS...)
	}

	for _, line := range rawPaths {
		expanded := os.ExpandEnv(line)
		resolved := filepath.Clean(expanded)
		info, err := os.Stat(resolved)
		if err == nil && info.IsDir() {
			validPaths = append(validPaths, resolved)
		} else {
			skippedPaths = append(skippedPaths, resolved)
		}
	}
	return validPaths, skippedPaths, nil
}

func PrintEvaluationReport(configPath, platform, shell string, inferred bool) error {
	valid, skipped, err := EvaluateConfig(configPath, platform)
	if err != nil {
		return err
	}

	fmt.Printf("Evaluating: %s\n\n", configPath)
	fmt.Printf("OS    : %s\n", platform)
	label := "specified"
	if inferred {
		label = "inferred"
	}
	fmt.Printf("Shell : %s (%s)\n\n", shell, label)

	fmt.Println("Included Paths:")
	for _, p := range valid {
		fmt.Printf("  [+] %s\n", p)
	}

	if len(skipped) > 0 {
		fmt.Println("\nSkipped (not found):")
		for _, p := range skipped {
			fmt.Printf("  [-] %s\n", p)
		}
	}

	fmt.Printf("\n%d paths included\n", len(valid))
	fmt.Printf("%d skipped\n\n", len(skipped))
	
	if len(valid) > 0 {
		fmt.Println("Output:")
		fmt.Printf("  %s\n\n", renderers[shell](valid))
	}
	
	fmt.Printf("To apply: run 'pathuni --shell=%s'\n", shell)
	return nil
}