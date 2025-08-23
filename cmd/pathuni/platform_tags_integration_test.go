package main

import (
	"path/filepath"
	"testing"
)

// TestPlatformTags_BasicInheritance tests basic platform-level tag inheritance scenarios
func TestPlatformTags_BasicInheritance(t *testing.T) {
	setupTestFilesystem(t)
	defer cleanupTestFilesystem()

	testConfigPath := filepath.Join("testdata", "platform_tags_basic.yaml")

	tests := []struct {
		name              string
		includeFlag       string
		excludeFlag       string
		expectedIncluded  int
		expectedSkipped   int
	}{
		{
			name:             "no filters - all paths included",
			includeFlag:      "",
			excludeFlag:      "",
			expectedIncluded: 5, // All existing paths
			expectedSkipped:  0,
		},
		{
			name:             "include essential tag (from all platform)",
			includeFlag:      "essential",
			excludeFlag:      "",
			expectedIncluded: 2, // /usr/local/bin, /usr/bin inherit [base, essential]
			expectedSkipped:  3, // Others don't have essential tag
		},
		{
			name:             "include mac tag (from macOS platform)",
			includeFlag:      "mac",
			excludeFlag:      "",
			expectedIncluded: 1, // /opt/homebrew/bin inherits [mac, desktop]
			expectedSkipped:  4, // Others don't have mac tag
		},
		{
			name:             "include dev tag (explicit override)",
			includeFlag:      "dev",
			excludeFlag:      "",
			expectedIncluded: 1, // /opt/dev/bin has explicit [dev, work]
			expectedSkipped:  4, // Others don't have dev tag
		},
		{
			name:             "exclude essential tag",
			includeFlag:      "",
			excludeFlag:      "essential",
			expectedIncluded: 3, // All except the ones with essential tag
			expectedSkipped:  2, // /usr/local/bin, /usr/bin have essential
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tagFilter, err := parseTagFlags(tt.includeFlag, tt.excludeFlag)
			if err != nil {
				t.Fatalf("Failed to parse tag flags: %v", err)
			}

			validPaths, skippedPaths, _, err := EvaluateConfig(testConfigPath, "macOS", "bash", false, tagFilter)
			if err != nil {
				t.Fatalf("EvaluateConfig failed: %v", err)
			}

			if len(validPaths) != tt.expectedIncluded {
				t.Errorf("Expected %d included paths, got %d: %v", tt.expectedIncluded, len(validPaths), validPaths)
			}

			if len(skippedPaths) != tt.expectedSkipped {
				t.Errorf("Expected %d skipped paths, got %d: %v", tt.expectedSkipped, len(skippedPaths), skippedPaths)
			}
		})
	}
}

// TestPlatformTags_BackwardsCompatibility tests that existing configs without platform tags still work
func TestPlatformTags_BackwardsCompatibility(t *testing.T) {
	setupTestFilesystem(t)
	defer cleanupTestFilesystem()

	testConfigPath := filepath.Join("testdata", "platform_tags_backwards_compat.yaml")

	// Test with no tag filters - should work exactly like before
	validPaths, skippedPaths, _, err := EvaluateConfig(testConfigPath, "macOS", "bash", false, TagFilter{})
	if err != nil {
		t.Fatalf("Backwards compatibility test failed: %v", err)
	}

	// Should include all existing paths
	expectedPaths := 3 // /usr/local/bin (untagged), /opt/homebrew/bin (untagged), /opt/dev/bin (tagged)
	totalPaths := len(validPaths) + len(skippedPaths)

	if totalPaths != expectedPaths {
		t.Errorf("Expected %d total paths, got %d (valid: %d, skipped: %d)", 
			expectedPaths, totalPaths, len(validPaths), len(skippedPaths))
	}

	// Test with tag filtering - should work on explicit tags only
	tagFilter := TagFilter{Include: [][]string{{"dev"}}}
	validPaths, skippedPaths, _, err = EvaluateConfig(testConfigPath, "macOS", "bash", false, tagFilter)
	if err != nil {
		t.Fatalf("Tag filtering test failed: %v", err)
	}

	// Should include untagged paths + paths with 'dev' tag
	if len(validPaths) != 3 { // 2 untagged + 1 with dev tag
		t.Errorf("Expected 3 valid paths with dev filter, got %d: %v", len(validPaths), validPaths)
	}
}

// TestPlatformTags_ComplexScenarios tests complex inheritance and filtering scenarios
func TestPlatformTags_ComplexScenarios(t *testing.T) {
	setupTestFilesystem(t)
	defer cleanupTestFilesystem()

	testConfigPath := filepath.Join("testdata", "platform_tags_complex.yaml")

	tests := []struct {
		name            string
		includeFlag     string
		excludeFlag     string
		expectedIncluded []string
		expectedSkipped  []string
	}{
		{
			name:        "include system tag (from all platform)",
			includeFlag: "system",
			excludeFlag: "",
			expectedIncluded: []string{"/tmp/pathuni/usr/local/bin"}, // Only inherits [system]
			// Other paths don't have system tag
		},
		{
			name:        "include mac tag (mixed inheritance and explicit)",
			includeFlag: "mac",
			excludeFlag: "",
			expectedIncluded: []string{
				"/tmp/pathuni/opt/homebrew/bin", // Inherits [mac, gui]
				"/tmp/pathuni/Applications/Docker.app/Contents/Resources/bin", // Explicit [mac, gui, docker]
			},
		},
		{
			name:        "exclude gui tag",
			includeFlag: "",
			excludeFlag: "gui",
			expectedIncluded: []string{
				"/tmp/pathuni/usr/local/bin", // [system] - no gui
				"/tmp/pathuni/opt/work/bin",  // [work, server] - no gui
			},
			// /opt/homebrew/bin and Docker.app both have gui and get excluded
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tagFilter, err := parseTagFlags(tt.includeFlag, tt.excludeFlag)
			if err != nil {
				t.Fatalf("Failed to parse tag flags: %v", err)
			}

			validPaths, skippedPaths, _, err := EvaluateConfig(testConfigPath, "macOS", "bash", false, tagFilter)
			if err != nil {
				t.Fatalf("EvaluateConfig failed: %v", err)
			}

			// Check included paths
			if len(validPaths) != len(tt.expectedIncluded) {
				t.Errorf("Expected %d included paths, got %d", len(tt.expectedIncluded), len(validPaths))
				t.Errorf("Expected: %v", tt.expectedIncluded)
				t.Errorf("Got: %v", validPaths)
			}

			// Verify specific included paths
			for _, expectedPath := range tt.expectedIncluded {
				found := false
				for _, actualPath := range validPaths {
					if actualPath == expectedPath {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected path %s not found in included paths: %v", expectedPath, validPaths)
				}
			}

			t.Logf("Test %s: included=%v, skipped=%v", tt.name, validPaths, skippedPaths)
		})
	}
}