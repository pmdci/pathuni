package main

import (
    "io"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func captureDry(f func()) string {
    old := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w
    f()
    w.Close()
    os.Stdout = old
    out, _ := io.ReadAll(r)
    r.Close()
    return string(out)
}

// Test classification of macOS system paths for PowerShell in dry-run using
// the mocked system_paths testdata via the PATHUNI_TEST_SYSTEM_PATHS_ROOT seam.
func TestDryRun_PowerShell_SystemPaths_Classification(t *testing.T) {
    setupTestFilesystem(t)
    defer cleanupTestFilesystem()

    // Use testdata/system_paths as the provider for getSystemPaths
    t.Setenv("PATHUNI_TEST_SYSTEM_PATHS_ROOT", filepath.Join("testdata", "system_paths"))

    // Minimal config; platform macOS, powershell include system paths
    cfgPath := filepath.Join("/tmp", "pathuni", "home", "Pratt", ".config", "pathuni", "psys-classify.yaml")
    if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
        t.Fatalf("mkdir cfg: %v", err)
    }

    // a) as=system (default) → expect [.] markers for mock system paths in full
    contentSystem := "macos:\n  powershell:\n    include_system_paths: true\n"
    if err := os.WriteFile(cfgPath, []byte(contentSystem), 0644); err != nil {
        t.Fatalf("write cfg: %v", err)
    }
    // Set globals for this run
    config = cfgPath
    osOverride = "macOS"
    shell = "powershell"
    prune = "none"
    scope = "full"
    // Clear PATH to avoid extra environment entries
    t.Setenv("PATH", "")
    out := captureDry(func() { _ = PrintDryRunReport(cfgPath, "macOS", "powershell", false, false, scope) })
    // Assert presence of a known mocked system path with [.] marker
    if !strings.Contains(out, "[.] /tmp/pathuni/usr/local/bin") {
        t.Fatalf("expected system marker for mocked path, got:\n%s", out)
    }

    // b) as=pathuni → expect [+] markers for the same mock system paths
    contentPathuni := "macos:\n  powershell:\n    include_system_paths: true\n    include_system_paths_as: pathuni\n"
    if err := os.WriteFile(cfgPath, []byte(contentPathuni), 0644); err != nil {
        t.Fatalf("write cfg2: %v", err)
    }
    out = captureDry(func() { _ = PrintDryRunReport(cfgPath, "macOS", "powershell", false, false, scope) })
    if !strings.Contains(out, "[+] /tmp/pathuni/usr/local/bin") {
        t.Fatalf("expected pathuni marker for mocked path, got:\n%s", out)
    }
}

// Verify that in system scope, prune=system shows not-found entries with [?]
func TestDryRun_PowerShell_SystemScope_PruneSystem_NotFound(t *testing.T) {
    setupTestFilesystem(t)
    defer cleanupTestFilesystem()

    t.Setenv("PATHUNI_TEST_SYSTEM_PATHS_ROOT", filepath.Join("testdata", "system_paths"))

    cfgPath := filepath.Join("/tmp", "pathuni", "home", "Pratt", ".config", "pathuni", "psys-system-prune.yaml")
    if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
        t.Fatalf("mkdir cfg: %v", err)
    }
    content := "macos:\n  powershell:\n    include_system_paths: true\n"
    if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
        t.Fatalf("write cfg: %v", err)
    }

    // Configure globals
    oldC, oldOS, oldShell := config, osOverride, shell
    oldScope, oldPrune := scope, prune
    defer func(){ config, osOverride, shell = oldC, oldOS, oldShell; scope, prune = oldScope, oldPrune }()
    config = cfgPath
    osOverride = "macOS"
    shell = "powershell"
    scope = "system"
    prune = "system"

    // Include a nonexistent path in PATH so prune=system will produce [?]
    t.Setenv("PATH", "/not-found-system")

    out := captureDry(func() { _ = PrintDryRunReport(cfgPath, "macOS", "powershell", false, false, scope) })
    if !strings.Contains(out, "[?] /not-found-system (not found)") {
        t.Fatalf("expected system not-found marker for /not-found-system, got:\n%s", out)
    }
}
