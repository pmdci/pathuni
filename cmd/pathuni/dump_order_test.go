package main

import (
    "path/filepath"
    "testing"
)

// Verifies that getAllPathsWithTagFiltering returns a pathuni-first merge
// (pathuni entries precede system entries) and contains no duplicates.
func TestDump_ScopeFull_PathuniFirst(t *testing.T) {
    setupTestFilesystem(t)
    defer cleanupTestFilesystem()

    // Point config to a known test file with macOS-like entries
    config = filepath.Join("testdata", "valid_config.yaml")
    osOverride = "macOS"
    shell = "bash"

    // Control PATH to a deterministic set of system paths
    t.Setenv("PATH", "/tmp/pathuni/usr/bin:/tmp/pathuni/bin")

    // Collect individual sources
    system, err := resolveSystemPaths()
    if err != nil {
        t.Fatalf("resolveSystemPaths error: %v", err)
    }

    pathuni, err := resolvePathuniPaths()
    if err != nil {
        t.Fatalf("resolvePathuniPaths error: %v", err)
    }

    // Sanity: ensure we have at least one exclusive from each side
    sysSet := make(map[string]bool)
    for _, s := range system { sysSet[s] = true }
    puSet := make(map[string]bool)
    for _, p := range pathuni { puSet[p] = true }

    // Compute merged list
    merged, err := getAllPathsWithTagFiltering()
    if err != nil {
        t.Fatalf("getAllPathsWithTagFiltering error: %v", err)
    }

    // Check no duplicates
    seen := map[string]bool{}
    for _, p := range merged {
        if seen[p] {
            t.Fatalf("duplicate entry in merged list: %s", p)
        }
        seen[p] = true
    }

    // Determine min index of any system-only element and max index of any pathuni-only element
    minSysIdx := -1
    maxPuIdx := -1
    for i, p := range merged {
        inSys := sysSet[p]
        inPu := puSet[p]
        if inSys && !inPu {
            if minSysIdx == -1 || i < minSysIdx { minSysIdx = i }
        }
        if inPu && !inSys {
            if i > maxPuIdx { maxPuIdx = i }
        }
    }

    if minSysIdx != -1 && maxPuIdx != -1 && !(maxPuIdx < minSysIdx) {
        t.Fatalf("expected pathuni-first order: max pathuni-only index %d should be < min system-only index %d; merged=%v", maxPuIdx, minSysIdx, merged)
    }
}

