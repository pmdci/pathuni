package main

import (
	"strings"
	"testing"
)

func TestTag_Validation(t *testing.T) {
	tests := []struct {
		name        string
		tag         string
		expectError bool
	}{
		// Valid tags
		{"valid short", "dev", false},
		{"valid with numbers", "work2", false},
		{"valid with underscores", "work_laptop", false},
		{"valid max length", "a1234567890123456789", false}, // 20 chars
		{"valid mixed case", "workLaptop", false},
		{"valid starts with capital", "Home", false},
		
		// Invalid tags - too short
		{"too short 1 char", "a", true},
		{"too short 2 chars", "ab", true},
		
		// Invalid tags - too long
		{"too long 21 chars", "a12345678901234567890", true}, // 21 chars
		{"way too long", "this_is_a_very_long_tag_name_that_exceeds_limits", true},
		
		// Invalid tags - bad start character
		{"starts with number", "2fast", true},
		{"starts with underscore", "_private", true},
		{"starts with hyphen", "-temp", true},
		
		// Invalid tags - bad characters
		{"contains hyphen", "work-laptop", true},
		{"contains space", "work laptop", true},
		{"contains special chars", "work@home", true},
		{"contains emoji", "fun😀", true},
		{"contains period", "work.home", true},
		{"contains slash", "home/work", true},
		
		// Edge cases
		{"empty string", "", true},
		{"only underscores", "___", true},
		{"only numbers", "123", true},
		
		// Valid wildcard patterns
		{"wildcard asterisk", "work_*", false},
		{"wildcard question mark", "dev?", false},
		{"wildcard character class", "[abc]*", false},
		{"wildcard range", "[a-z]*", false},
		{"wildcard negated class", "[^test]*", false},
		{"complex wildcard", "server[1-3]*", false},
		{"multiple wildcards", "*_temp*", false},
		
		// Invalid wildcard patterns  
		{"invalid wildcard unmatched bracket", "work_[", true},
		{"invalid wildcard empty class", "work_[]", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTag(tt.tag)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error for tag '%s', but got none", tt.tag)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for tag '%s': %v", tt.tag, err)
			}
		})
	}
}

func TestTag_ValidateTags(t *testing.T) {
	tests := []struct {
		name        string
		tags        []string
		context     string
		expectError bool
		errorContains string
	}{
		{
			name:        "valid tags",
			tags:        []string{"home", "dev", "work_laptop"},
			context:     "test path",
			expectError: false,
		},
		{
			name:        "empty tags array",
			tags:        []string{},
			context:     "test path",
			expectError: false,
		},
		{
			name:        "nil tags",
			tags:        nil,
			context:     "test path", 
			expectError: false,
		},
		{
			name:        "duplicate tags",
			tags:        []string{"home", "dev", "home"},
			context:     "test path",
			expectError: true,
			errorContains: "duplicate tag 'home'",
		},
		{
			name:        "case insensitive duplicates",
			tags:        []string{"Home", "dev", "home"},
			context:     "test path",
			expectError: true,
			errorContains: "duplicate tag 'home'",
		},
		{
			name:        "invalid tag format",
			tags:        []string{"home", "2fast", "dev"},
			context:     "test path",
			expectError: true,
			errorContains: "invalid tag '2fast'",
		},
		{
			name:        "multiple issues - duplicate wins",
			tags:        []string{"home", "home", "2fast"},
			context:     "test path",
			expectError: true,
			errorContains: "duplicate tag 'home'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTags(tt.tags, tt.context)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for tags %v, but got none", tt.tags)
				} else if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', but got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for tags %v: %v", tt.tags, err)
				}
			}
		})
	}
}

func TestTag_ParseTagFilter(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		expected    [][]string
		expectError bool
	}{
		{
			name:     "empty filter",
			filter:   "",
			expected: nil,
		},
		{
			name:     "single tag",
			filter:   "home",
			expected: [][]string{{"home"}},
		},
		{
			name:     "comma separated - OR logic",
			filter:   "home,dev",
			expected: [][]string{{"home"}, {"dev"}},
		},
		{
			name:     "plus separated - AND logic", 
			filter:   "home+dev",
			expected: [][]string{{"home", "dev"}},
		},
		{
			name:     "mixed OR and AND",
			filter:   "home,work+server",
			expected: [][]string{{"home"}, {"work", "server"}},
		},
		{
			name:     "complex combination",
			filter:   "home+dev,work,gaming+temp",
			expected: [][]string{{"home", "dev"}, {"work"}, {"gaming", "temp"}},
		},
		{
			name:     "whitespace handling",
			filter:   " home , dev + server ",
			expected: [][]string{{"home"}, {"dev", "server"}},
		},
		
		// Error cases
		{
			name:        "invalid tag",
			filter:      "home,2fast",
			expectError: true,
		},
		{
			name:        "empty tag in AND group",
			filter:      "home+",
			expectError: true,
		},
		{
			name:     "empty tag in OR group - skipped",
			filter:   "home,,dev", 
			expected: [][]string{{"home"}, {"dev"}},
		},
		{
			name:        "only separators",
			filter:      "++,,",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTagFilter(tt.filter)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for filter '%s', but got none", tt.filter)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error for filter '%s': %v", tt.filter, err)
				return
			}
			
			// Compare results
			if !equalStringSlices2D(result, tt.expected) {
				t.Errorf("Filter '%s': expected %v, got %v", tt.filter, tt.expected, result)
			}
		})
	}
}

