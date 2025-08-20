package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

var supportedShells = map[string]struct{}{
	"bash": {}, "zsh": {}, "sh": {},
	"fish": {},
	"powershell": {},
}

func shellIsValid(s string) bool {
	_, ok := supportedShells[s]
	return ok
}

func shellNames() []string {
	keys := make([]string, 0, len(supportedShells))
	for k := range supportedShells {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

var renderers = map[string]func([]string) string{
	"bash":       renderBash,
	"zsh":        renderBash,
	"sh":         renderBash,
	"fish":       renderFish,
	"powershell": renderPwsh,
}

func renderBash(paths []string) string {
	return fmt.Sprintf("export PATH=\"%s\"", strings.Join(paths, ":"))
}

func renderFish(paths []string) string {
	return fmt.Sprintf("set -gx PATH %s", strings.Join(paths, " "))
}

func renderPwsh(paths []string) string {
	return fmt.Sprintf("$env:PATH = \"%s\"", strings.Join(paths, ":"))
}

func runInit(withWrappers bool) {
	configPath := getConfigPath()
	osName := getOSName()
	shellName, _ := getShellName()

	if !shellIsValid(shellName) {
		fmt.Fprintf(os.Stderr, "Unsupported shell '%s'. Supported shells: %s\n", shellName, strings.Join(shellNames(), ", "))
		os.Exit(1)
	}

	paths, err := collectValidPaths(configPath, osName, platformOnly)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
		os.Exit(1)
	}

	// Add version manager paths with smart cleaning
	versionManagers, err := getVersionManagers(configPath, osName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading version managers config: %v\n", err)
		os.Exit(1)
	}

	// Collect version managers for new PATH building logic
	var nvmVM VersionManager
	var otherVMs []VersionManager

	for name, config := range versionManagers {
		if name == "nvm" && config.Enabled {
			nvmManager, err := NewNvmManager(config)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating nvm manager: %v\n", err)
				continue
			}
			
			if nvmManager.Detect() {
				nvmVM = nvmManager
			}
		}
		// Future: Add other version managers (rbenv, pyenv) to otherVMs
	}

	// Build clean PATH using Pedro's invariant enforcement logic
	finalPaths, err := buildCleanPathV2(paths, nvmVM, otherVMs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building clean PATH: %v\n", err)
		os.Exit(1)
	}

	// Generate PATH export using original renderers
	fmt.Println(renderers[shellName](finalPaths))

	// Generate wrapper functions if requested
	if withWrappers {
		for name, config := range versionManagers {
			if name == "nvm" && config.Enabled {
				nvmManager, _ := NewNvmManager(config)
				if nvmManager.Detect() {
					if wrapper := nvmManager.GenerateWrapper(shellName); wrapper != "" {
						fmt.Printf("\n%s\n", wrapper)
					}
				}
			}
		}
	}
}