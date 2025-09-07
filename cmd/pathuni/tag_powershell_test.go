package main

import (
    "io"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func captureOut(f func()) string {
    old := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w
    f()
    w.Close()
    os.Stdout = old
    b, _ := io.ReadAll(r)
    r.Close()
    return string(b)
}

// Stage 2: explicit tags for PowerShell system paths when as=pathuni
func TestDryRun_PowerShell_AsPathuni_WithTags(t *testing.T) {
    setupTestFilesystem(t)
    defer cleanupTestFilesystem()

    os.Setenv("PATHUNI_TEST_SYSTEM_PATHS_ROOT", filepath.Join("testdata", "system_paths"))
    t.Setenv("PATH", "")

    cfg := filepath.Join("/tmp", "pathuni", "home", "Pratt", ".config", "pathuni", "psys-tags.yaml")
    if err := os.MkdirAll(filepath.Dir(cfg), 0755); err != nil { t.Fatalf("mkdir: %v", err) }
    cfgContent := "macos:\n  powershell:\n    include_system_paths: true\n    include_system_paths_as: pathuni\n    tags: [sys]\n"
    if err := os.WriteFile(cfg, []byte(cfgContent), 0644); err != nil { t.Fatalf("write cfg: %v", err) }

    // Save/restore globals
    oldC, oldOS, oldShell := config, osOverride, shell
    oldScope, oldPrune := scope, prune
    oldInc, oldExc := tagsInclude, tagsExclude
    defer func(){ config, osOverride, shell = oldC, oldOS, oldShell; scope, prune = oldScope, oldPrune; tagsInclude, tagsExclude = oldInc, oldExc }()

    config = cfg
    osOverride = "macOS"
    shell = "powershell"
    scope = "full"
    prune = "pathuni"

    // Include by tag
    tagsInclude = "sys"
    tagsExclude = ""
    out := captureOut(func(){ _ = PrintDryRunReport(cfg, "macOS", "powershell", false, false, scope) })
    if !strings.Contains(out, "[+] /tmp/pathuni/usr/local/bin") {
        t.Fatalf("expected included [+] mocked path with tag 'sys', got:\n%s", out)
    }

    // Exclude by tag
    tagsInclude = ""
    tagsExclude = "sys"
    out = captureOut(func(){ _ = PrintDryRunReport(cfg, "macOS", "powershell", false, false, scope) })
    if strings.Contains(out, "[+] /tmp/pathuni/usr/local/bin") {
        t.Fatalf("did not expect mocked path included when excluded by tag 'sys':\n%s", out)
    }
    if !strings.Contains(out, "[-] /tmp/pathuni/usr/local/bin") {
        t.Fatalf("expected mocked path to appear as filtered [-] when excluded by tag 'sys':\n%s", out)
    }
}

// Stage 3: explicit empty tags break inheritance
func TestDryRun_PowerShell_AsPathuni_EmptyTagsBreakInheritance(t *testing.T) {
    setupTestFilesystem(t)
    defer cleanupTestFilesystem()

    os.Setenv("PATHUNI_TEST_SYSTEM_PATHS_ROOT", filepath.Join("testdata", "system_paths"))
    t.Setenv("PATH", "")

    cfg := filepath.Join("/tmp", "pathuni", "home", "Pratt", ".config", "pathuni", "psys-empty-tags.yaml")
    if err := os.MkdirAll(filepath.Dir(cfg), 0755); err != nil { t.Fatalf("mkdir: %v", err) }
    cfgContent := "macos:\n  tags: [mac]\n  powershell:\n    include_system_paths: true\n    include_system_paths_as: pathuni\n    tags: []\n"
    if err := os.WriteFile(cfg, []byte(cfgContent), 0644); err != nil { t.Fatalf("write cfg: %v", err) }

    oldC, oldOS, oldShell := config, osOverride, shell
    oldScope, oldPrune := scope, prune
    oldInc, oldExc := tagsInclude, tagsExclude
    defer func(){ config, osOverride, shell = oldC, oldOS, oldShell; scope, prune = oldScope, oldPrune; tagsInclude, tagsExclude = oldInc, oldExc }()

    config = cfg
    osOverride = "macOS"
    shell = "powershell"
    scope = "full"
    prune = "pathuni"

    tagsInclude = "mac"
    tagsExclude = ""
    out := captureOut(func(){ _ = PrintDryRunReport(cfg, "macOS", "powershell", false, false, scope) })
    // Since tags: [] breaks inheritance, include filter 'mac' should NOT include them
    if strings.Contains(out, "[+] /tmp/pathuni/usr/local/bin\n") {
        t.Fatalf("did not expect mocked path included when tags: [] break inheritance: \n%s", out)
    }
    if !strings.Contains(out, "[-]") && !strings.Contains(out, "[!]") {
        t.Fatalf("expected mocked path to appear in Skipped when include=mac and tags: []: \n%s", out)
    }
}

// Stage 1: as=pathuni with no tags -> inherit platform tags
func TestDryRun_PowerShell_AsPathuni_InheritPlatformTags(t *testing.T) {
    setupTestFilesystem(t)
    defer cleanupTestFilesystem()

    os.Setenv("PATHUNI_TEST_SYSTEM_PATHS_ROOT", filepath.Join("testdata", "system_paths"))
    t.Setenv("PATH", "")

    cfg := filepath.Join("/tmp", "pathuni", "home", "Pratt", ".config", "pathuni", "psys-inherit.yaml")
    if err := os.MkdirAll(filepath.Dir(cfg), 0755); err != nil { t.Fatalf("mkdir: %v", err) }
    // powershell.tag omitted -> Tags=nil -> inherit platform [mac]
    cfgContent := "macos:\n  tags: [mac]\n  powershell:\n    include_system_paths: true\n    include_system_paths_as: pathuni\n"
    if err := os.WriteFile(cfg, []byte(cfgContent), 0644); err != nil { t.Fatalf("write cfg: %v", err) }

    oldC, oldOS, oldShell := config, osOverride, shell
    oldScope, oldPrune := scope, prune
    oldInc, oldExc := tagsInclude, tagsExclude
    defer func(){ config, osOverride, shell = oldC, oldOS, oldShell; scope, prune = oldScope, oldPrune; tagsInclude, tagsExclude = oldInc, oldExc }()

    config = cfg
    osOverride = "macOS"
    shell = "powershell"
    scope = "full"
    prune = "pathuni"

    // Should include by platform tag
    tagsInclude = "mac"
    tagsExclude = ""
    out := captureOut(func(){ _ = PrintDryRunReport(cfg, "macOS", "powershell", false, false, scope) })
    if !strings.Contains(out, "[+] /tmp/pathuni/usr/local/bin") {
        t.Fatalf("expected mocked path included by inherited platform tag 'mac', got:\n%s", out)
    }

    // Excluding 'mac' should filter it out
    tagsInclude = ""
    tagsExclude = "mac"
    out = captureOut(func(){ _ = PrintDryRunReport(cfg, "macOS", "powershell", false, false, scope) })
    if strings.Contains(out, "[+] /tmp/pathuni/usr/local/bin") {
        t.Fatalf("did not expect mocked path included when excluded by inherited tag 'mac':\n%s", out)
    }
}
