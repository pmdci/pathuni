package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ShellConfig struct {
	IncludeSystemPaths bool `yaml:"include_system_paths,omitempty"`
}

type PlatformConfig struct {
	Paths      []string                `yaml:"paths,omitempty"`
	PowerShell *ShellConfig            `yaml:"powershell,omitempty"`
}

type Config struct {
	All   PlatformConfig   `yaml:"all,omitempty"`
	Linux PlatformConfig `yaml:"linux,omitempty"`
	MacOS PlatformConfig `yaml:"macos,omitempty"`
}

func collectValidPaths(configPath, platform, shell string, platformOnly bool) ([]string, int, error) {
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
		rawPaths = append(rawPaths, cfg.All.Paths...)
	}
	
	// Get platform-specific paths
	switch platform {
	case "Linux":
		rawPaths = append(rawPaths, cfg.Linux.Paths...)
		shellPaths := getShellSpecificPaths(shell, cfg.Linux)
		rawPaths = append(rawPaths, shellPaths...)
		totalSystemPaths += countValidSystemPaths(shell, cfg.Linux)
	case "macOS":
		rawPaths = append(rawPaths, cfg.MacOS.Paths...)
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


func EvaluateConfig(configPath, platform, shell string, platformOnly bool) (validPaths []string, skippedPaths []string, systemPathsCount int, err error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, nil, 0, fmt.Errorf("failed to parse yaml: %w", err)
	}

	var rawPaths []string
	var totalSystemPaths int
	
	// Add All section paths unless platform-only is specified
	if !platformOnly {
		rawPaths = append(rawPaths, cfg.All.Paths...)
	}
	
	// Get platform-specific paths
	switch platform {
	case "Linux":
		rawPaths = append(rawPaths, cfg.Linux.Paths...)
		shellPaths := getShellSpecificPaths(shell, cfg.Linux)
		rawPaths = append(rawPaths, shellPaths...)
		totalSystemPaths += countValidSystemPaths(shell, cfg.Linux)
	case "macOS":
		rawPaths = append(rawPaths, cfg.MacOS.Paths...)
		shellPaths := getShellSpecificPaths(shell, cfg.MacOS)
		rawPaths = append(rawPaths, shellPaths...)
		totalSystemPaths += countValidSystemPaths(shell, cfg.MacOS)
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
	return validPaths, skippedPaths, totalSystemPaths, nil
}

func PrintEvaluationReport(configPath, platform, shell string, inferred bool, platformOnly bool) error {
	valid, skipped, systemPathsCount, err := EvaluateConfig(configPath, platform, shell, platformOnly)
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

	fmt.Printf("\n%d paths included", len(valid))
	if systemPathsCount > 0 {
		fmt.Printf("*\n* Including %d system paths due to include_system_paths setting\n", systemPathsCount)
	} else {
		fmt.Printf("\n")
	}
	
	fmt.Printf("%d skipped", len(skipped))
	if platformOnly {
		fmt.Printf("*\n* Not including paths from 'All' section due to --platform-only\n")
	} else {
		fmt.Printf("\n")
	}
	fmt.Printf("\n")
	
	if len(valid) > 0 {
		fmt.Println("Output:")
		fmt.Printf("  %s\n\n", renderers[shell](valid))
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