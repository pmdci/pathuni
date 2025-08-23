package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Tag validation regex: ^[a-zA-Z][a-zA-Z0-9_]{2,19}$
// - Must start with letter
// - Can contain letters, numbers, underscores
// - 3-20 characters total
var tagRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{2,19}$`)

// validateTag checks if a tag meets the validation criteria
func validateTag(tag string) error {
	if !tagRegex.MatchString(tag) {
		return fmt.Errorf("invalid tag '%s': tags must be 3-20 characters, start with a letter, and contain only letters, numbers, and underscores", tag)
	}
	return nil
}

// validateTags validates a slice of tags and returns detailed error information
func validateTags(tags []string, context string) error {
	if len(tags) == 0 {
		// Empty tags array is allowed (treated as untagged)
		return nil
	}
	
	// Check for duplicate tags
	seen := make(map[string]bool)
	for _, tag := range tags {
		tagLower := strings.ToLower(tag)
		if seen[tagLower] {
			return fmt.Errorf("duplicate tag '%s' in %s", tag, context)
		}
		seen[tagLower] = true
		
		// Validate individual tag
		if err := validateTag(tag); err != nil {
			return fmt.Errorf("%s in %s", err.Error(), context)
		}
	}
	
	return nil
}

// TagFilter represents parsed include/exclude tag logic
type TagFilter struct {
	Include [][]string // OR groups of AND conditions: [["home", "dev"], ["work"]] = (home AND dev) OR work
	Exclude [][]string // OR groups of AND conditions: [["gaming"], ["work", "server"]] = gaming OR (work AND server)
}

// parseTagFilter parses a tag filter string like "home,dev" or "work+server"
// Returns [][]string where:
// - "home,dev" becomes [["home"], ["dev"]] (OR logic)
// - "home+dev" becomes [["home", "dev"]] (AND logic)  
// - "home,work+server" becomes [["home"], ["work", "server"]] (home OR (work AND server))
func parseTagFilter(filter string) ([][]string, error) {
	if filter == "" {
		return nil, nil
	}
	
	var result [][]string
	
	// Split by comma for OR groups
	orGroups := strings.Split(filter, ",")
	for _, group := range orGroups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		
		// Split by plus for AND conditions within the group
		andTags := strings.Split(group, "+")
		var andGroup []string
		
		for _, tag := range andTags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				return nil, fmt.Errorf("empty tag in filter '%s'", filter)
			}
			
			// Validate the tag format
			if err := validateTag(tag); err != nil {
				return nil, fmt.Errorf("invalid tag in filter '%s': %v", filter, err)
			}
			
			andGroup = append(andGroup, tag)
		}
		
		if len(andGroup) > 0 {
			result = append(result, andGroup)
		}
	}
	
	return result, nil
}

// matchesTagConditions checks if a set of path tags matches the filter conditions
func matchesTagConditions(pathTags []string, conditions [][]string) bool {
	if len(conditions) == 0 {
		return false // No conditions means no match
	}
	
	// Convert path tags to lowercase for case-insensitive matching
	pathTagsLower := make(map[string]bool)
	for _, tag := range pathTags {
		pathTagsLower[strings.ToLower(tag)] = true
	}
	
	// OR logic between condition groups
	for _, andGroup := range conditions {
		// AND logic within each group
		allMatch := true
		for _, requiredTag := range andGroup {
			if !pathTagsLower[strings.ToLower(requiredTag)] {
				allMatch = false
				break
			}
		}
		
		if allMatch {
			return true // Found a matching OR group
		}
	}
	
	return false
}

// parseTagFlags parses CLI tag include/exclude flags into a TagFilter struct
func parseTagFlags(includeFlag, excludeFlag string) (TagFilter, error) {
	var filter TagFilter
	var err error
	
	// Parse include filter
	if includeFlag != "" {
		filter.Include, err = parseTagFilter(includeFlag)
		if err != nil {
			return TagFilter{}, fmt.Errorf("invalid --tags-include flag: %v", err)
		}
	}
	
	// Parse exclude filter
	if excludeFlag != "" {
		filter.Exclude, err = parseTagFilter(excludeFlag)
		if err != nil {
			return TagFilter{}, fmt.Errorf("invalid --tags-exclude flag: %v", err)
		}
	}
	
	return filter, nil
}

// shouldIncludePath determines if a path should be included based on tag filters
// Returns true if the path should be included, false otherwise
// Logic:
// - If pathTags is empty and isExplicitlyTagged is false: always include (untagged paths are immune to filtering)
// - If pathTags is empty and isExplicitlyTagged is true: apply normal filtering (explicitly no tags)
// - If include filter exists and path doesn't match: exclude
// - If exclude filter exists and path matches: exclude (exclude wins)
// - Otherwise: include
func shouldIncludePath(pathTags []string, isExplicitlyTagged bool, filter TagFilter) bool {
	// Truly untagged paths (not explicitly tagged) are always included (immune to filtering)
	if len(pathTags) == 0 && !isExplicitlyTagged {
		return true
	}
	
	// Check exclude conditions first (exclude wins)
	if len(filter.Exclude) > 0 && matchesTagConditions(pathTags, filter.Exclude) {
		return false
	}
	
	// Check include conditions
	if len(filter.Include) > 0 {
		return matchesTagConditions(pathTags, filter.Include)
	}
	
	// No filters applied, include by default
	return true
}

// getPathSkipReasons returns the reasons why a path should be skipped, or nil if included
func getPathSkipReasons(pathTags []string, isExplicitlyTagged bool, filter TagFilter) []SkipReason {
	// Truly untagged paths (not explicitly tagged) are always included
	if len(pathTags) == 0 && !isExplicitlyTagged {
		return nil
	}
	
	// Check exclude conditions first (exclude wins)
	if len(filter.Exclude) > 0 {
		if excludeReason := getExcludeReason(pathTags, filter.Exclude); excludeReason != "" {
			return []SkipReason{{Type: "tags", Detail: excludeReason}}
		}
	}
	
	// Check include conditions
	if len(filter.Include) > 0 {
		if includeReason := getIncludeFailureReason(pathTags, filter.Include); includeReason != "" {
			return []SkipReason{{Type: "tags", Detail: includeReason}}
		}
	}
	
	// No filters applied or path matches include, include by default
	return nil
}

// getExcludeReason returns a reason string if path matches exclude conditions, empty otherwise
func getExcludeReason(pathTags []string, excludeConditions [][]string) string {
	pathTagsLower := make(map[string]bool)
	for _, tag := range pathTags {
		pathTagsLower[strings.ToLower(tag)] = true
	}
	
	// Find the first exclude condition that matches
	for _, andGroup := range excludeConditions {
		allMatch := true
		for _, requiredTag := range andGroup {
			if !pathTagsLower[strings.ToLower(requiredTag)] {
				allMatch = false
				break
			}
		}
		
		if allMatch {
			// Find which tag from pathTags matches this condition
			for _, pathTag := range pathTags {
				for _, excludeTag := range andGroup {
					if strings.EqualFold(pathTag, excludeTag) {
						return fmt.Sprintf("%s = %s", pathTag, excludeTag)
					}
				}
			}
		}
	}
	
	return ""
}

// getIncludeFailureReason returns a reason string if path fails include conditions, empty otherwise
func getIncludeFailureReason(pathTags []string, includeConditions [][]string) string {
	if matchesTagConditions(pathTags, includeConditions) {
		return "" // Path matches include conditions
	}
	
	// Path doesn't match include conditions, generate reason
	// Format the include conditions as a readable string
	conditionStr := formatTagConditions(includeConditions)
	
	// Find the best representative tag from the path to show in the comparison
	if len(pathTags) > 0 {
		return fmt.Sprintf("%s != %s", pathTags[0], conditionStr)
	}
	
	return fmt.Sprintf("no tags != %s", conditionStr)
}

// formatTagConditions converts tag conditions to readable string like "dev+work,audio"
func formatTagConditions(conditions [][]string) string {
	var parts []string
	for _, andGroup := range conditions {
		if len(andGroup) == 1 {
			parts = append(parts, andGroup[0])
		} else {
			parts = append(parts, strings.Join(andGroup, "+"))
		}
	}
	return strings.Join(parts, ",")
}