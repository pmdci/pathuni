package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// pathExists checks if a path exists and is a directory
func pathExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// PathCleaner defines the contract for cleaning version manager paths from PATH
type PathCleaner interface {
	GetPathPattern() string
	CleanPath(currentPath string) string
}

// VersionManager defines the contract for version management tools
type VersionManager interface {
	Name() string
	IsEnabled() bool
	Detect() bool
	ResolvePath() (string, error)
	GenerateWrapper(shell string) string
	PathCleaner
}

// VersionManagerConfig holds configuration for a version manager
type VersionManagerConfig struct {
	Enabled     bool     `yaml:"enabled"`
	Directories []string `yaml:"directories"` // Max 5 items
}

// NvmManager implements VersionManager for Node Version Manager
type NvmManager struct {
	config VersionManagerConfig
}

// NewNvmManager creates a new NvmManager instance
func NewNvmManager(config VersionManagerConfig) (*NvmManager, error) {
	// Validate directories limit
	if len(config.Directories) > 5 {
		return nil, fmt.Errorf("nvm directories limit exceeded (max 5, got %d)", len(config.Directories))
	}
	return &NvmManager{config: config}, nil
}

// Name returns the version manager name
func (n *NvmManager) Name() string {
	return "nvm"
}

// IsEnabled checks if nvm support is enabled in config
func (n *NvmManager) IsEnabled() bool {
	return n.config.Enabled
}

// Detect checks if nvm is available on the system
func (n *NvmManager) Detect() bool {
	nvmDir := n.resolveNvmDir()
	return nvmDir != "" && pathExists(nvmDir)
}

// ResolvePath determines the current active nvm node version path
func (n *NvmManager) ResolvePath() (string, error) {
	nvmDir := n.resolveNvmDir()
	if nvmDir == "" {
		return "", fmt.Errorf("nvm directory not found")
	}

	// Try to find active version
	version, err := n.findActiveVersion(nvmDir)
	if err != nil {
		return "", fmt.Errorf("no active nvm version: %w", err)
	}

	// Build path to version's bin directory
	binPath := filepath.Join(nvmDir, "versions", "node", version, "bin")
	if !pathExists(binPath) {
		return "", fmt.Errorf("nvm version path not found: %s", binPath)
	}

	return binPath, nil
}

// GenerateWrapper creates shell-specific wrapper function for nvm
func (n *NvmManager) GenerateWrapper(shell string) string {
	switch shell {
	case "bash", "zsh", "sh":
		return n.generateBashWrapper()
	case "fish":
		return n.generateFishWrapper()
	case "powershell":
		return n.generatePowershellWrapper()
	default:
		return ""
	}
}

// GetPathPattern returns the regex pattern to match nvm paths in PATH
func (n *NvmManager) GetPathPattern() string {
	nvmHome := n.resolveNvmDir()
	if nvmHome == "" {
		// No valid nvm directory found - can't clean anything
		return ""
	}
	escaped := regexp.QuoteMeta(nvmHome)
	return escaped + `/versions/node/[^/]+/bin`
}

// CleanPath removes all nvm paths from the given PATH string
func (n *NvmManager) CleanPath(currentPath string) string {
	pattern := n.GetPathPattern()
	if pattern == "" {
		// No valid nvm directory found - return PATH unchanged
		return currentPath
	}
	
	re := regexp.MustCompile(pattern)
	
	// Split PATH into individual entries
	pathEntries := strings.Split(currentPath, ":")
	var cleanedEntries []string
	
	// Filter out nvm paths
	for _, entry := range pathEntries {
		if !re.MatchString(entry) {
			cleanedEntries = append(cleanedEntries, entry)
		}
	}
	
	return strings.Join(cleanedEntries, ":")
}