func TestTag_ParseTagFlags(t *testing.T) {
	tests := []struct {
		name            string
		includeFlag     string
		excludeFlag     string
		expectedInclude [][]string
		expectedExclude [][]string
		expectError     bool
	}{
		{
			name:            "empty flags",
			includeFlag:     "",
			excludeFlag:     "",
			expectedInclude: nil,
			expectedExclude: nil,
		},
		{
			name:            "include only",
			includeFlag:     "home,dev",
			excludeFlag:     "",
			expectedInclude: [][]string{{"home"}, {"dev"}},
			expectedExclude: nil,
		},
		{
			name:            "exclude only",
			includeFlag:     "",
			excludeFlag:     "gaming",
			expectedInclude: nil,
			expectedExclude: [][]string{{"gaming"}},
		},
		{
			name:            "both flags",
			includeFlag:     "home+dev",
			excludeFlag:     "gaming,temp",
			expectedInclude: [][]string{{"home", "dev"}},
			expectedExclude: [][]string{{"gaming"}, {"temp"}},
		},
		{
			name:        "invalid include flag",
			includeFlag: "2fast",
			excludeFlag: "",
			expectError: true,
		},
		{
			name:        "invalid exclude flag",
			includeFlag: "",
			excludeFlag: "work@home",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTagFlags(tt.includeFlag, tt.excludeFlag)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if !equalStringSlices2D(result.Include, tt.expectedInclude) {
				t.Errorf("Include: expected %v, got %v", tt.expectedInclude, result.Include)
			}
			
			if !equalStringSlices2D(result.Exclude, tt.expectedExclude) {
				t.Errorf("Exclude: expected %v, got %v", tt.expectedExclude, result.Exclude)
			}
		})
	}
}

func TestTag_MatchesTagConditions(t *testing.T) {
	tests := []struct {
		name       string
		pathTags   []string
		conditions [][]string
		expected   bool
	}{
		{
			name:       "empty conditions",
			pathTags:   []string{"home", "dev"},
			conditions: [][]string{},
			expected:   false,
		},
		{
			name:       "single condition match",
			pathTags:   []string{"home", "dev"},
			conditions: [][]string{{"home"}},
			expected:   true,
		},
		{
			name:       "single condition no match",
			pathTags:   []string{"home", "dev"},
			conditions: [][]string{{"work"}},
			expected:   false,
		},
		{
			name:       "AND condition match",
			pathTags:   []string{"home", "dev", "laptop"},
			conditions: [][]string{{"home", "dev"}},
			expected:   true,
		},
		{
			name:       "AND condition partial match",
			pathTags:   []string{"home", "laptop"},
			conditions: [][]string{{"home", "dev"}},
			expected:   false,
		},
		{
			name:       "OR conditions - first matches",
			pathTags:   []string{"home"},
			conditions: [][]string{{"home"}, {"work"}},
			expected:   true,
		},
		{
			name:       "OR conditions - second matches",
			pathTags:   []string{"work"},
			conditions: [][]string{{"home"}, {"work"}},
			expected:   true,
		},
		{
			name:       "OR conditions - none match",
			pathTags:   []string{"gaming"},
			conditions: [][]string{{"home"}, {"work"}},
			expected:   false,
		},
		{
			name:       "complex OR and AND",
			pathTags:   []string{"work", "server", "prod"},
			conditions: [][]string{{"home", "dev"}, {"work", "server"}},
			expected:   true,
		},
		{
			name:       "case insensitive matching",
			pathTags:   []string{"Home", "Dev"},
			conditions: [][]string{{"home", "dev"}},
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesTagConditions(tt.pathTags, tt.conditions)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for tags %v with conditions %v", 
					tt.expected, result, tt.pathTags, tt.conditions)
			}
		})
	}
}

