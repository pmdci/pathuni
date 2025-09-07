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

    config = filepath.Join("testdata", "valid_config.yaml")
    osOverride = "macOS"
    shell = "bash"
    t.Setenv("PATH", "/tmp/pathuni/usr/bin:/tmp/pathuni/bin")

    // system scope
    out := captureDryRunOutput(func() { _ = PrintDryRunReport(config, "macOS", "bash", false, false, "system") })
    if !strings.Contains(out, "Scope : system") { t.Fatalf("missing scope header in system output: %s", out) }
    if !strings.Contains(out, "[.] /tmp/pathuni/usr/bin") || !strings.Contains(out, "[.] /tmp/pathuni/bin") {
        t.Fatalf("system entries not listed with [.] marker: %s", out)
    }
    if strings.Contains(out, "[+] ") { t.Fatalf("unexpected pathuni marker in system scope: %s", out) }

    // full scope
    out = captureDryRunOutput(func() { _ = PrintDryRunReport(config, "macOS", "bash", false, false, "full") })
    if !strings.Contains(out, "Scope : full") { t.Fatalf("missing scope header in full output: %s", out) }
    if !strings.Contains(out, "[+] /tmp/pathuni/usr/local/bin") { t.Fatalf("expected pathuni entry not found with [+]: %s", out) }
    if !strings.Contains(out, "[.] /tmp/pathuni/bin") { t.Fatalf("system entry not found with [.]: %s", out) }
}

