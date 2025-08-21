package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpers_ReadPathsFile(t *testing.T) {
	setupTestFilesystem(t)
	defer cleanupTestFilesystem()
	
	tests := []struct {
		name        string
		filename    string
		expectPaths []string
		expectError bool
	}{
		{
			name:        "system paths file",
			filename:    "system_paths/paths",
			expectPaths: []string{"/tmp/pathuni/usr/local/bin", "/tmp/pathuni/usr/bin", "/tmp/pathuni/bin", "/tmp/pathuni/usr/sbin", "/tmp/pathuni/sbin"},
			expectError: false,
		},
		{
			name:        "homebrew paths",
			filename:    "system_paths/paths.d/homebrew",
			expectPaths: []string{"/tmp/pathuni/opt/homebrew/bin", "/tmp/pathuni/opt/homebrew/sbin"},
			expectError: false,
		},
		{
			name:        "user paths with comments",
			filename:    "system_paths/paths.d/user_paths",
			expectPaths: []string{"/tmp/pathuni/usr/local/go/bin", "/tmp/pathuni/usr/local/node/bin"},
			expectError: false,
		},
		{
			name:        "nonexistent file",
			filename:    "nonexistent/paths",
			expectPaths: nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join("testdata", tt.filename)
			
			paths, err := readPathsFile(filePath)
			
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
			
			if len(paths) != len(tt.expectPaths) {
				t.Errorf("Expected %d paths for %s, got %d. Expected: %v, Got: %v", 
					len(tt.expectPaths), tt.name, len(paths), tt.expectPaths, paths)
				return
			}
			
			for i, expected := range tt.expectPaths {
				if i >= len(paths) || paths[i] != expected {
					t.Errorf("Path mismatch at index %d for %s. Expected: %q, Got: %q", 
						i, tt.name, expected, paths[i])
				}
			}
		})
	}
}

// Test helper function that uses testdata instead of real system paths
func getTestSystemPaths() ([]string, error) {
	var systemPaths []string
	
	// Read testdata/system_paths/paths
	if paths, err := readPathsFile(filepath.Join("testdata", "system_paths", "paths")); err == nil {
		systemPaths = append(systemPaths, paths...)
	}
	
	// Read files in testdata/system_paths/paths.d/
	pathsDir := filepath.Join("testdata", "system_paths", "paths.d")
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

// Test helper function that mimics getShellSpecificPaths but uses test data
func getTestShellSpecificPaths(shell string, platformConfig PlatformConfig) []string {
	var additionalPaths []string
	
	switch shell {
	case "powershell":
		if platformConfig.PowerShell != nil && platformConfig.PowerShell.IncludeSystemPaths {
			if systemPaths, err := getTestSystemPaths(); err == nil {
				additionalPaths = append(additionalPaths, systemPaths...)
			}
		}
	}
	
	return additionalPaths
}

func TestHelpers_GetShellSpecificPaths(t *testing.T) {
	setupTestFilesystem(t)
	defer cleanupTestFilesystem()
	
	tests := []struct {
		name           string
		shell          string
		platformConfig PlatformConfig
		expectPaths    bool // Whether we expect any paths
	}{
		{
			name:  "bash no special paths",
			shell: "bash",
			platformConfig: PlatformConfig{
				Paths: []interface{}{"/usr/local/bin"},
			},
			expectPaths: false,
		},
		{
			name:  "powershell with include_system_paths false",
			shell: "powershell",
			platformConfig: PlatformConfig{
				Paths: []interface{}{"/usr/local/bin"},
				PowerShell: &ShellConfig{
					IncludeSystemPaths: false,
				},
			},
			expectPaths: false,
		},
		{
			name:  "powershell with include_system_paths true",
			shell: "powershell",
			platformConfig: PlatformConfig{
				Paths: []interface{}{"/usr/local/bin"},
				PowerShell: &ShellConfig{
					IncludeSystemPaths: true,
				},
			},
			expectPaths: true,
		},
		{
			name:  "powershell with nil config",
			shell: "powershell",
			platformConfig: PlatformConfig{
				Paths: []interface{}{"/usr/local/bin"},
			},
			expectPaths: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := getTestShellSpecificPaths(tt.shell, tt.platformConfig)
			
			if tt.expectPaths && len(paths) == 0 {
				t.Errorf("Expected some system paths for %s, got none", tt.name)
			}
			
			if !tt.expectPaths && len(paths) > 0 {
				t.Errorf("Expected no system paths for %s, got %d: %v", tt.name, len(paths), paths)
			}
		})
	}
}

