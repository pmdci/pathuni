package main

import (
    "io"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func captureDryRunOutput(f func()) string {
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

func TestDryRun_ScopeSystemAndFull(t *testing.T) {
    setupTestFilesystem(t)
    defer cleanupTestFilesystem()

    // Reset globals to safe defaults for this test
    prune = "pathuni"
    deferEnv = false

    config = filepath.Join("testdata", "valid_config.yaml")
    osOverride = "macOS"
    shell = "bash"
    t.Setenv("PATH", "/tmp/pathuni/usr/bin:/tmp/pathuni/bin")

    // system scope
    out := captureDryRunOutput(func() { _ = PrintDryRunReport(config, "macOS", "bash", false, false, "system") })
    if !strings.Contains(out, "Flags : scope=system") { t.Fatalf("missing flags header in system output: %s", out) }
    if !strings.Contains(out, "[.] /tmp/pathuni/usr/bin") || !strings.Contains(out, "[.] /tmp/pathuni/bin") {
        t.Fatalf("system entries not listed with [.] marker: %s", out)
    }
    if strings.Contains(out, "[+] ") { t.Fatalf("unexpected pathuni marker in system scope: %s", out) }

    // full scope
    prune = "pathuni" // default behavior shows only pathuni reasons
    out = captureDryRunOutput(func() { _ = PrintDryRunReport(config, "macOS", "bash", false, false, "full") })
    if !strings.Contains(out, "Flags : scope=full") { t.Fatalf("missing flags header in full output: %s", out) }
    if !strings.Contains(out, "[+] /tmp/pathuni/usr/local/bin") { t.Fatalf("expected pathuni entry not found with [+]: %s", out) }
    if !strings.Contains(out, "[.] /tmp/pathuni/bin") { t.Fatalf("system entry not found with [.]: %s", out) }
    if !strings.Contains(out, "Paths included in total") || !strings.Contains(out, "Pathuni path") || !strings.Contains(out, "System path") {
        t.Fatalf("missing included breakdown summary: %s", out)
    }
}

func TestDryRun_PruneVariants(t *testing.T) {
    setupTestFilesystem(t)
    defer cleanupTestFilesystem()

    // Prepare a temp config with one existing and one missing path
    tmpCfg, err := os.CreateTemp("/tmp/pathuni/home/Pratt/.config/pathuni", "cfg-*.yaml")
    if err != nil { t.Fatalf("temp cfg: %v", err) }
    defer os.Remove(tmpCfg.Name())
    cfgContent := "all:\n  paths:\n    - \"/tmp/pathuni/usr/local/bin\"\n    - \"/tmp/pathuni/does-not-exist\"\n"
    if _, err := tmpCfg.WriteString(cfgContent); err != nil { t.Fatalf("write cfg: %v", err) }
    tmpCfg.Close()

    config = tmpCfg.Name()
    osOverride = "macOS"
    shell = "bash"
    t.Setenv("PATH", "/tmp/pathuni/usr/bin:/does/not/exist:/tmp/pathuni/bin")

    // prune=none: missing pathuni should be included, and no [!] shown
    prune = "none"
    out := captureDryRunOutput(func() { _ = PrintDryRunReport(config, "macOS", "bash", false, false, "full") })
    if !strings.Contains(out, "/tmp/pathuni/does-not-exist") { t.Fatalf("expected missing pathuni included for prune=none: %s", out) }
    if strings.Contains(out, "[!] /tmp/pathuni/does-not-exist") { t.Fatalf("unexpected [!] for prune=none: %s", out) }

    // prune=system: missing pathuni still included; show [?] for system, not [!]
    prune = "system"
    out = captureDryRunOutput(func() { _ = PrintDryRunReport(config, "macOS", "bash", false, false, "full") })
    if !strings.Contains(out, "/tmp/pathuni/does-not-exist") { t.Fatalf("expected missing pathuni included for prune=system: %s", out) }
    if !strings.Contains(out, "[?] /does/not/exist (not found)") { t.Fatalf("expected system [?] for prune=system: %s", out) }
    if strings.Contains(out, "[!] /tmp/pathuni/does-not-exist") { t.Fatalf("unexpected [!] for prune=system: %s", out) }

    // prune=all: missing pathuni excluded and listed with [!]; summary should not show system line if none
    prune = "all"
    out = captureDryRunOutput(func() { _ = PrintDryRunReport(config, "macOS", "bash", false, false, "full") })
    if strings.Contains(out, "/tmp/pathuni/does-not-exist\n") { t.Fatalf("did not expect missing pathuni in Included for prune=all: %s", out) }
    if !strings.Contains(out, "[!] /tmp/pathuni/does-not-exist (not found)") { t.Fatalf("expected [!] for prune=all: %s", out) }
    if strings.Contains(out, "System path skipped in total") && !strings.Contains(out, "0 System paths skipped in total") {
        t.Fatalf("unexpected system skipped summary when none pruned: %s", out)
    }
}