func TestTag_WildcardMatching(t *testing.T) {
	tests := []struct {
		name       string
		pathTags   []string
		conditions [][]string
		expected   bool
	}{
		// Basic wildcard patterns
		{
			name:       "asterisk wildcard matches",
			pathTags:   []string{"work_prod", "dev"},
			conditions: [][]string{{"work_*"}},
			expected:   true,
		},
		{
			name:       "asterisk wildcard no match",
			pathTags:   []string{"home", "dev"},
			conditions: [][]string{{"work_*"}},
			expected:   false,
		},
		{
			name:       "question mark wildcard matches",
			pathTags:   []string{"hunt", "gaming"},
			conditions: [][]string{{"?unt"}},
			expected:   true,
		},
		{
			name:       "question mark wildcard no match",
			pathTags:   []string{"blunt", "gaming"}, // blunt has 5 chars, ?unt expects 4
			conditions: [][]string{{"?unt"}},
			expected:   false,
		},
		// Case insensitive wildcard matching (critical test from spec)
		{
			name:       "case insensitive wildcard a?l matches ALL",
			pathTags:   []string{"ALL", "general"},
			conditions: [][]string{{"a?l"}},
			expected:   true,
		},
		{
			name:       "case insensitive wildcard server* matches Server3",
			pathTags:   []string{"Server3", "backend"},
			conditions: [][]string{{"server*"}},
			expected:   true,
		},
		// Character class patterns
		{
			name:       "character class matches",
			pathTags:   []string{"server1", "production"},
			conditions: [][]string{{"server[123]"}},
			expected:   true,
		},
		{
			name:       "character class no match",
			pathTags:   []string{"server4", "production"},
			conditions: [][]string{{"server[123]"}},
			expected:   false,
		},
		// Complex patterns
		{
			name:       "complex pattern *_temp matches",
			pathTags:   []string{"build_temp", "temporary"},
			conditions: [][]string{{"*_temp"}},
			expected:   true,
		},
		{
			name:       "multiple wildcard conditions OR logic",
			pathTags:   []string{"work_dev", "rust"},
			conditions: [][]string{{"work_*"}, {"server*"}}, // OR logic
			expected:   true,
		},
		// AND logic with wildcards  
		{
			name:       "AND logic with wildcard matches",
			pathTags:   []string{"work_prod", "dev"},
			conditions: [][]string{{"work_*", "dev"}}, // AND logic
			expected:   true,
		},
		{
			name:       "AND logic with wildcard partial match",
			pathTags:   []string{"work_prod", "rust"}, // has work_* but missing dev
			conditions: [][]string{{"work_*", "dev"}}, // AND logic
			expected:   false,
		},
		// Mixed exact and wildcard
		{
			name:       "mixed exact and wildcard matching",
			pathTags:   []string{"home", "server1"},
			conditions: [][]string{{"home"}, {"server*"}}, // OR logic
			expected:   true,
		},
		{
			name:       "mixed exact and wildcard no match",
			pathTags:   []string{"office", "database"},
			conditions: [][]string{{"home"}, {"server*"}}, // OR logic
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesTagConditions(tt.pathTags, tt.conditions)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for tags %v with conditions %v", 
					tt.expected, result, tt.pathTags, tt.conditions)
			}
		})
	}
}

func TestTag_ShouldIncludePath(t *testing.T) {
	tests := []struct {
		name               string
		pathTags           []string
		isExplicitlyTagged bool
		filter             TagFilter
		expected           bool
	}{
		{
			name:               "untagged path (not explicit) always included",
			pathTags:           []string{},
			isExplicitlyTagged: false,
			filter:             TagFilter{Include: [][]string{{"home"}}, Exclude: [][]string{{"gaming"}}},
			expected:           true,
		},
		{
			name:               "explicitly empty tags should apply filtering",
			pathTags:           []string{},
			isExplicitlyTagged: true,
			filter:             TagFilter{Include: [][]string{{"home"}}, Exclude: [][]string{{"gaming"}}},
			expected:           false,
		},
		{
			name:               "no filters - include by default",
			pathTags:           []string{"anything"},
			isExplicitlyTagged: true,
			filter:             TagFilter{},
			expected:           true,
		},
		{
			name:               "include filter matches",
			pathTags:           []string{"home", "dev"},
			isExplicitlyTagged: true,
			filter:             TagFilter{Include: [][]string{{"home"}}},
			expected:           true,
		},
		{
			name:               "include filter no match",
			pathTags:           []string{"gaming"},
			isExplicitlyTagged: true,
			filter:             TagFilter{Include: [][]string{{"home"}}},
			expected:           false,
		},
		{
			name:               "exclude filter matches - exclude wins",
			pathTags:           []string{"gaming"},
			isExplicitlyTagged: true,
			filter:             TagFilter{Exclude: [][]string{{"gaming"}}},
			expected:           false,
		},
		{
			name:               "exclude filter no match",
			pathTags:           []string{"home"},
			isExplicitlyTagged: true,
			filter:             TagFilter{Exclude: [][]string{{"gaming"}}},
			expected:           true,
		},
		{
			name:               "both filters - include matches, exclude doesn't",
			pathTags:           []string{"home", "dev"},
			isExplicitlyTagged: true,
			filter:             TagFilter{Include: [][]string{{"home"}}, Exclude: [][]string{{"gaming"}}},
			expected:           true,
		},
		{
			name:               "both filters - exclude wins",
			pathTags:           []string{"home", "gaming"},
			isExplicitlyTagged: true,
			filter:             TagFilter{Include: [][]string{{"home"}}, Exclude: [][]string{{"gaming"}}},
			expected:           false,
		},
		{
			name:               "complex AND logic in include",
			pathTags:           []string{"work", "server", "prod"},
			isExplicitlyTagged: true,
			filter:             TagFilter{Include: [][]string{{"work", "server"}}},
			expected:           true,
		},
		{
			name:               "complex OR logic in exclude",
			pathTags:           []string{"temp"},
			isExplicitlyTagged: true,
			filter:             TagFilter{Include: [][]string{{"home"}}, Exclude: [][]string{{"gaming"}, {"temp"}}},
			expected:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldIncludePath(tt.pathTags, tt.isExplicitlyTagged, tt.filter)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for tags %v with filter %+v", 
					tt.expected, result, tt.pathTags, tt.filter)
			}
		})
	}
}

