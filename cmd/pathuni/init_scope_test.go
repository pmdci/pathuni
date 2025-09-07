package main

import (
    "io"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

// captureOutput temporarily redirects os.Stdout to capture printed output.
func captureOutput(f func()) string {
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

func TestInit_Scopes_Render_Bash(t *testing.T) {
    setupTestFilesystem(t)
    defer cleanupTestFilesystem()

    // Reset globals to safe defaults for this test
    prune = "pathuni"
    deferEnv = false

    // Set deterministic config and environment
    config = filepath.Join("testdata", "valid_config.yaml")
    osOverride = "macOS"
    shell = "bash"
    t.Setenv("PATH", "/tmp/pathuni/usr/bin:/tmp/pathuni/bin")

    // pathuni scope
    scope = "pathuni"
    out := captureOutput(runInit)
    expectedPathuni := strings.Join([]string{
        "/tmp/pathuni/usr/local/bin",
        "/tmp/pathuni/home/Pratt/.local/bin",
        "/tmp/pathuni/usr/bin",
        "/tmp/pathuni/opt/homebrew/bin",
        "/tmp/pathuni/opt/homebrew/sbin",
        "/tmp/pathuni/home/Pratt/.cargo/bin",
        "/tmp/pathuni/Applications/Docker.app/Contents/Resources/bin",
    }, ":")
    expected := "export PATH=\"" + expectedPathuni + "\"\n"
    if out != expected {
        t.Errorf("pathuni scope render mismatch:\nwant: %q\n got: %q", expected, out)
    }

    // system scope
    scope = "system"
    out = captureOutput(runInit)
    expected = "export PATH=\"/tmp/pathuni/usr/bin:/tmp/pathuni/bin\"\n"
    if out != expected {
        t.Errorf("system scope render mismatch:\nwant: %q\n got: %q", expected, out)
    }

    // full scope (pathuni-first; dedupe removes duplicate /tmp/pathuni/usr/bin)
    scope = "full"
    out = captureOutput(runInit)
    expectedFull := expectedPathuni + ":/tmp/pathuni/bin"
    expected = "export PATH=\"" + expectedFull + "\"\n"
    if out != expected {
        t.Errorf("full scope render mismatch:\nwant: %q\n got: %q", expected, out)
    }

    // full scope with defer-env (prepend pathuni and reference $PATH)
    deferEnv = true
    scope = "full"
    out = captureOutput(runInit)
    expected = "export PATH=\"" + expectedPathuni + ":${PATH}\"\n"
    if out != expected {
        t.Errorf("full scope defer-env render mismatch:\nwant: %q\n got: %q", expected, out)
    }
    deferEnv = false

    // system scope with prune behavior
    scope = "system"
    prune = "none"
    out = captureOutput(runInit)
    // Should include both entries including non-existent ones (if any were present)
    expected = "export PATH=\"/tmp/pathuni/usr/bin:/tmp/pathuni/bin\"\n"
    if out != expected {
        t.Errorf("system scope no-prune mismatch:\nwant: %q\n got: %q", expected, out)
    }

    // Add a fake missing path to PATH and test prune=system
    t.Setenv("PATH", "/tmp/pathuni/usr/bin:/does/not/exist:/tmp/pathuni/bin")
    prune = "system"
    out = captureOutput(runInit)
    // Missing entry should be removed in output
    expected = "export PATH=\"/tmp/pathuni/usr/bin:/tmp/pathuni/bin\"\n"
    if out != expected {
        t.Errorf("system scope prune=system mismatch:\nwant: %q\n got: %q", expected, out)
    }

    // Now test pathuni list includes missing entries when prune=none|system
    // Create a temporary config with one existing and one missing path
    tmpCfg, err := os.CreateTemp("/tmp/pathuni/home/Pratt/.config/pathuni", "cfg-*.yaml")
    if err != nil { t.Fatalf("temp cfg: %v", err) }
    defer os.Remove(tmpCfg.Name())
    cfgContent := "all:\n  paths:\n    - \"/tmp/pathuni/usr/local/bin\"\n    - \"/tmp/pathuni/does-not-exist\"\n"
    if _, err := tmpCfg.WriteString(cfgContent); err != nil { t.Fatalf("write cfg: %v", err) }
    tmpCfg.Close()
    config = tmpCfg.Name()

    // scope=pathuni, prune=none should include both paths
    scope = "pathuni"
    prune = "none"
    out = captureOutput(runInit)
    if !strings.Contains(out, "/tmp/pathuni/usr/local/bin") || !strings.Contains(out, "/tmp/pathuni/does-not-exist") {
        t.Errorf("prune=none should include missing pathuni entries, got: %s", out)
    }

    // prune=pathuni should drop the missing one
    prune = "pathuni"
    out = captureOutput(runInit)
    if strings.Contains(out, "/tmp/pathuni/does-not-exist") {
        t.Errorf("prune=pathuni should drop missing pathuni entry, got: %s", out)
    }
}
