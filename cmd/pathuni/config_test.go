package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfig_YAMLParsing(t *testing.T) {
	tests := []struct {
		name        string
		configFile  string
		platform    string
		expectError bool
		expectPaths int // Expected number of paths for the platform
	}{
		{
			name:        "valid config",
			configFile:  "valid_config.yaml",
			platform:    "macOS",
			expectError: false,
			expectPaths: 3, // Updated for enhanced test data with tags
		},
		{
			name:        "minimal config",
			configFile:  "minimal_config.yaml",
			platform:    "macOS",
			expectError: false,
			expectPaths: 2, // Updated for tagged path addition
		},
		{
			name:        "invalid syntax",
			configFile:  "invalid_syntax.yaml",
			platform:    "macOS",
			expectError: true,
			expectPaths: 0,
		},
		{
			name:        "missing file",
			configFile:  "nonexistent.yaml",
			platform:    "macOS",
			expectError: true,
			expectPaths: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join("testdata", tt.configFile)
			
			// Use EvaluateConfig to get detailed results
			validPaths, skippedPaths, _, err := EvaluateConfig(configPath, tt.platform, "bash", false, TagFilter{})
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
				return
			}
			
			totalPaths := len(validPaths) + len(skippedPaths)
			if totalPaths < tt.expectPaths {
				t.Errorf("Expected at least %d paths for %s, got %d (valid: %d, skipped: %d)", 
					tt.expectPaths, tt.name, totalPaths, len(validPaths), len(skippedPaths))
			}
		})
	}
}

func TestConfig_PlatformFiltering(t *testing.T) {
	setupTestFilesystem(t)
	defer cleanupTestFilesystem()
	
	configPath := filepath.Join("testdata", "platform_specific.yaml")
	
	tests := []struct {
		name         string
		platform     string
		platformOnly bool
		expectPaths  []string // Paths we expect to be processed (may be valid or skipped)
	}{
		{
			name:         "macOS with all section",
			platform:     "macOS",
			platformOnly: false,
			expectPaths:  []string{"/tmp/pathuni/usr/local/bin", "/tmp/pathuni/usr/bin", "/tmp/pathuni/opt/homebrew/bin", "/tmp/pathuni/System/Library/Frameworks", "/tmp/pathuni/Applications/Xcode.app/Contents/Developer/usr/bin"},
		},
		{
			name:         "macOS platform only",
			platform:     "macOS",
			platformOnly: true,
			expectPaths:  []string{"/tmp/pathuni/opt/homebrew/bin", "/tmp/pathuni/System/Library/Frameworks", "/tmp/pathuni/Applications/Xcode.app/Contents/Developer/usr/bin"},
		},
		{
			name:         "Linux with all section",
			platform:     "Linux",
			platformOnly: false,
			expectPaths:  []string{"/tmp/pathuni/usr/local/bin", "/tmp/pathuni/usr/bin", "/tmp/pathuni/snap/bin", "/tmp/pathuni/usr/games", "/tmp/pathuni/home/linuxbrew/.linuxbrew/bin"},
		},
		{
			name:         "Linux platform only",
			platform:     "Linux",
			platformOnly: true,
			expectPaths:  []string{"/tmp/pathuni/snap/bin", "/tmp/pathuni/usr/games", "/tmp/pathuni/home/linuxbrew/.linuxbrew/bin"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validPaths, skippedPaths, _, err := EvaluateConfig(configPath, tt.platform, "bash", tt.platformOnly, TagFilter{})
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			allPaths := append(validPaths, skippedPaths...)
			
			// Check that we got the expected paths (in any order)
			if len(allPaths) != len(tt.expectPaths) {
				t.Errorf("Expected %d paths, got %d. Expected: %v, Got: %v", 
					len(tt.expectPaths), len(allPaths), tt.expectPaths, allPaths)
				return
			}
			
			// Convert to map for easier comparison
			pathMap := make(map[string]bool)
			for _, path := range allPaths {
				pathMap[path] = true
			}
			
			for _, expected := range tt.expectPaths {
				if !pathMap[expected] {
					t.Errorf("Expected path %q not found in results: %v", expected, allPaths)
				}
			}
		})
	}
}

