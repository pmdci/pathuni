package main

import (
	"os"
	"testing"
)

// setupTestFilesystem creates the test directory structure under /tmp/pathuni/
// Only creates directories that should exist for testing
func setupTestFilesystem(t *testing.T) {
	t.Helper()
	
	// Remove any existing test structure
	os.RemoveAll("/tmp/pathuni")
	
	// Create directories that should exist for testing
	testDirs := []string{
		"/tmp/pathuni/usr/local/bin",
		"/tmp/pathuni/usr/bin", 
		"/tmp/pathuni/usr/sbin",
		"/tmp/pathuni/usr/games",
		"/tmp/pathuni/bin",
		"/tmp/pathuni/sbin", 
		"/tmp/pathuni/tmp",
		"/tmp/pathuni/snap/bin",
		"/tmp/pathuni/opt/games/bin",
		"/tmp/pathuni/opt/homebrew/bin",
		"/tmp/pathuni/opt/homebrew/sbin",
		"/tmp/pathuni/home/Pratt/.local/bin",
		"/tmp/pathuni/home/Pratt/.cargo/bin",
		"/tmp/pathuni/home/Pratt/.npm-global/bin",
		"/tmp/pathuni/home/Pratt/bin",
		"/tmp/pathuni/home/Pratt/.node_modules/.bin",
		"/tmp/pathuni/home/Pratt/.config/pathuni",
		"/tmp/pathuni/home/linuxbrew/.linuxbrew/bin",
		"/tmp/pathuni/home/linuxbrew/.linuxbrew/sbin",
		"/tmp/pathuni/Applications/Docker.app/Contents/Resources/bin",
		"/tmp/pathuni/Applications/Xcode.app/Contents/Developer/usr/bin",
		"/tmp/pathuni/System/Library/Frameworks",
		"/tmp/pathuni/usr/local/go/bin",
		"/tmp/pathuni/usr/local/node/bin",
		"/tmp/pathuni/opt/dev/bin",
		"/tmp/pathuni/opt/server/bin",
		"/tmp/pathuni/opt/work/bin",
	}
	
	for _, dir := range testDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create test directory %s: %v", dir, err)
		}
	}
	
	// Uncomment to inspect filesystem structure:
	//time.Sleep(100 * time.Second)

}

// cleanupTestFilesystem removes the test directory structure
func cleanupTestFilesystem() {
	os.RemoveAll("/tmp/pathuni")
}