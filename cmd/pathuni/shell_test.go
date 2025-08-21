package main

import (
	"testing"
)

func TestShell_Validation(t *testing.T) {
	tests := []struct {
		name     string
		shell    string
		expected bool
	}{
		{"bash valid", "bash", true},
		{"zsh valid", "zsh", true},
		{"sh valid", "sh", true},
		{"fish valid", "fish", true},
		{"powershell valid", "powershell", true},
		{"invalid shell", "cmd", false},
		{"empty shell", "", false},
		{"case sensitive", "BASH", false},
		{"partial match", "bas", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := shellIsValid(tt.shell)
			if actual != tt.expected {
				t.Errorf("shellIsValid(%q) = %v, want %v", tt.shell, actual, tt.expected)
			}
		})
	}
}

func TestShell_Names(t *testing.T) {
	names := shellNames()
	
	// Check that we get expected shells
	expectedShells := []string{"bash", "fish", "powershell", "sh", "zsh"}
	if len(names) != len(expectedShells) {
		t.Errorf("Expected %d shells, got %d", len(expectedShells), len(names))
	}
	
	// Check that the list is sorted
	for i := 1; i < len(names); i++ {
		if names[i-1] >= names[i] {
			t.Errorf("Shell names not sorted: %v", names)
			break
		}
	}
	
	// Check that all expected shells are present
	shellSet := make(map[string]bool)
	for _, shell := range names {
		shellSet[shell] = true
	}
	
	for _, expected := range expectedShells {
		if !shellSet[expected] {
			t.Errorf("Expected shell %q not found in list: %v", expected, names)
		}
	}
}

func TestShell_BashRendering(t *testing.T) {
	tests := []struct {
		name     string
		paths    []string
		expected string
	}{
		{
			name:     "single path",
			paths:    []string{"/usr/bin"},
			expected: `export PATH="/usr/bin"`,
		},
		{
			name:     "multiple paths",
			paths:    []string{"/usr/bin", "/usr/local/bin"},
			expected: `export PATH="/usr/bin:/usr/local/bin"`,
		},
		{
			name:     "empty paths",
			paths:    []string{},
			expected: `export PATH=""`,
		},
		{
			name:     "paths with spaces",
			paths:    []string{"/path with spaces", "/usr/bin"},
			expected: `export PATH="/path with spaces:/usr/bin"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := renderBash(tt.paths)
			if actual != tt.expected {
				t.Errorf("renderBash(%v) = %q, want %q", tt.paths, actual, tt.expected)
			}
		})
	}
}

func TestShell_FishRendering(t *testing.T) {
	tests := []struct {
		name     string
		paths    []string
		expected string
	}{
		{
			name:     "single path",
			paths:    []string{"/usr/bin"},
			expected: `set -gx PATH /usr/bin`,
		},
		{
			name:     "multiple paths",
			paths:    []string{"/usr/bin", "/usr/local/bin"},
			expected: `set -gx PATH /usr/bin /usr/local/bin`,
		},
		{
			name:     "empty paths",
			paths:    []string{},
			expected: `set -gx PATH `,
		},
		{
			name:     "paths with spaces",
			paths:    []string{"/path with spaces", "/usr/bin"},
			expected: `set -gx PATH /path with spaces /usr/bin`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := renderFish(tt.paths)
			if actual != tt.expected {
				t.Errorf("renderFish(%v) = %q, want %q", tt.paths, actual, tt.expected)
			}
		})
	}
}

func TestShell_PowershellRendering(t *testing.T) {
	tests := []struct {
		name     string
		paths    []string
		expected string
	}{
		{
			name:     "single path",
			paths:    []string{"/usr/bin"},
			expected: `$env:PATH = "/usr/bin"`,
		},
		{
			name:     "multiple paths",
			paths:    []string{"/usr/bin", "/usr/local/bin"},
			expected: `$env:PATH = "/usr/bin:/usr/local/bin"`,
		},
		{
			name:     "empty paths",
			paths:    []string{},
			expected: `$env:PATH = ""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := renderPwsh(tt.paths)
			if actual != tt.expected {
				t.Errorf("renderPwsh(%v) = %q, want %q", tt.paths, actual, tt.expected)
			}
		})
	}
}

func TestShell_Normalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"pwsh to powershell", "pwsh", "powershell"},
		{"bash unchanged", "bash", "bash"},
		{"zsh unchanged", "zsh", "zsh"},
		{"fish unchanged", "fish", "fish"},
		{"powershell unchanged", "powershell", "powershell"},
		{"unknown unchanged", "unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := normalizeShellName(tt.input)
			if actual != tt.expected {
				t.Errorf("normalizeShellName(%q) = %q, want %q", tt.input, actual, tt.expected)
			}
		})
	}
}

func TestShell_RendererMapping(t *testing.T) {
	// Verify all supported shells have renderers
	for shell := range supportedShells {
		t.Run("renderer_exists_for_"+shell, func(t *testing.T) {
			renderer, exists := renderers[shell]
			if !exists {
				t.Errorf("No renderer found for supported shell: %s", shell)
			}
			
			// Test that renderer doesn't panic with empty input
			result := renderer([]string{})
			if result == "" {
				t.Errorf("Renderer for %s returned empty string", shell)
			}
		})
	}
	
	// Test that bash, zsh, sh all use the same renderer
	bashRenderer := renderers["bash"]
	zshRenderer := renderers["zsh"]
	shRenderer := renderers["sh"]
	
	testPaths := []string{"/usr/bin", "/usr/local/bin"}
	
	bashResult := bashRenderer(testPaths)
	zshResult := zshRenderer(testPaths)
	shResult := shRenderer(testPaths)
	
	if bashResult != zshResult || bashResult != shResult {
		t.Errorf("bash, zsh, and sh should produce identical output. Got bash: %q, zsh: %q, sh: %q", bashResult, zshResult, shResult)
	}
}