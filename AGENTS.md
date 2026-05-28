# AGENTS.md

## Assistant Behaviour

- Write in British English: `colour` not `color`, `optimised` not `optimized`, `centre` not `center`.
- Be direct and pragmatic. Push back on questionable architectural decisions.
- When SOLID conflicts with minimalism, suggest both a strict and a pragmatic option with trade-offs.

## Commands

```bash
make build    # Build binary to bin/pathuni
make test     # Run all tests
make dev      # Build + run with --eval
make clean    # Remove bin/
make install  # Build + copy to ~/.local/bin
```

Run a single test:
```bash
go test ./cmd/pathuni/... -run TestName
```

Run tests with verbose output:
```bash
go test ./cmd/pathuni/... -v -run TestName
```

## Architecture

All code lives in a single Go package at `cmd/pathuni/` (`package main`). There are no sub-packages. The entry point is `main.go`, which wires up three Cobra commands:

- **`init`** (default): Generates a shell `export PATH=…` statement from config + system paths.
- **`dry-run`**: Prints a human-readable report of what would be included/skipped and why.
- **`dump`**: Outputs current effective paths in `plain`, `json`, or `yaml` format.

### Key files

| File | Responsibility |
|------|---------------|
| `main.go` | Cobra command definitions, global flag declarations, OS/shell detection |
| `config.go` | YAML loading, `Config`/`PlatformConfig`/`PathEntry` types, all dry-run report rendering logic, `EvaluateConfig*` functions |
| `shell.go` | Shell-specific renderers (`renderBash`, `renderFish`, `renderPwsh` and their `*Defer` variants), `runInit` |
| `tag.go` | Tag validation, `TagFilter` struct, `shouldIncludePath`, `getPathSkipReasons` |
| `paths.go` | Shared path resolution helpers: `resolvePathuniPaths`, `resolveSystemPathsContext`, `mergeFull`, `filterExisting`, PowerShell system path injection |
| `dump.go` | `runDump`, format helpers, `getCurrentPath`, `isValid*` guards |
| `helpers.go` | `getSystemPaths` (reads macOS `/etc/paths` + `/etc/paths.d/*`), `getShellSpecificPaths` |
| `test_utils.go` | `setupTestFilesystem` / `cleanupTestFilesystem` — creates `/tmp/pathuni/` mock tree |

### Config structure

```yaml
all:
  tags: [optional, platform-level-tags]
  paths:
    - /plain/string/path          # no tags: immune to tag filtering
    - path: /tagged/path
      tags: [tag1, tag2]          # explicit tags: subject to filtering
    - path: /inheriting/path      # tags omitted: inherits platform tags
macos:
  powershell:
    include_system_paths: true
    include_system_paths_as: system  # or: pathuni
    tags: [optional]
  paths: [...]
linux:
  paths: [...]
```

### Scope and prune flags

`--scope` (`system` | `pathuni` | `full`) and `--prune` (`none` | `pathuni` | `system` | `all`) are persistent flags shared across all three commands. `paths.go` owns the canonical resolution logic that all commands call.

### Tag semantics

- Path with `tags: nil` (field absent): inherits platform-level tags; if platform has none, untagged → immune to filtering.
- Path with `tags: []` (explicit empty): breaks inheritance; treated as explicitly untagged but still subject to filtering.
- Tag filters: comma = OR, plus = AND (e.g. `home,work+server`).

### Testing conventions

- Tests use table-driven style throughout.
- `setupTestFilesystem(t)` creates a mock directory tree under `/tmp/pathuni/`; all test configs must reference paths within it.
- Static reusable configs live in `cmd/pathuni/testdata/`; per-iteration dynamic configs use `os.CreateTemp`.
- The `PATHUNI_TEST_SYSTEM_PATHS_ROOT` env var is the test seam for `getSystemPaths()` — set it to redirect reads away from the real `/etc/paths`.
- Manual testing requires real paths; mock dirs are deleted after each test run.
