package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

var supportedShells = map[string]struct{}{
	"bash": {}, "zsh": {}, "sh": {}, "dash": {}, "ash": {}, "ksh": {}, "mksh": {}, "yash": {},
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
	"dash":       renderBash,
	"ash":        renderBash,
	"ksh":        renderBash,
	"mksh":       renderBash,
	"yash":       renderBash,
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
    osName, _ := getOSName()
    shellName, _ := getShellName()

    if !osIsValid(osName) {
        fmt.Fprintf(os.Stderr, "Unsupported OS '%s'. Supported OS: %s\n", osName, strings.Join(osNames(), ", "))
        os.Exit(1)
    }

    if !shellIsValid(shellName) {
        fmt.Fprintf(os.Stderr, "Unsupported shell '%s'. Supported shells: %s\n", shellName, strings.Join(shellNames(), ", "))
        os.Exit(1)
    }

    // Validate scope flag
    if !isValidScope(scope) {
        fmt.Fprintf(os.Stderr, "Error: Invalid scope option '%s'. Use 'system', 'pathuni', or 'full'\n", scope)
        os.Exit(1)
    }

    // Compute the list based on scope using shared resolvers
    var paths []string
    switch scope {
    case "system":
        p, err := resolveSystemPaths()
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            os.Exit(1)
        }
        paths = p
    case "pathuni":
        p, err := resolvePathuniPaths()
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            os.Exit(1)
        }
        paths = p
    case "full":
        sys, err := resolveSystemPaths()
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            os.Exit(1)
        }
        pu, err := resolvePathuniPaths()
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            os.Exit(1)
        }
        // pathuni-first precedence for init
        paths = mergeFull(pu, sys, true)
    }

    // Generate PATH export using original renderers
    fmt.Println(renderers[shellName](paths))
}
