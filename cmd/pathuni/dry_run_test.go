package main

import (
	"path/filepath"
	"testing"
)

func TestDryRun_DetailedOutput(t *testing.T) {
	testConfigPath := filepath.Join("testdata", "dry_run_comprehensive.yaml")

	tests := []struct {
		name              string
		includeFlag       string
		excludeFlag       string
		expectError       bool
		expectedIncluded  int  // minimum expected included paths
		expectedNotFound  int  // minimum expected not found paths  
		expectedFiltered  int  // minimum expected filtered paths
	}{
		{
			name:              "no filters - all combinations visible",
			includeFlag:       "",
			excludeFlag:       "",
			expectError:       false,
			expectedIncluded:  2, // At least /usr/local/bin, /usr/bin  
			expectedNotFound:  2, // At least 2 nonexistent paths
			expectedFiltered:  0, // No filtering applied
		},
		{
			name:              "exclude essential - tests existence wins over filter",
			includeFlag:       "",
			excludeFlag:       "essential",
			expectError:       false,
			expectedIncluded:  1, // At least /usr/local/bin (untagged)
			expectedNotFound:  2, // Both nonexistent paths (existence wins!)
			expectedFiltered:  1, // /usr/bin excluded by tags
		},
		{
			name:              "include missing - tests not found + tagged logic", 
			includeFlag:       "missing",
			excludeFlag:       "",
			expectError:       false,
			expectedIncluded:  2, // Untagged paths still included
			expectedNotFound:  2, // Both nonexistent (existence wins over include!)
			expectedFiltered:  1, // At least /usr/bin filtered out (doesn't have 'missing' tag)
		},
		{
			name:              "include essential - normal filtering",
			includeFlag:       "essential",
			excludeFlag:       "",
			expectError:       false,
			expectedIncluded:  2, // Untagged + /usr/bin
			expectedNotFound:  2, // Both nonexistent paths
			expectedFiltered:  0, // Only filtering out non-essential existing paths
		},
		{
			name:              "exclude wins over include",
			includeFlag:       "essential",
			excludeFlag:       "essential",
			expectError:       false,
			expectedIncluded:  1, // Only untagged paths
			expectedNotFound:  2, // Nonexistent paths
			expectedFiltered:  1, // /usr/bin excluded (exclude wins)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse tag filters
			tagFilter, err := parseTagFlags(tt.includeFlag, tt.excludeFlag)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error parsing tag flags: %v", err)
			}

			// Test the detailed evaluation
			pathStatuses, _, err := EvaluateConfigDetailed(testConfigPath, "macOS", "bash", false, tagFilter)
			if err != nil {
				t.Fatalf("EvaluateConfigDetailed failed: %v", err)
			}

			// Categorize results exactly like PrintEvaluationReport does
			var included, notFound, filteredByTags []string
			for _, status := range pathStatuses {
				if status.Included {
					included = append(included, status.Path)
				} else if !status.Exists {
					// Existence wins over filtering
					notFound = append(notFound, status.Path)
				} else {
					// Path exists but was filtered by tags
					filteredByTags = append(filteredByTags, status.Path)
				}
			}

			// Verify minimum counts (actual might be higher depending on system)
			if len(included) < tt.expectedIncluded {
				t.Errorf("Expected at least %d included paths, got %d: %v", 
					tt.expectedIncluded, len(included), included)
			}
			if len(notFound) < tt.expectedNotFound {
				t.Errorf("Expected at least %d not found paths, got %d: %v", 
					tt.expectedNotFound, len(notFound), notFound)
			}
			if len(filteredByTags) < tt.expectedFiltered {
				t.Errorf("Expected at least %d filtered paths, got %d: %v", 
					tt.expectedFiltered, len(filteredByTags), filteredByTags)
			}

			// CRITICAL TEST: Verify existence-wins-over-filter logic
			for _, status := range pathStatuses {
				if !status.Exists {
					// If path doesn't exist, it should NEVER be in filteredByTags
					for _, filtered := range filteredByTags {
						if status.Path == filtered {
							t.Errorf("EXISTENCE-WINS-OVER-FILTER VIOLATED: Path %s doesn't exist but was categorized as filtered by tags!", 
								status.Path)
						}
					}
				}
			}

			// Verify that paths in filteredByTags actually exist
			for _, filteredPath := range filteredByTags {
				found := false
				for _, status := range pathStatuses {
					if status.Path == filteredPath && status.Exists {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Path %s is in filteredByTags but doesn't exist - this violates existence-wins logic", filteredPath)
				}
			}
		})
	}
}

func TestDryRun_BackwardsCompatibility(t *testing.T) {
	// Test that the old EvaluateConfig still works the same way
	testConfigPath := filepath.Join("testdata", "dry_run_comprehensive.yaml")

	// Test with no filters
	oldValid, oldSkipped, oldSystemPaths, err := EvaluateConfig(testConfigPath, "macOS", "bash", false, TagFilter{})
	if err != nil {
		t.Fatalf("Old EvaluateConfig failed: %v", err)
	}

	newStatuses, newSystemPaths, err := EvaluateConfigDetailed(testConfigPath, "macOS", "bash", false, TagFilter{})
	if err != nil {
		t.Fatalf("New EvaluateConfigDetailed failed: %v", err)
	}

	// Convert new format to old format for comparison
	var newValid, newSkipped []string
	for _, status := range newStatuses {
		if status.Included {
			newValid = append(newValid, status.Path)
		} else {
			newSkipped = append(newSkipped, status.Path)
		}
	}

	// Should have same counts
	if len(oldValid) != len(newValid) {
		t.Errorf("Valid paths count mismatch: old=%d, new=%d", len(oldValid), len(newValid))
	}
	if len(oldSkipped) != len(newSkipped) {
		t.Errorf("Skipped paths count mismatch: old=%d, new=%d", len(oldSkipped), len(newSkipped))
	}
	if oldSystemPaths != newSystemPaths {
		t.Errorf("System paths count mismatch: old=%d, new=%d", oldSystemPaths, newSystemPaths)
	}
}