// Test helper function that mimics countValidSystemPaths but uses test data
func countTestValidSystemPaths(shell string, platformConfig PlatformConfig) int {
	if shell != "powershell" || platformConfig.PowerShell == nil || !platformConfig.PowerShell.IncludeSystemPaths {
		return 0
	}
	
	systemPaths, err := getTestSystemPaths()
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

func TestHelpers_CountValidSystemPaths(t *testing.T) {
	setupTestFilesystem(t)
	defer cleanupTestFilesystem()
	
	tests := []struct {
		name           string
		shell          string
		platformConfig PlatformConfig
		expectCount    int // -1 means "don't check exact count, just check > 0"
	}{
		{
			name:  "bash shell no counting",
			shell: "bash",
			platformConfig: PlatformConfig{
				PowerShell: &ShellConfig{
					IncludeSystemPaths: true,
				},
			},
			expectCount: 0,
		},
		{
			name:  "powershell with include_system_paths false",
			shell: "powershell",
			platformConfig: PlatformConfig{
				PowerShell: &ShellConfig{
					IncludeSystemPaths: false,
				},
			},
			expectCount: 0,
		},
		{
			name:  "powershell with include_system_paths true",
			shell: "powershell",
			platformConfig: PlatformConfig{
				PowerShell: &ShellConfig{
					IncludeSystemPaths: true,
				},
			},
			expectCount: -1, // Should be > 0, exact count depends on system
		},
		{
			name:  "powershell with nil config",
			shell: "powershell",
			platformConfig: PlatformConfig{},
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := countTestValidSystemPaths(tt.shell, tt.platformConfig)
			
			if tt.expectCount == -1 {
				// Just check that we got some paths
				if count == 0 {
					t.Errorf("Expected some valid system paths for %s, got 0", tt.name)
				}
			} else {
				if count != tt.expectCount {
					t.Errorf("Expected %d valid system paths for %s, got %d", tt.expectCount, tt.name, count)
				}
			}
		})
	}
}

// Test the file reading functionality with edge cases
func TestHelpers_ReadPathsFileEdgeCases(t *testing.T) {
	setupTestFilesystem(t)
	defer cleanupTestFilesystem()
	
	// Create a temporary test file with various edge cases
	testContent := `# Comment at the start
/usr/local/bin

# Empty line above and comment
/opt/test/bin  
  /path/with/spaces  
	/path/with/tab/prefix

# Another comment
/final/path
# Comment at end`
	
	// Create temporary file
	tmpFile, err := os.CreateTemp("/tmp/pathuni/home/Pratt/.config/pathuni", "pathuni-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.WriteString(testContent); err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}
	tmpFile.Close()
	
	// Read the file
	paths, err := readPathsFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Unexpected error reading file: %v", err)
	}
	
	expectedPaths := []string{
		"/usr/local/bin",
		"/opt/test/bin",
		"/path/with/spaces",
		"/path/with/tab/prefix",
		"/final/path",
	}
	
	if len(paths) != len(expectedPaths) {
		t.Errorf("Expected %d paths, got %d. Expected: %v, Got: %v", 
			len(expectedPaths), len(paths), expectedPaths, paths)
		return
	}
	
	for i, expected := range expectedPaths {
		if paths[i] != expected {
			t.Errorf("Path mismatch at index %d. Expected: %q, Got: %q", i, expected, paths[i])
		}
	}
}

