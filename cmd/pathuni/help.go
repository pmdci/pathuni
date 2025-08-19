package main

import "fmt"

func printUsage() {
	fmt.Print(`pathuni - Cross-platform PATH management for dotfiles

USAGE:
    pathuni [OPTIONS]

    Generate shell-specific PATH export commands from a YAML config file.
    Validates that directories exist before including them.

OPTIONS:
    -shell <type>     Shell type: bash, zsh, sh, fish, powershell
                      (auto-detected if not specified)
    
    -eval             Preview what paths will be included/skipped
                      instead of generating export command
    
    -config <path>    Path to config file
                      (default: ~/.config/pathuni/my_paths.yaml)

EXAMPLES:
    pathuni                    # Generate export for current shell
    pathuni -shell=fish        # Generate for fish shell specifically  
    pathuni -eval              # Preview what would be exported
    
    eval "$(pathuni)"          # Apply to current shell session

CONFIG FORMAT:
    ~/.config/pathuni/my_paths.yaml:
    
        All:
          - "$HOME/.local/bin"
        
        Linux:
          - "/home/linuxbrew/.linuxbrew/bin"
          - "/home/linuxbrew/.linuxbrew/sbin"
        
        macOS:
          - /opt/homebrew/bin
          - "/opt/homebrew/sbin"

`)
}