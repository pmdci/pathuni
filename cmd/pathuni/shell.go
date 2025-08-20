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

func runInit() {
	configPath := getConfigPath()
	osName := getOSName()
	shellName, _ := getShellName()

	if !shellIsValid(shellName) {
		fmt.Fprintf(os.Stderr, "Unsupported shell '%s'. Supported shells: %s\n", shellName, strings.Join(shellNames(), ", "))
		os.Exit(1)
	}

	paths, _, err := collectValidPaths(configPath, osName, shellName, platformOnly)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
		os.Exit(1)
	}

	// Generate PATH export using original renderers
	fmt.Println(renderers[shellName](paths))
}