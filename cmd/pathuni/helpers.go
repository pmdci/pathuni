package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func getSystemPaths() ([]string, error) {
	var systemPaths []string
	
	// Read /etc/paths
	if paths, err := readPathsFile("/etc/paths"); err == nil {
		systemPaths = append(systemPaths, paths...)
	}
	
	// Read files in /etc/paths.d/
	pathsDir := "/etc/paths.d"
	if entries, err := os.ReadDir(pathsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				filePath := filepath.Join(pathsDir, entry.Name())
				if paths, err := readPathsFile(filePath); err == nil {
					systemPaths = append(systemPaths, paths...)
				}
			}
		}
	}
	
	return systemPaths, nil
}

func readPathsFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var paths []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			paths = append(paths, line)
		}
	}
	
	return paths, scanner.Err()
}

func getShellSpecificPaths(shell string, platformConfig PlatformConfig) []string {
	var additionalPaths []string
	
	switch shell {
	case "powershell":
		if platformConfig.PowerShell != nil && platformConfig.PowerShell.IncludeSystemPaths {
			if systemPaths, err := getSystemPaths(); err == nil {
				additionalPaths = append(additionalPaths, systemPaths...)
			}
		}
	}
	
	return additionalPaths
}

func countValidSystemPaths(shell string, platformConfig PlatformConfig) int {
	if shell != "powershell" || platformConfig.PowerShell == nil || !platformConfig.PowerShell.IncludeSystemPaths {
		return 0
	}
	
	systemPaths, err := getSystemPaths()
	if err != nil {
		return 0
	}
	
	validCount := 0
	for _, path := range systemPaths {
		expanded := os.ExpandEnv(path)
		if info, err := os.Stat(expanded); err == nil && info.IsDir() {
			validCount++
		}
	}
	
	return validCount
}