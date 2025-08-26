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
	osOverride   string
	dumpFormat   string
	dumpScope    string
	tagsInclude  string
	tagsExclude  string
)

func getConfigPath() string {
	if config != "" {
		return config
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "pathuni", "my_paths.yaml")
}

func getOSName() (string, bool) {
	osName := strings.ToLower(osOverride)
	inferred := false
	
	if osName == "" {
		switch runtime.GOOS {
		case "darwin":
			osName = "macos"
		case "linux":
			osName = "linux"
		default:
			osName = ""
		}
		inferred = true
	}
	
	// Normalise OS name
	switch osName {
	case "darwin", "macos":
		return "macOS", inferred
	case "linux":
		return "Linux", inferred
	default:
		return "", inferred
	}
}

func osIsValid(osName string) bool {
	switch osName {
	case "macOS", "Linux":
		return true
	default:
		return false
	}
}

func osNames() []string {
	return []string{"macOS", "Linux"}
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
	rootCmd.PersistentFlags().StringVarP(&shell, "shell", "S", "", "Shell type: sh|ash|bash|dash|ksh|mksh|yash|zsh|fish|powershell (detected if not specified)")
	// If building for Windows in the future, will need to be something like %USERPROFILE%\AppData\Local\pathuni\my_paths.yaml
	rootCmd.PersistentFlags().StringVarP(&config, "config", "c", "", "Path to config file (default: ~/.config/pathuni/my_paths.yaml)")
	rootCmd.PersistentFlags().StringVarP(&osOverride, "os", "O", "", "OS type: macOS|linux (detected if not specified)")
	rootCmd.PersistentFlags().StringVarP(&tagsInclude, "tags-include", "t", "", "Include paths with tags (comma=OR, plus=AND): home,dev or work+server")
	rootCmd.PersistentFlags().StringVarP(&tagsExclude, "tags-exclude", "x", "", "Exclude paths with tags (comma=OR, plus=AND): gaming,temp or work+gaming")

	// Add subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(dryRunCmd)
	rootCmd.AddCommand(dumpCmd)


	// Add flags specific to dump command
	dumpCmd.Flags().StringVarP(&dumpFormat, "format", "f", "plain", "Output format: plain|json|yaml")
	dumpCmd.Flags().StringVarP(&dumpScope, "scope", "s", "full", "Paths to include: system|pathuni|full")

	// Custom version template
	rootCmd.SetVersionTemplate(`pathuni ` + Version + `

░█▀█░█▀█░▀█▀░█░█░█░█░█▀█░▀█▀ Copyright (C) 2025 Pedro Innecco
░█▀▀░█▀█░░█░░█▀█░█░█░█░█░░█░ <https://pedroinnecco.com>
░▀░░░▀░▀░░▀░░▀░▀░▀▀▀░▀░▀░▀▀▀

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

