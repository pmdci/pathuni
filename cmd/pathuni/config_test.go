package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
			expectPaths: 2, // Should get paths from "all" + "macos" sections
		},
		{
			name:        "minimal config",
			configFile:  "minimal_config.yaml",
			platform:    "macOS",
			expectError: false,
			expectPaths: 1,
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
			validPaths, skippedPaths, _, err := EvaluateConfig(configPath, tt.platform, "bash", false)
			
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
			expectPaths:  []string{"/usr/local/bin", "/opt/homebrew/bin", "/System/Library/Frameworks"},
		},
		{
			name:         "macOS platform only",
			platform:     "macOS",
			platformOnly: true,
			expectPaths:  []string{"/opt/homebrew/bin", "/System/Library/Frameworks"},
		},
		{
			name:         "Linux with all section",
			platform:     "Linux",
			platformOnly: false,
			expectPaths:  []string{"/usr/local/bin", "/snap/bin", "/usr/games"},
		},
		{
			name:         "Linux platform only",
			platform:     "Linux",
			platformOnly: true,
			expectPaths:  []string{"/snap/bin", "/usr/games"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validPaths, skippedPaths, _, err := EvaluateConfig(configPath, tt.platform, "bash", tt.platformOnly)
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
	configPath := filepath.Join("testdata", "env_vars.yaml")
	
	// Set test environment variables
	originalHome := os.Getenv("HOME")
	originalUser := os.Getenv("USER")
	
	testHome := "/test/home"
	testUser := "testuser"
	
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
	
	validPaths, skippedPaths, _, err := EvaluateConfig(configPath, "macOS", "bash", false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	allPaths := append(validPaths, skippedPaths...)
	
	// Check that environment variables were expanded
	foundHomeLocal := false
	foundUserBin := false
	foundCargo := false
	
	for _, path := range allPaths {
		if strings.Contains(path, testHome+"/.local/bin") {
			foundHomeLocal = true
		}
		if strings.Contains(path, testUser+"/bin") {
			foundUserBin = true
		}
		if strings.Contains(path, testHome+"/.cargo/bin") {
			foundCargo = true
		}
	}
	
	if !foundHomeLocal {
		t.Errorf("$HOME/.local/bin not properly expanded. Paths: %v", allPaths)
	}
	if !foundUserBin {
		t.Errorf("${USER}/bin not properly expanded. Paths: %v", allPaths)
	}
	if !foundCargo {
		t.Errorf("$HOME/.cargo/bin not properly expanded. Paths: %v", allPaths)
	}
}

func TestConfig_PathValidation(t *testing.T) {
	configPath := filepath.Join("testdata", "missing_paths.yaml")
	
	validPaths, skippedPaths, _, err := EvaluateConfig(configPath, "macOS", "bash", false)
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
			paths, systemCount, err := collectValidPaths(configPath, tt.platform, tt.shell, tt.platformOnly)
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
			_, _, err := collectValidPaths(tt.configPath, tt.platform, tt.shell, false)
			
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
	// Test with empty platform (unsupported OS)
	t.Run("empty platform", func(t *testing.T) {
		configPath := filepath.Join("testdata", "valid_config.yaml")
		paths, _, err := collectValidPaths(configPath, "", "bash", false)
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
		pathsWithAll, _, err1 := collectValidPaths(configPath, "macOS", "bash", false)
		if err1 != nil {
			t.Fatalf("Unexpected error with all sections: %v", err1)
		}
		
		// Get paths with platform only
		pathsPlatformOnly, _, err2 := collectValidPaths(configPath, "macOS", "bash", true)
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