// Helper functions

func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

func equalStringSlices2D(a, b [][]string) bool {
	if len(a) != len(b) {
		return false
	}
	
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}
	return true
}

func TestTag_FormatTagsForDisplay(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		expected string
	}{
		// Single tag cases
		{"single tag", []string{"mac"}, "mac"},
		{"single tag uppercase", []string{"GAMING"}, "GAMING"},
		
		// Two tag cases
		{"two tags", []string{"mac", "gaming"}, "mac,gaming"},
		{"two tags mixed case", []string{"Work", "HOME"}, "Work,HOME"},
		
		// Three or more tag cases
		{"three tags", []string{"mac", "gaming", "video"}, "mac,gaming (+1)"},
		{"four tags", []string{"work", "dev", "server", "prod"}, "work,dev (+2)"},
		{"five tags", []string{"home", "personal", "media", "games", "temp"}, "home,personal (+3)"},
		
		// Edge cases
		{"no tags", []string{}, ""},
		{"empty tag in list", []string{"mac", "", "gaming"}, "mac, (+1)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTagsForDisplay(tt.tags)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s' for tags %v", tt.expected, result, tt.tags)
			}
		})
	}
}

func TestTag_GetIncludeFailureReasonImproved(t *testing.T) {
	tests := []struct {
		name               string
		pathTags           []string
		includeConditions  [][]string
		expectedReason     string
	}{
		// Single tag cases
		{
			name:              "single tag fails include",
			pathTags:          []string{"mac"},
			includeConditions: [][]string{{"essential"}},
			expectedReason:    "mac != essential",
		},
		// Two tag cases
		{
			name:              "two tags fail include",
			pathTags:          []string{"mac", "gaming"},
			includeConditions: [][]string{{"essential"}},
			expectedReason:    "mac,gaming != essential",
		},
		// Three or more tag cases
		{
			name:              "three tags fail include",
			pathTags:          []string{"mac", "gaming", "video"},
			includeConditions: [][]string{{"essential"}},
			expectedReason:    "mac,gaming (+1) != essential",
		},
		{
			name:              "five tags fail include",
			pathTags:          []string{"work", "dev", "server", "prod", "temp"},
			includeConditions: [][]string{{"essential"}},
			expectedReason:    "work,dev (+3) != essential",
		},
		// Complex include conditions
		{
			name:              "multiple include conditions",
			pathTags:          []string{"mac", "gaming", "video"},
			includeConditions: [][]string{{"essential"}, {"work", "dev"}},
			expectedReason:    "mac,gaming (+1) != essential,work+dev",
		},
		// Matching cases (should return empty)
		{
			name:              "tags match include condition",
			pathTags:          []string{"essential", "gaming"},
			includeConditions: [][]string{{"essential"}},
			expectedReason:    "",
		},
		// No tags case
		{
			name:              "no tags fail include",
			pathTags:          []string{},
			includeConditions: [][]string{{"essential"}},
			expectedReason:    "no tags != essential",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIncludeFailureReason(tt.pathTags, tt.includeConditions)
			if result != tt.expectedReason {
				t.Errorf("Expected '%s', got '%s' for tags %v with conditions %v", 
					tt.expectedReason, result, tt.pathTags, tt.includeConditions)
			}
		})
	}
}

