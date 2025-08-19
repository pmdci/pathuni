package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func printVersion() {
	fmt.Print(`pathuni 0.1
Copyright (C) 2025 Pedro Innecco <https://pedroinnecco.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program comes with ABSOLUTELY NO WARRANTY.
See <https://www.gnu.org/licenses/gpl-3.0.html> for details.

Source: https://github.com/pmdci/pathuni
`)
}

func main() {
	shellFlag := flag.String("shell", "", "")
	evalFlag := flag.Bool("eval", false, "")
	configFlag := flag.String("config", "", "")
	versionFlag := flag.Bool("version", false, "")
	
	flag.Usage = printUsage
	flag.Parse()
	
	if *versionFlag {
		printVersion()
		return
	}

	home, _ := os.UserHomeDir()
	configPath := *configFlag
	if configPath == "" {
		configPath = filepath.Join(home, ".config", "pathuni", "my_paths.yaml")
	}

	var osName string
	switch runtime.GOOS {
	case "darwin":
		osName = "macOS"
	case "linux":
		osName = "Linux"
	default:
		osName = ""
	}

	shellName := strings.ToLower(*shellFlag)
	inferred := false
	if shellName == "" {
		if shellEnv := os.Getenv("SHELL"); shellEnv != "" {
			shellName = strings.ToLower(filepath.Base(shellEnv))
			inferred = true
		} else {
			shellName = "bash"
			inferred = true
		}
	}

	if !shellIsValid(shellName) {
		fmt.Fprintf(os.Stderr, "Unsupported shell '%s'. Supported shells: %s\n", shellName, strings.Join(shellNames(), ", "))
		os.Exit(1)
	}

	if *evalFlag {
		err := PrintEvaluationReport(configPath, osName, shellName, inferred)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	paths, err := collectValidPaths(configPath, osName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(renderers[shellName](paths))
}

