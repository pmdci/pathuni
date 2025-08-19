package main

import (
	"fmt"
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
	return fmt.Sprintf("export PATH=\"%s:$PATH\"", strings.Join(paths, ":"))
}

func renderFish(paths []string) string {
	return fmt.Sprintf("set -gx PATH %s $PATH", strings.Join(paths, " "))
}

func renderPwsh(paths []string) string {
	return fmt.Sprintf("$env:PATH = \"%s:$env:PATH\"", strings.Join(paths, ":"))
}