// Test error handling in readPathsFile
func TestHelpers_ReadPathsFileErrors(t *testing.T) {
	t.Run("permission denied", func(t *testing.T) {
		// Try to read a directory as a file (should cause error)
		_, err := readPathsFile("/")
		if err == nil {
			t.Error("Expected error when trying to read directory as file")
		}
	})
	
	t.Run("nonexistent file", func(t *testing.T) {
		_, err := readPathsFile("/nonexistent/file/path")
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
	})
}

// Integration test that mimics getSystemPaths behavior with our test data
func TestHelpers_SystemPathsIntegration(t *testing.T) {
	setupTestFilesystem(t)
	defer cleanupTestFilesystem()
	
	// This test simulates what getSystemPaths would do with our test data
	// We can't easily test getSystemPaths directly because it hardcodes /etc/paths
	
	testDataDir := filepath.Join("testdata", "system_paths")
	
	// Read the main paths file
	pathsFile := filepath.Join(testDataDir, "paths")
	mainPaths, err := readPathsFile(pathsFile)
	if err != nil {
		t.Fatalf("Failed to read main paths file: %v", err)
	}
	
	// Read files from paths.d directory
	pathsDDir := filepath.Join(testDataDir, "paths.d")
	entries, err := os.ReadDir(pathsDDir)
	if err != nil {
		t.Fatalf("Failed to read paths.d directory: %v", err)
	}
	
	var allPaths []string
	allPaths = append(allPaths, mainPaths...)
	
	for _, entry := range entries {
		if !entry.IsDir() {
			filePath := filepath.Join(pathsDDir, entry.Name())
			paths, err := readPathsFile(filePath)
			if err == nil {
				allPaths = append(allPaths, paths...)
			}
		}
	}
	
	// Verify we got expected paths from our test data
	expectedPaths := []string{
		"/tmp/pathuni/usr/local/bin", "/tmp/pathuni/usr/bin", "/tmp/pathuni/bin", "/tmp/pathuni/usr/sbin", "/tmp/pathuni/sbin", // from main paths
		"/tmp/pathuni/opt/homebrew/bin", "/tmp/pathuni/opt/homebrew/sbin",                  // from homebrew
		"/tmp/pathuni/usr/local/go/bin", "/tmp/pathuni/usr/local/node/bin",                 // from user_paths
	}
	
	if len(allPaths) != len(expectedPaths) {
		t.Errorf("Expected %d total paths, got %d. Expected: %v, Got: %v", 
			len(expectedPaths), len(allPaths), expectedPaths, allPaths)
		return
	}
	
	// Check that all expected paths are present (order might differ)
	pathMap := make(map[string]bool)
	for _, path := range allPaths {
		pathMap[path] = true
	}
	
	for _, expected := range expectedPaths {
		if !pathMap[expected] {
			t.Errorf("Expected path %q not found in results: %v", expected, allPaths)
		}
	}
}

// Test path trimming and comment filtering
func TestHelpers_PathProcessing(t *testing.T) {
	setupTestFilesystem(t)
	defer cleanupTestFilesystem()
	
	// Create a test file with various whitespace and comment scenarios
	testCases := []string{
		"  /path/with/leading/spaces  ",
		"\t/path/with/tab\t",
		"/normal/path",
		"# This is a comment",
		"   # Comment with spaces",
		"",
		"   ",
		"/path/after/empty/lines",
	}
	
	tmpFile, err := os.CreateTemp("/tmp/pathuni/home/Pratt/.config/pathuni", "pathuni-whitespace-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	content := strings.Join(testCases, "\n")
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}
	tmpFile.Close()
	
	paths, err := readPathsFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	expectedPaths := []string{
		"/path/with/leading/spaces",
		"/path/with/tab",
		"/normal/path",
		"/path/after/empty/lines",
	}
	
	if len(paths) != len(expectedPaths) {
		t.Errorf("Expected %d paths, got %d. Expected: %v, Got: %v", 
			len(expectedPaths), len(paths), expectedPaths, paths)
		return
	}
	
	for i, expected := range expectedPaths {
		if paths[i] != expected {
			t.Errorf("Path mismatch at index %d. Expected: %q, Got: %q", i, expected, paths[i])
		}
	}
}