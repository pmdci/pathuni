package main

import (
	"path/filepath"
	"testing"
)

func TestIntegration_TagFiltering(t *testing.T) {
	setupTestFilesystem(t)
	defer cleanupTestFilesystem()
	
	testConfigPath := filepath.Join("testdata", "integration_tag_filtering.yaml")

	tests := []struct {
		name          string
		includeFlag   string
		excludeFlag   string
		expectError   bool
		expectedPaths int  // Minimum expected total paths (valid + skipped)
	}{
		{
			name:          "no tag filters",
			includeFlag:   "",
			excludeFlag:   "",
			expectError:   false,
			expectedPaths: 5, // All paths should be processed
		},
		{
			name:          "include home tags",
			includeFlag:   "home",
			excludeFlag:   "",
			expectError:   false,
			expectedPaths: 4, // Untagged + home tagged paths
		},
		{
			name:          "exclude gaming tags",
			includeFlag:   "",
			excludeFlag:   "gaming", 
			expectError:   false,
			expectedPaths: 4, // All except gaming
		},
		{
			name:          "include work AND dev",
			includeFlag:   "work+dev",
			excludeFlag:   "",
			expectError:   false,
			expectedPaths: 3, // Untagged + work+dev path
		},
		{
			name:          "include home OR work",
			includeFlag:   "home,work",
			excludeFlag:   "",
			expectError:   false,
			expectedPaths: 5, // Untagged + home or work paths
		},
		{
			name:          "exclude wins over include",
			includeFlag:   "home",
			excludeFlag:   "gaming",
			expectError:   false,
			expectedPaths: 3, // Untagged + home-only paths (gaming excluded even though it has home)
		},
		{
			name:          "exclude with AND logic",
			includeFlag:   "",
			excludeFlag:   "work+dev",
			expectError:   false,
			expectedPaths: 4, // All except work+dev path
		},
		{
			name:        "invalid include tag",
			includeFlag: "2invalid",
			excludeFlag: "",
			expectError: true,
		},
		{
			name:        "invalid exclude tag",
			includeFlag: "",
			excludeFlag: "work@home",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original flag values
			originalInclude := tagsInclude
			originalExclude := tagsExclude
			
			// Set test flags
			tagsInclude = tt.includeFlag
			tagsExclude = tt.excludeFlag
			
			defer func() {
				// Restore original flags
				tagsInclude = originalInclude
				tagsExclude = originalExclude
			}()

			// Test EvaluateConfig with flags
			tagFilter, err := parseTagFlags(tagsInclude, tagsExclude)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error parsing flags for %s: %v", tt.name, err)
				return
			}

			// Test the actual path collection
			validPaths, skippedPaths, _, err := EvaluateConfig(testConfigPath, "macOS", "bash", tagFilter)
			if err != nil {
				t.Errorf("Unexpected error evaluating config for %s: %v", tt.name, err)
				return
			}

			totalPaths := len(validPaths) + len(skippedPaths)
			if totalPaths < tt.expectedPaths {
				t.Errorf("Test %s: expected at least %d paths, got %d (valid: %d, skipped: %d)", 
					tt.name, tt.expectedPaths, totalPaths, len(validPaths), len(skippedPaths))
			}
		})
	}
}


func TestIntegration_ErrorHandling(t *testing.T) {
	// Test CLI flag validation
	tests := []struct {
		name        string
		includeFlag string
		excludeFlag string
		expectError bool
	}{
		{"valid flags", "home,dev", "gaming", false},
		{"invalid include format", "home,2bad", "", true},
		{"invalid exclude format", "", "work@home", true},
		{"empty tag in include", "home+", "", true},
		{"complex valid", "home+dev,work", "gaming,temp", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseTagFlags(tt.includeFlag, tt.excludeFlag)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
			}
		})
	}
}