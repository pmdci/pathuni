package main

import (
	"os"
	"testing"
)

func TestGetOSName(t *testing.T) {
	// Store original value and restore at end
	originalOsOverride := osOverride
	defer func() {
		osOverride = originalOsOverride
	}()
	
	tests := []struct {
		name           string
		osOverride     string
		expectedOS     string
		expectedInferred bool
	}{
		{
			name:           "no override - detects current OS",
			osOverride:     "",
			expectedOS:     getExpectedDetectedOS(),
			expectedInferred: true,
		},
		{
			name:           "macOS override",
			osOverride:     "macOS",
			expectedOS:     "macOS",
			expectedInferred: false,
		},
		{
			name:           "darwin synonym",
			osOverride:     "darwin",
			expectedOS:     "macOS",
			expectedInferred: false,
		},
		{
			name:           "case insensitive - macos",
			osOverride:     "macos",
			expectedOS:     "macOS",
			expectedInferred: false,
		},
		{
			name:           "case insensitive - MACOS",
			osOverride:     "MACOS",
			expectedOS:     "macOS",
			expectedInferred: false,
		},
		{
			name:           "linux override",
			osOverride:     "linux",
			expectedOS:     "Linux",
			expectedInferred: false,
		},
		{
			name:           "case insensitive - LINUX",
			osOverride:     "LINUX",
			expectedOS:     "Linux",
			expectedInferred: false,
		},
		{// Use an OS name that DOESN'T exist in real life
			name:           "invalid OS returns empty",
			osOverride:     "yyz",
			expectedOS:     "",
			expectedInferred: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			osOverride = tt.osOverride
			osName, inferred := getOSName()
			
			if osName != tt.expectedOS {
				t.Errorf("getOSName() osName = %v, want %v", osName, tt.expectedOS)
			}
			if inferred != tt.expectedInferred {
				t.Errorf("getOSName() inferred = %v, want %v", inferred, tt.expectedInferred)
			}
		})
	}
}

func TestOsIsValid(t *testing.T) {
	tests := []struct {
		osName   string
		expected bool
	}{
		{"macOS", true},
		{"Linux", true},
		{"", false},
		{"yyz", false},
		{"darwin", false}, // darwin is mapped to macOS, but validation checks normalised names
		{"invalid", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.osName, func(t *testing.T) {
			actual := osIsValid(tt.osName)
			if actual != tt.expected {
				t.Errorf("osIsValid(%q) = %v, want %v", tt.osName, actual, tt.expected)
			}
		})
	}
}

func TestOsNames(t *testing.T) {
	names := osNames()
	expected := []string{"macOS", "Linux"}
	
	if len(names) != len(expected) {
		t.Errorf("osNames() returned %d names, want %d", len(names), len(expected))
	}
	
	for _, expectedName := range expected {
		found := false
		for _, name := range names {
			if name == expectedName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("osNames() missing expected OS %q", expectedName)
		}
	}
}

// Helper function to get the expected OS for the current runtime
func getExpectedDetectedOS() string {
	switch os.Getenv("GOOS") {
	case "darwin":
		return "macOS"
	case "linux":
		return "Linux"
	default:
		// Fallback to actual runtime detection
		originalOsOverride := osOverride
		osOverride = ""
		osName, _ := getOSName()
		osOverride = originalOsOverride
		return osName
	}
}