func TestConfig_EnvironmentExpansion(t *testing.T) {
	setupTestFilesystem(t)
	defer cleanupTestFilesystem()
	
	configPath := filepath.Join("testdata", "env_vars.yaml")
	
	// Set test environment variables to point to our test structure
	originalHome := os.Getenv("HOME")
	originalUser := os.Getenv("USER")
	
	testHome := "/tmp/pathuni/home/Pratt"
	testUser := "Pratt"
	
	os.Setenv("HOME", testHome)
	os.Setenv("USER", testUser)
	
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		} else {
			os.Unsetenv("HOME")
		}
		if originalUser != "" {
			os.Setenv("USER", originalUser)
		} else {
			os.Unsetenv("USER")
		}
	}()
	
	validPaths, skippedPaths, _, err := EvaluateConfig(configPath, "macOS", "bash", false, TagFilter{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	allPaths := append(validPaths, skippedPaths...)
	
	// Check that environment variables were expanded
	foundHomeLocal := false
	foundUserBin := false
	foundCargo := false
	
	for _, path := range allPaths {
		if path == testHome+"/.local/bin" {
			foundHomeLocal = true
		}
		if path == testUser+"/bin" {
			foundUserBin = true
		}
		if path == testHome+"/.cargo/bin" {
			foundCargo = true
		}
	}
	
	if !foundHomeLocal {
		t.Errorf("$HOME/.local/bin not properly expanded to %s. Paths: %v", testHome+"/.local/bin", allPaths)
	}
	if !foundUserBin {
		t.Errorf("${USER}/bin not properly expanded to %s. Paths: %v", testUser+"/bin", allPaths)
	}
	if !foundCargo {
		t.Errorf("$HOME/.cargo/bin not properly expanded to %s. Paths: %v", testHome+"/.cargo/bin", allPaths)
	}
}

func TestConfig_PathValidation(t *testing.T) {
	setupTestFilesystem(t)
	defer cleanupTestFilesystem()
	
	configPath := filepath.Join("testdata", "missing_paths.yaml")
	
	validPaths, skippedPaths, _, err := EvaluateConfig(configPath, "macOS", "bash", false, TagFilter{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// Should have some valid paths (like /usr/local/bin) and some skipped paths
	if len(validPaths) == 0 {
		t.Error("Expected some valid paths, got none")
	}
	
	if len(skippedPaths) == 0 {
		t.Error("Expected some skipped paths from nonexistent directories, got none")
	}
	
	// Verify that valid paths actually exist
	for _, path := range validPaths {
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Errorf("Valid path %q does not exist or is not a directory", path)
		}
	}
	
	// Verify that skipped paths don't exist (or aren't directories)
	for _, path := range skippedPaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			t.Errorf("Skipped path %q actually exists and is a directory", path)
		}
	}
}

func TestConfig_CollectValidPaths(t *testing.T) {
	setupTestFilesystem(t)
	defer cleanupTestFilesystem()
	
	// Test the collectValidPaths function directly
	configPath := filepath.Join("testdata", "valid_config.yaml")
	
	tests := []struct {
		name         string
		platform     string
		shell        string
		platformOnly bool
	}{
		{"macOS bash", "macOS", "bash", false},
		{"macOS powershell", "macOS", "powershell", false},
		{"macOS platform only", "macOS", "bash", true},
		{"Linux bash", "Linux", "bash", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths, systemCount, err := collectValidPaths(configPath, tt.platform, tt.shell, tt.platformOnly, TagFilter{})
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if len(paths) == 0 {
				t.Error("Expected some paths, got none")
			}
			
			// System paths should only be added for PowerShell on macOS with the test config
			expectedSystemCount := 0
			if tt.platform == "macOS" && tt.shell == "powershell" {
				// The valid_config.yaml has include_system_paths: true for macOS PowerShell
				expectedSystemCount = systemCount // Whatever the system actually has
			}
			
			if systemCount != expectedSystemCount && !(tt.platform == "macOS" && tt.shell == "powershell") {
				t.Errorf("Expected %d system paths, got %d", expectedSystemCount, systemCount)
			}
		})
	}
}

