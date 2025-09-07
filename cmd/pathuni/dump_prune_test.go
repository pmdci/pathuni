package main

import (
    "io"
    "os"
    "strings"
    "testing"
)

func captureDumpOutput(f func()) string {
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

func TestDump_Prune_SystemAndPathuni(t *testing.T) {
    setupTestFilesystem(t)
    defer cleanupTestFilesystem()

    // Ensure deferEnv is off for dump tests
    deferEnv = false

    // Deterministic PATH with a missing entry
    t.Setenv("PATH", "/tmp/pathuni/usr/bin:/does/not/exist:/tmp/pathuni/bin")
    scope = "system"
    dumpFormat = "plain"

    // prune=none keeps missing system path
    prune = "none"
    out := captureDumpOutput(runDump)
    if !strings.Contains(out, "/does/not/exist") {
        t.Fatalf("expected missing system path to remain for prune=none, got: %s", out)
    }

    // prune=system removes missing system path
    prune = "system"
    out = captureDumpOutput(runDump)
    if strings.Contains(out, "/does/not/exist") {
        t.Fatalf("expected missing system path to be pruned for prune=system, got: %s", out)
    }

    // Now verify pathuni behavior with temp config
    tmpCfg, err := os.CreateTemp("/tmp/pathuni/home/Pratt/.config/pathuni", "cfg-*.yaml")
    if err != nil { t.Fatalf("temp cfg: %v", err) }
    defer os.Remove(tmpCfg.Name())
    cfgContent := "all:\n  paths:\n    - \"/tmp/pathuni/usr/local/bin\"\n    - \"/tmp/pathuni/does-not-exist\"\n"
    if _, err := tmpCfg.WriteString(cfgContent); err != nil { t.Fatalf("write cfg: %v", err) }
    tmpCfg.Close()
    config = tmpCfg.Name()

    scope = "pathuni"

    prune = "none"
    out = captureDumpOutput(runDump)
    if !strings.Contains(out, "/tmp/pathuni/usr/local/bin") || !strings.Contains(out, "/tmp/pathuni/does-not-exist") {
        t.Fatalf("expected missing pathuni entry when prune=none, got: %s", out)
    }

    prune = "pathuni"
    out = captureDumpOutput(runDump)
    if strings.Contains(out, "/tmp/pathuni/does-not-exist") {
        t.Fatalf("expected missing pathuni entry to be pruned for prune=pathuni, got: %s", out)
    }

    // full scope: ensure system pruning applies and merge pathuni-first
    scope = "full"
    t.Setenv("PATH", "/tmp/pathuni/usr/bin:/does/not/exist:/tmp/pathuni/bin")

    prune = "system"
    out = captureDumpOutput(runDump)
    if strings.Contains(out, "/does/not/exist") {
        t.Fatalf("expected system missing pruned in full, got: %s", out)
    }
    // pathuni missing should still be present for prune=system
    if !strings.Contains(out, "/tmp/pathuni/does-not-exist") {
        t.Fatalf("expected pathuni missing present for prune=system, got: %s", out)
    }
}
