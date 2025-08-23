package main

import (
	"path/filepath"
	"testing"
)

func TestDryRun_DetailedOutput(t *testing.T) {
	setupTestFilesystem(t)
	defer cleanupTestFilesystem()
	
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
	setupTestFilesystem(t)
	defer cleanupTestFilesystem()
	
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

// TestDryRunV2_SkipReasons tests the new tree-structured output with skip reasons
func TestDryRunV2_SkipReasons(t *testing.T) {
	setupTestFilesystem(t)
	defer cleanupTestFilesystem()
	
	testConfigPath := filepath.Join("testdata", "dry_run_tag_filtering.yaml")
	
	tests := []struct {
		name           string
		includeFlag    string
		excludeFlag    string
		expectedPaths  map[string]bool // path -> should be included
		expectedReasons map[string]string // path -> expected skip reason type
	}{
		{
			name:        "complex tag filtering scenario",
			includeFlag: "dev+work,audio",
			excludeFlag: "gaming,video",
			expectedPaths: map[string]bool{
				"/tmp/pathuni/usr/local/bin":                                           true,  // untagged
				"/tmp/pathuni/usr/local/go/bin":                                        true,  // dev+work
				"/tmp/pathuni/home/Pratt/.cargo/bin":                                   true,  // dev+work+audio
				"/tmp/pathuni/Applications/Docker.app/Contents/Resources/bin":          true,  // audio
				"/tmp/pathuni/home/Pratt/.npm-global/bin":                              true,  // audio+personal
				"/tmp/pathuni/opt/games/bin":                                           false, // gaming (exclude)
				"/tmp/pathuni/System/Library/Frameworks":                               false, // video (exclude)
				"/tmp/pathuni/usr/local/node/bin":                                      false, // gaming wins over dev+work
				"/tmp/pathuni/Applications/Xcode.app/Contents/Developer/usr/bin":       false, // video wins over audio
				"/tmp/pathuni/home/Pratt/.local/bin":                                   false, // dev only (needs work)
				"/tmp/pathuni/opt/homebrew/bin":                                        false, // work only (needs dev)
				"/tmp/pathuni/usr/games":                                               false, // personal only
				"/tmp/pathuni/nonexistent/path":                                        false, // not found
			},
			expectedReasons: map[string]string{
				"/tmp/pathuni/opt/games/bin":                                           "tags", // gaming = gaming
				"/tmp/pathuni/System/Library/Frameworks":                               "tags", // video = video
				"/tmp/pathuni/usr/local/node/bin":                                      "tags", // gaming = gaming (exclude wins)
				"/tmp/pathuni/Applications/Xcode.app/Contents/Developer/usr/bin":       "tags", // video = video (exclude wins)
				"/tmp/pathuni/home/Pratt/.local/bin":                                   "tags", // dev != dev+work,audio
				"/tmp/pathuni/opt/homebrew/bin":                                        "tags", // work != dev+work,audio
				"/tmp/pathuni/usr/games":                                               "tags", // personal != dev+work,audio
				"/tmp/pathuni/nonexistent/path":                                        "not_found", // not found
			},
		},
		{
			name:        "no filters - all tagged paths included",
			includeFlag: "",
			excludeFlag: "",
			expectedPaths: map[string]bool{
				"/tmp/pathuni/usr/local/bin":                                           true,
				"/tmp/pathuni/usr/local/go/bin":                                        true,
				"/tmp/pathuni/home/Pratt/.cargo/bin":                                   true,
				"/tmp/pathuni/Applications/Docker.app/Contents/Resources/bin":          true,
				"/tmp/pathuni/home/Pratt/.npm-global/bin":                              true,
				"/tmp/pathuni/opt/games/bin":                                           true,
				"/tmp/pathuni/System/Library/Frameworks":                               true,
				"/tmp/pathuni/usr/local/node/bin":                                      true,
				"/tmp/pathuni/Applications/Xcode.app/Contents/Developer/usr/bin":       true,
				"/tmp/pathuni/home/Pratt/.local/bin":                                   true,
				"/tmp/pathuni/opt/homebrew/bin":                                        true,
				"/tmp/pathuni/usr/games":                                               true,
				"/tmp/pathuni/nonexistent/path":                                        false, // still not found
			},
			expectedReasons: map[string]string{
				"/tmp/pathuni/nonexistent/path": "not_found",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse tag filters
			tagFilter, err := parseTagFlags(tt.includeFlag, tt.excludeFlag)
			if err != nil {
				t.Fatalf("Failed to parse tag flags: %v", err)
			}
			
			// Test the new evaluation function
			result, err := EvaluateConfigWithReasons(testConfigPath, "macOS", "bash", false, tagFilter)
			if err != nil {
				t.Fatalf("EvaluateConfigWithReasons failed: %v", err)
			}
			
			// Check included paths
			includedMap := make(map[string]bool)
			for _, path := range result.IncludedPaths {
				includedMap[path] = true
			}
			
			// Check skipped paths and reasons
			skippedMap := make(map[string][]SkipReason)
			for _, skipped := range result.SkippedPaths {
				skippedMap[skipped.Path] = skipped.Reasons
			}
			
			// Verify each expected path
			for expectedPath, shouldBeIncluded := range tt.expectedPaths {
				if shouldBeIncluded {
					if !includedMap[expectedPath] {
						t.Errorf("Path %s should be included but was not", expectedPath)
					}
				} else {
					if includedMap[expectedPath] {
						t.Errorf("Path %s should be skipped but was included", expectedPath)
					}
					
					// Check that skipped path has the expected reason type
					if expectedReasonType, hasExpectedReason := tt.expectedReasons[expectedPath]; hasExpectedReason {
						reasons, isSkipped := skippedMap[expectedPath]
						if !isSkipped {
							t.Errorf("Path %s should be skipped with reason but was not found in skipped paths", expectedPath)
						} else if len(reasons) == 0 {
							t.Errorf("Path %s is skipped but has no reasons", expectedPath)
						} else if reasons[0].Type != expectedReasonType {
							t.Errorf("Path %s has reason type %s, expected %s", expectedPath, reasons[0].Type, expectedReasonType)
						}
					}
				}
			}
		})
	}
}

// TestRenderSkippedPath tests the tree rendering function
func TestRenderSkippedPath(t *testing.T) {
	tests := []struct {
		name     string
		skipped  SkippedPath
		expected string
	}{
		{
			name: "single tag reason",
			skipped: SkippedPath{
				Path:    "/opt/gaming/bin",
				Reasons: []SkipReason{{Type: "tags", Detail: "gaming = gaming"}},
			},
			expected: "  [-] /opt/gaming/bin\n       └gaming = gaming",
		},
		{
			name: "multiple reasons",
			skipped: SkippedPath{
				Path: "/opt/mixed/bin",
				Reasons: []SkipReason{
					{Type: "hostname", Detail: "work-laptop = *-work"},
					{Type: "tags", Detail: "gaming != dev"},
				},
			},
			expected: "  [-] /opt/mixed/bin\n       ├work-laptop = *-work\n       └gaming != dev",
		},
		{
			name: "not found path",
			skipped: SkippedPath{
				Path:    "/nonexistent/path",
				Reasons: []SkipReason{{Type: "not_found", Detail: "not found"}},
			},
			expected: "  [!] /nonexistent/path (not found)",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderSkippedPath(tt.skipped)
			if result != tt.expected {
				t.Errorf("renderSkippedPath() =\n%s\n\nExpected:\n%s", result, tt.expected)
			}
		})
	}
}