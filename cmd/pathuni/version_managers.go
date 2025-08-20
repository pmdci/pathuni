package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// pathExists checks if a path exists and is a directory
func pathExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// expandUser expands "~" and env vars
func expandUser(p string) string {
	if p == "" {
		return p
	}
	p = os.ExpandEnv(p)
	if len(p) == 0 || p[0] != '~' {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(home, strings.TrimPrefix(p, "~"))
}

// semver represents a semantic version for proper sorting
type semver struct {
	maj, min, pat int
	raw           string
}

// parseSemver parses a directory name into semver struct
func parseSemver(dirName string) (semver, bool) {
	s := strings.TrimPrefix(dirName, "v")
	parts := strings.Split(s, ".")
	if len(parts) < 3 {
		return semver{}, false
	}
	ma, e1 := strconv.Atoi(parts[0])
	mi, e2 := strconv.Atoi(parts[1])
	pa, e3 := strconv.Atoi(parts[2])
	if e1 != nil || e2 != nil || e3 != nil {
		return semver{}, false
	}
	return semver{ma, mi, pa, "v" + s}, true
}

// listInstalled returns all installed Node versions sorted by semver
func listInstalled(nvmDir string) ([]semver, error) {
	root := filepath.Join(nvmDir, "versions", "node")
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	out := []semver{}
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		if v, ok := parseSemver(e.Name()); ok {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].maj != out[j].maj {
			return out[i].maj < out[j].maj
		}
		if out[i].min != out[j].min {
			return out[i].min < out[j].min
		}
		return out[i].pat < out[j].pat
	})
	return out, nil
}

// highestInstalled returns the highest installed version
func highestInstalled(nvmDir string) (string, error) {
	vs, err := listInstalled(nvmDir)
	if err != nil || len(vs) == 0 {
		return "", fmt.Errorf("no installed versions")
	}
	return vs[len(vs)-1].raw, nil
}

// highestForMajor returns the highest installed version for a specific major
func highestForMajor(nvmDir string, major int) (string, error) {
	vs, err := listInstalled(nvmDir)
	if err != nil {
		return "", err
	}
	cand := []semver{}
	for _, v := range vs {
		if v.maj == major {
			cand = append(cand, v)
		}
	}
	if len(cand) == 0 {
		return "", fmt.Errorf("no version for major %d", major)
	}
	return cand[len(cand)-1].raw, nil
}

// resolveAliasFile reads alias file if present
func resolveAliasFile(nvmDir, name string) (string, bool) {
	p := filepath.Join(nvmDir, "alias", name)
	if b, err := os.ReadFile(p); err == nil {
		tgt := strings.TrimSpace(string(b))
		if tgt != "" {
			return tgt, true
		}
	}
	return "", false
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
	// Prefer NVM_BIN if set - this reflects what nvm actually chose
	if nb := strings.TrimSpace(os.Getenv("NVM_BIN")); nb != "" && pathExists(nb) {
		return nb, nil
	}

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
	base := regexp.QuoteMeta(nvmHome)
	// Match either "/" or "\\" as separators for cross-platform support
	sep := `(?:[/\\])`
	return base + sep + `versions` + sep + `node` + sep + `[^/\\]+` + sep + `bin`
}

// CleanPath removes all nvm paths from the given PATH string
func (n *NvmManager) CleanPath(currentPath string) string {
	fmt.Fprintf(os.Stderr, "[DEBUG] CleanPath: Raw PATH before cleaning:\n%s\n", currentPath)
	
	pattern := n.GetPathPattern()
	fmt.Fprintf(os.Stderr, "[DEBUG] CleanPath: Using regex pattern: %s\n", pattern)
	
	if pattern == "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] CleanPath: No pattern - returning PATH unchanged\n")
		return currentPath
	}
	
	re := regexp.MustCompile(pattern)
	
	// Split PATH into individual entries
	pathEntries := strings.Split(currentPath, string(os.PathListSeparator))
	var cleanedEntries []string
	
	fmt.Fprintf(os.Stderr, "[DEBUG] CleanPath: Evaluating %d PATH entries:\n", len(pathEntries))
	
	// Filter out nvm paths
	for i, entry := range pathEntries {
		matches := re.MatchString(entry)
		action := "KEEP"
		if matches {
			action = "REMOVE"
		} else {
			cleanedEntries = append(cleanedEntries, entry)
		}
		fmt.Fprintf(os.Stderr, "[DEBUG] CleanPath: [%d] %s -> %s: %s\n", i, action, entry, entry)
	}
	
	result := strings.Join(cleanedEntries, string(os.PathListSeparator))
	fmt.Fprintf(os.Stderr, "[DEBUG] CleanPath: Result has %d entries (removed %d)\n", len(cleanedEntries), len(pathEntries)-len(cleanedEntries))
	
	return result
}

