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

// Defer renderers: generate code that references the live PATH at evaluation
// time, prepending the provided pathuni paths (pathuni-first semantics).
var renderersDefer = map[string]func([]string) string{
    "bash":       renderBashDefer,
    "zsh":        renderBashDefer,
    "sh":         renderBashDefer,
    "dash":       renderBashDefer,
    "ash":        renderBashDefer,
    "ksh":        renderBashDefer,
    "mksh":       renderBashDefer,
    "yash":       renderBashDefer,
    "fish":       renderFishDefer,
    "powershell": renderPwshDefer,
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

func renderBashDefer(paths []string) string {
    joined := strings.Join(paths, ":")
    if joined == "" {
        return "export PATH=\"${PATH}\""
    }
    return fmt.Sprintf("export PATH=\"%s:${PATH}\"", joined)
}

func renderFishDefer(paths []string) string {
    joined := strings.Join(paths, " ")
    if joined == "" {
        return "set -gx PATH $PATH"
    }
    return fmt.Sprintf("set -gx PATH %s $PATH", joined)
}

func renderPwshDefer(paths []string) string {
    joined := strings.Join(paths, ":")
    if joined == "" {
        return "$env:PATH = \"$env:PATH\""
    }
    return fmt.Sprintf("$env:PATH = \"%s:$env:PATH\"", joined)
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
    if !isValidPrune(prune) {
        fmt.Fprintf(os.Stderr, "Error: Invalid prune option '%s'. Use 'none', 'pathuni', 'system', or 'all'\n", prune)
        os.Exit(1)
    }

    // Handle defer-env for init: prepend pathuni and reference live PATH
    if deferEnv {
        if scope != "full" {
            fmt.Fprintf(os.Stderr, "Error: --defer-env is only valid with --scope=full\n")
            os.Exit(1)
        }
        if prune == "system" || prune == "all" {
            fmt.Fprintf(os.Stderr, "Error: --prune=%s is incompatible with --defer-env (system PATH is not expanded)\n", prune)
            os.Exit(1)
        }
        pu, err := resolvePathuniPaths()
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            os.Exit(1)
        }
        fmt.Println(renderersDefer[shellName](pu))
        return
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
        if prune == "system" || prune == "all" {
            p = filterExisting(p)
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
        if prune == "system" || prune == "all" {
            sys = filterExisting(sys)
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
