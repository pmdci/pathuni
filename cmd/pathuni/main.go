package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var Version = "dev"

var (
	shell        string
	config       string
	platformOnly bool
	dumpFormat   string
	dumpInclude  string
)

func getConfigPath() string {
	if config != "" {
		return config
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "pathuni", "my_paths.yaml")
}

func getOSName() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS"
	case "linux":
		return "Linux"
	default:
		return ""
	}
}

func normalizeShellName(shell string) string {
	switch shell {
	case "pwsh":
		return "powershell"
	default:
		return shell
	}
}

func getShellName() (string, bool) {
	shellName := strings.ToLower(shell)
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
	shellName = normalizeShellName(shellName)
	return shellName, inferred
}

var initCmd = &cobra.Command{
	Use:     "init",
	Aliases: []string{"i"},
	Short:   "Generate shell initialization code (default mode)",
	Run: func(cmd *cobra.Command, args []string) {
		runInit()
	},
}

var dryRunCmd = &cobra.Command{
	Use:     "dry-run",
	Aliases: []string{"n"},
	Short:   "Show what paths would be included/skipped",
	Run: func(cmd *cobra.Command, args []string) {
		runDryRun()
	},
}

var dumpCmd = &cobra.Command{
	Use:     "dump",
	Aliases: []string{"d"},
	Short:   "Dump current PATH entries in various formats",
	Run: func(cmd *cobra.Command, args []string) {
		runDump()
	},
}

var rootCmd = &cobra.Command{
	Use:   "pathuni",
	Short: "Cross-platform PATH management for dotfiles",
	Long: `pathuni - Cross-platform PATH management for dotfiles

Generate shell-specific PATH export commands from a YAML config file.
Validates that directories exist before including them.`,
	Version: Version,
	Run: func(cmd *cobra.Command, args []string) {
		// Default to init command
		runInit()
	},
}

func init() {
	// Add persistent flags (available to all commands)
	rootCmd.PersistentFlags().StringVarP(&shell, "shell", "s", "", "Shell type: bash|zsh|sh|fish|powershell (auto-detected if not specified)")
	// If building for Windows in the future, will need to be something like %USERPROFILE%\AppData\Local\pathuni\my_paths.yaml
	rootCmd.PersistentFlags().StringVarP(&config, "config", "c", "", "Path to config file (default: ~/.config/pathuni/my_paths.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&platformOnly, "platform-only", "p", false, "Include only platform-specific paths, skip 'All' section")

	// Add subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(dryRunCmd)
	rootCmd.AddCommand(dumpCmd)


	// Add flags specific to dump command
	dumpCmd.Flags().StringVarP(&dumpFormat, "format", "f", "plain", "Output format: plain|json|yaml")
	dumpCmd.Flags().StringVarP(&dumpInclude, "include", "i", "all", "Paths to include: all|pathuni")

	// Custom version template
	rootCmd.SetVersionTemplate(`pathuni ` + Version + `
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
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

