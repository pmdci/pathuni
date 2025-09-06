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
}