// buildCleanPath creates a clean PATH by removing old version manager paths and adding new ones
func buildCleanPath(staticPaths []string, versionManagers []VersionManager, versionManagerPaths map[string]string) []string {
	currentPath := os.Getenv("PATH")
	fmt.Fprintf(os.Stderr, "[DEBUG] buildCleanPath: Starting with %d static paths, %d VMs, %d VM paths\n", len(staticPaths), len(versionManagers), len(versionManagerPaths))
	fmt.Fprintf(os.Stderr, "[DEBUG] buildCleanPath: VM paths map: %v\n", versionManagerPaths)
	
	// Clean out all version manager paths from current PATH
	cleanedPath := currentPath
	fmt.Fprintf(os.Stderr, "[DEBUG] buildCleanPath: Original PATH length: %d chars\n", len(currentPath))
	
	for _, vm := range versionManagers {
		if vm.IsEnabled() {
			fmt.Fprintf(os.Stderr, "[DEBUG] buildCleanPath: Cleaning with %s version manager\n", vm.Name())
			cleanedPath = vm.CleanPath(cleanedPath)
		}
	}
	
	fmt.Fprintf(os.Stderr, "[DEBUG] buildCleanPath: After cleaning, PATH length: %d chars\n", len(cleanedPath))
	
	// Build new PATH: static paths + version manager paths + cleaned system PATH entries
	var allPaths []string
	
	// Add static paths first
	fmt.Fprintf(os.Stderr, "[DEBUG] buildCleanPath: Adding %d static paths\n", len(staticPaths))
	allPaths = append(allPaths, staticPaths...)
	
	// Add version manager paths
	fmt.Fprintf(os.Stderr, "[DEBUG] buildCleanPath: Adding %d version manager paths\n", len(versionManagerPaths))
	for name, vmPath := range versionManagerPaths {
		fmt.Fprintf(os.Stderr, "[DEBUG] buildCleanPath: Adding %s: %s\n", name, vmPath)
		allPaths = append(allPaths, vmPath)
	}
	
	// Add cleaned system PATH entries
	if cleanedPath != "" {
		systemPaths := strings.Split(cleanedPath, string(os.PathListSeparator))
		fmt.Fprintf(os.Stderr, "[DEBUG] buildCleanPath: Adding %d cleaned system paths\n", len(systemPaths))
		allPaths = append(allPaths, systemPaths...)
	}
	
	// Remove duplicates while preserving order
	seen := make(map[string]bool)
	var uniquePaths []string
	duplicates := 0
	for _, path := range allPaths {
		if path != "" && !seen[path] {
			seen[path] = true
			uniquePaths = append(uniquePaths, path)
		} else if seen[path] {
			duplicates++
		}
	}
	
	fmt.Fprintf(os.Stderr, "[DEBUG] buildCleanPath: Final result: %d unique paths (%d duplicates removed)\n", len(uniquePaths), duplicates)
	
	return uniquePaths
}

// resolveNvmDir finds the nvm home directory, preferring NVM_DIR env var
func (n *NvmManager) resolveNvmDir() string {
	// Prefer NVM_DIR environment variable first
	if env := os.Getenv("NVM_DIR"); env != "" {
		if p := expandUser(env); pathExists(p) {
			return p
		}
	}
	
	// Fall back to config directories
	for _, dir := range n.config.Directories {
		if expanded := expandUser(dir); expanded != "" && pathExists(expanded) {
			return expanded
		}
	}
	return ""
}

// findActiveVersion determines which node version is currently active
func (n *NvmManager) findActiveVersion(nvmDir string) (string, error) {
	// Try .nvmrc with upward search
	if version := n.readNvmrcUpwards("."); version != "" {
		return n.resolveToFullVersion(nvmDir, version)
	}

	// Try default alias
	if content, err := os.ReadFile(filepath.Join(nvmDir, "alias", "default")); err == nil {
		if version := strings.TrimSpace(string(content)); version != "" {
			return n.resolveToFullVersion(nvmDir, version)
		}
	}

	return "", fmt.Errorf("no active version found")
}

// readNvmrcUpwards reads .nvmrc by searching upward from start directory
func (n *NvmManager) readNvmrcUpwards(start string) string {
	dir := start
	for {
		p := filepath.Join(dir, ".nvmrc")
		if content, err := os.ReadFile(p); err == nil {
			return strings.TrimSpace(string(content))
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// resolveToFullVersion converts version alias to full version string
func (n *NvmManager) resolveToFullVersion(nvmDir, version string) (string, error) {
	// 1) Direct match like "v22.10.1"
	if strings.HasPrefix(version, "v") {
		if pathExists(filepath.Join(nvmDir, "versions", "node", version)) {
			return version, nil
		}
	}

	// 2) Aliases via files first (covers default, named aliases, and often lts/*)
	if tgt, ok := resolveAliasFile(nvmDir, version); ok {
		return n.resolveToFullVersion(nvmDir, tgt)
	}

	// 3) Well-known aliases
	switch version {
	case "node", "stable", "current":
		return highestInstalled(nvmDir)
	}
	if strings.HasPrefix(version, "lts/") {
		// Try alias file (already tried above). If missing, best-effort: choose highest installed.
		if tgt, ok := resolveAliasFile(nvmDir, version); ok {
			return n.resolveToFullVersion(nvmDir, tgt)
		}
		return highestInstalled(nvmDir)
	}

	// 4) Major-only like "22"
	if m, err := strconv.Atoi(version); err == nil {
		return highestForMajor(nvmDir, m)
	}

	return "", fmt.Errorf("cannot resolve version spec: %s", version)
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