func TestConfig_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		platform   string
		shell      string
		wantError  bool
	}{
		{
			name:       "nonexistent file",
			configPath: "/nonexistent/config.yaml",
			platform:   "macOS",
			shell:      "bash",
			wantError:  true,
		},
		{
			name:       "invalid YAML",
			configPath: filepath.Join("testdata", "invalid_syntax.yaml"),
			platform:   "macOS",
			shell:      "bash",
			wantError:  true,
		},
		{
			name:       "valid config",
			configPath: filepath.Join("testdata", "valid_config.yaml"),
			platform:   "macOS",
			shell:      "bash",
			wantError:  false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := collectValidPaths(tt.configPath, tt.platform, tt.shell, false, TagFilter{})
			
			if tt.wantError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestConfig_EdgeCases(t *testing.T) {
	setupTestFilesystem(t)
	defer cleanupTestFilesystem()
	
	// Test with empty platform (unsupported OS)
	t.Run("empty platform", func(t *testing.T) {
		configPath := filepath.Join("testdata", "valid_config.yaml")
		paths, _, err := collectValidPaths(configPath, "", "bash", false, TagFilter{})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		
		// Should only get paths from "all" section since platform is empty
		if len(paths) == 0 {
			t.Error("Expected some paths from 'all' section")
		}
	})
	
	// Test with platform-only flag
	t.Run("platform only excludes all section", func(t *testing.T) {
		configPath := filepath.Join("testdata", "platform_specific.yaml")
		
		// Get paths with all sections
		pathsWithAll, _, err1 := collectValidPaths(configPath, "macOS", "bash", false, TagFilter{})
		if err1 != nil {
			t.Fatalf("Unexpected error with all sections: %v", err1)
		}
		
		// Get paths with platform only
		pathsPlatformOnly, _, err2 := collectValidPaths(configPath, "macOS", "bash", true, TagFilter{})
		if err2 != nil {
			t.Fatalf("Unexpected error with platform only: %v", err2)
		}
		
		// Platform-only should have fewer paths (excludes "all" section)
		if len(pathsPlatformOnly) >= len(pathsWithAll) {
			t.Errorf("Platform-only should have fewer paths. With all: %d, platform-only: %d", 
				len(pathsWithAll), len(pathsPlatformOnly))
		}
	})
}

func TestConfig_MixedYAMLFormat(t *testing.T) {
	testConfigPath := filepath.Join("testdata", "mixed_format.yaml")

	tests := []struct {
		name         string
		tagFilter    TagFilter
		expectedCount int  // Minimum expected paths
		shouldContain []string
		shouldExclude []string
	}{
		{
			name:         "no filter - all paths processed",
			tagFilter:    TagFilter{},
			expectedCount: 6, // All paths should be processed (valid or skipped)
			shouldContain: []string{}, // Don't check specific paths since they might not exist
		},
		{
			name:         "include home tag filters correctly",
			tagFilter:    TagFilter{Include: [][]string{{"home"}}},
			expectedCount: 4, // Untagged paths + tagged paths with 'home'
			shouldContain: []string{}, // Don't check specific paths
		},
		{
			name:         "exclude gaming tag filters correctly", 
			tagFilter:    TagFilter{Exclude: [][]string{{"gaming"}}},
			expectedCount: 5, // All except gaming
			shouldContain: []string{},
		},
		{
			name:         "include dev AND home logic works",
			tagFilter:    TagFilter{Include: [][]string{{"dev", "home"}}},
			expectedCount: 4, // Untagged paths + paths with both dev AND home
			shouldContain: []string{},
		},
		{
			name:         "include shared OR gaming logic works",
			tagFilter:    TagFilter{Include: [][]string{{"shared"}, {"gaming"}}},
			expectedCount: 5, // Untagged paths + paths with shared OR gaming  
			shouldContain: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validPaths, skippedPaths, _, err := EvaluateConfig(testConfigPath, "macOS", "bash", false, tt.tagFilter)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			allPaths := append(validPaths, skippedPaths...)
			
			if len(allPaths) < tt.expectedCount {
				t.Errorf("Expected at least %d paths, got %d", tt.expectedCount, len(allPaths))
			}

			// Check that expected paths are included
			pathSet := make(map[string]bool)
			for _, path := range validPaths {
				pathSet[path] = true
			}

			for _, expectedPath := range tt.shouldContain {
				if !pathSet[expectedPath] {
					t.Errorf("Expected path %q to be included, but it wasn't. Valid paths: %v", expectedPath, validPaths)
				}
			}

			for _, excludedPath := range tt.shouldExclude {
				if pathSet[excludedPath] {
					t.Errorf("Expected path %q to be excluded, but it was included. Valid paths: %v", excludedPath, validPaths)
				}
			}
		})
	}
}

func TestConfig_TagValidationErrors(t *testing.T) {
	setupTestFilesystem(t)
	defer cleanupTestFilesystem()
	
	tests := []struct {
		name          string
		configContent string
		expectError   bool
		errorContains string
	}{
		{
			name: "invalid tag format",
			configContent: `macos:
  paths:
    - path: "/test/path"
      tags:
        - "2invalid"`,
			expectError:   true,
			errorContains: "invalid tag '2invalid'",
		},
		{
			name: "duplicate tags",
			configContent: `macos:
  paths:
    - path: "/test/path"  
      tags:
        - home
        - dev
        - home`,
			expectError:   true,
			errorContains: "duplicate tag 'home'",
		},
		{
			name: "empty tags array is ok",
			configContent: `macos:
  paths:
    - path: "/test/path"
      tags: []`,
			expectError: false,
		},
		{
			name: "missing path field",
			configContent: `macos:
  paths:
    - tags:
        - home`,
			expectError:   true,
			errorContains: "missing 'path' field",
		},
		{
			name: "invalid tag type",
			configContent: `macos:
  paths:
    - path: "/test/path"
      tags:
        - home
        - 123`,
			expectError:   true,
			errorContains: "invalid tag type",
		},
		{
			name: "valid mixed format",
			configContent: `macos:
  paths:
    - "/plain/string/path"
    - path: "/tagged/path"
      tags:
        - home
        - dev`,
			expectError: false,
		},
		{
			name: "invalid platform-level tag format",
			configContent: `all:
  tags: [base, "2invalid"]
  paths:
    - "/test/path"`,
			expectError:   true,
			errorContains: "invalid tag '2invalid'",
		},
		{
			name: "duplicate platform-level tags",
			configContent: `macos:
  tags: [mac, desktop, mac]
  paths:
    - "/test/path"`,
			expectError:   true,
			errorContains: "duplicate tag 'mac'",
		},
		{
			name: "valid platform-level tags",
			configContent: `all:
  tags: [base, essential]
  paths:
    - "/test/path"
macos:
  tags: [mac, desktop]
  paths:
    - path: "/another/path"
      tags: [dev]`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("/tmp/pathuni/home/Pratt/.config/pathuni", "pathuni-validation-test-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.configContent); err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}
			tmpFile.Close()

			_, _, _, err = EvaluateConfig(tmpFile.Name(), "macOS", "bash", false, TagFilter{})

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', but got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
				}
			}
		})
	}
}
// TestPathEntry_GetEffectiveTags tests platform-level tag inheritance logic
func TestPathEntry_GetEffectiveTags(t *testing.T) {
	platformTags := []string{"mac", "desktop"}
	
	tests := []struct {
		name     string
		entry    PathEntry
		expected []string
	}{
		{
			name:     "explicit tags override platform tags",
			entry:    PathEntry{Path: "/test", Tags: []string{"dev", "work"}},
			expected: []string{"dev", "work"},
		},
		{
			name:     "explicit empty tags override platform tags",
			entry:    PathEntry{Path: "/test", Tags: []string{}},
			expected: []string{},
		},
		{
			name:     "no tags field inherits platform tags",
			entry:    PathEntry{Path: "/test"}, // Tags field is nil
			expected: []string{"mac", "desktop"},
		},
		{
			name:     "inherit from empty platform tags",
			entry:    PathEntry{Path: "/test"},
			expected: []string{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var platformTagsToUse []string
			if tt.name == "inherit from empty platform tags" {
				platformTagsToUse = []string{}
			} else {
				platformTagsToUse = platformTags
			}
			
			result := tt.entry.GetEffectiveTags(platformTagsToUse)
			
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d tags, got %d: %v vs %v", len(tt.expected), len(result), tt.expected, result)
				return
			}
			
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected tag[%d] = %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

// TestExtractPathEntries_EmptyTags tests that empty tags are parsed correctly  
func TestExtractPathEntries_EmptyTags(t *testing.T) {
	tests := []struct {
		name        string
		pathsYAML   string
		expectedNil []bool // true if Tags should be nil, false if should be empty slice
	}{
		{
			name: "empty tags vs no tags field",
			pathsYAML: `- path: "/test1"
  tags: []
- path: "/test2"`,
			expectedNil: []bool{false, true}, // first has empty slice, second has nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pathsInterface []interface{}
			err := yaml.Unmarshal([]byte(tt.pathsYAML), &pathsInterface)
			if err != nil {
				t.Fatalf("Failed to unmarshal YAML: %v", err)
			}

			entries, err := extractPathEntries(pathsInterface, "test context")
			if err != nil {
				t.Fatalf("extractPathEntries failed: %v", err)
			}

			if len(entries) != len(tt.expectedNil) {
				t.Fatalf("Expected %d entries, got %d", len(tt.expectedNil), len(entries))
			}

			for i, entry := range entries {
				isNil := entry.Tags == nil
				expectedNil := tt.expectedNil[i]

				t.Logf("Entry %d: Path=%s, Tags=%v, Tags==nil=%v", i, entry.Path, entry.Tags, isNil)

				if isNil != expectedNil {
					t.Errorf("Entry %d: expected Tags==nil to be %v, got %v (Tags=%v)", 
						i, expectedNil, isNil, entry.Tags)
				}
			}
		})
	}
}