// buildCleanPath creates a clean PATH by removing old version manager paths and adding new ones
func buildCleanPath(staticPaths []string, versionManagers []VersionManager, versionManagerPaths map[string]string) []string {
	currentPath := os.Getenv("PATH")
	
	// Clean out all version manager paths from current PATH
	cleanedPath := currentPath
	for _, vm := range versionManagers {
		if vm.IsEnabled() {
			cleanedPath = vm.CleanPath(cleanedPath)
		}
	}
	
	// Build new PATH: static paths + version manager paths + cleaned system PATH entries
	var allPaths []string
	
	// Add static paths first
	allPaths = append(allPaths, staticPaths...)
	
	// Add version manager paths
	for _, vmPath := range versionManagerPaths {
		allPaths = append(allPaths, vmPath)
	}
	
	// Add cleaned system PATH entries
	if cleanedPath != "" {
		systemPaths := strings.Split(cleanedPath, ":")
		allPaths = append(allPaths, systemPaths...)
	}
	
	// Remove duplicates while preserving order
	seen := make(map[string]bool)
	var uniquePaths []string
	for _, path := range allPaths {
		if path != "" && !seen[path] {
			seen[path] = true
			uniquePaths = append(uniquePaths, path)
		}
	}
	
	return uniquePaths
}

// resolveNvmDir finds the nvm home directory using the directories array
func (n *NvmManager) resolveNvmDir() string {
	for _, dir := range n.config.Directories {
		if expanded := os.ExpandEnv(dir); expanded != "" && pathExists(expanded) {
			return expanded
		}
	}
	return ""
}

// findActiveVersion determines which node version is currently active
func (n *NvmManager) findActiveVersion(nvmDir string) (string, error) {
	// Try .nvmrc in current directory first
	if version := n.readNvmrc("."); version != "" {
		return n.resolveToFullVersion(nvmDir, version)
	}

	// Try default alias
	defaultAlias := filepath.Join(nvmDir, "alias", "default")
	if content, err := os.ReadFile(defaultAlias); err == nil {
		version := strings.TrimSpace(string(content))
		if version != "" {
			return n.resolveToFullVersion(nvmDir, version)
		}
	}

	return "", fmt.Errorf("no active version found")
}

// readNvmrc reads .nvmrc file from specified directory
func (n *NvmManager) readNvmrc(dir string) string {
	nvmrcPath := filepath.Join(dir, ".nvmrc")
	if content, err := os.ReadFile(nvmrcPath); err == nil {
		return strings.TrimSpace(string(content))
	}
	return ""
}

// resolveToFullVersion converts version alias to full version string
func (n *NvmManager) resolveToFullVersion(nvmDir, version string) (string, error) {
	versionsDir := filepath.Join(nvmDir, "versions", "node")

	// If version already looks like a full version (starts with v), use it
	if strings.HasPrefix(version, "v") {
		if pathExists(filepath.Join(versionsDir, version)) {
			return version, nil
		}
	}

	// If it's a major version like "22", find the latest v22.x.x
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return "", fmt.Errorf("cannot read versions directory: %w", err)
	}

	var candidates []string
	prefix := "v" + version + "."

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
			candidates = append(candidates, entry.Name())
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no version found matching %s", version)
	}

	// Return the latest (last in sorted order)
	// TODO: Implement proper semver sorting
	return candidates[len(candidates)-1], nil
}

// generateBashWrapper creates bash/zsh wrapper function
func (n *NvmManager) generateBashWrapper() string {
	return `nvm() {
    case "$1" in
        use|install|uninstall)
            unset -f nvm
            [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh" --no-use
            nvm "$@"
            local exit_code=$?
            eval "$(pathuni init --with-wrappers)"
            return $exit_code
            ;;
        *)
            unset -f nvm
            [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh" --no-use
            nvm "$@"
            local exit_code=$?
            eval "$(pathuni init --with-wrappers)"
            return $exit_code
            ;;
    esac
}`
}

// generateFishWrapper creates fish shell wrapper function
func (n *NvmManager) generateFishWrapper() string {
	return `function nvm
    switch $argv[1]
        case use install uninstall
            command nvm $argv
            if test $status -eq 0
                eval (pathuni init --with-wrappers)
            end
        case '*'
            command nvm $argv
    end
end`
}

// generatePowershellWrapper creates PowerShell wrapper function
func (n *NvmManager) generatePowershellWrapper() string {
	return `function nvm {
    switch ($args[0]) {
        { $_ -in "use", "install", "uninstall" } {
            & nvm @args
            if ($LASTEXITCODE -eq 0) {
                Invoke-Expression (pathuni init --with-wrappers)
            }
        }
        default {
            & nvm @args
        }
    }
}`
}