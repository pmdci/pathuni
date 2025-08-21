# Pathuni

A lightweight, cross-platform PATH management tool for dotfiles that work across multiple operating systems (macOS and Linux for now) and shells.

## What it does

Pathuni reads a simple YAML config file and generates shell-specific PATH export commands. It validates that directories exist before including them and supports platform-specific path lists.

## Installation

### Quick Install (Recommended)

```bash
# Install latest release
curl -sSL https://raw.githubusercontent.com/pmdci/pathuni/main/install.sh | bash
```

### Homebrew (Coming Soon)

```bash
brew tap pmdci/pathuni
brew install pathuni
```

### Manual Build

```bash
git clone https://github.com/pmdci/pathuni
cd pathuni
make build
make install  # copies to ~/.local/bin
```

### Download Binary

Download pre-built binaries from the [releases page](https://github.com/pmdci/pathuni/releases).

## Usage

Create `~/.config/pathuni/my_paths.yaml`:

```yaml
all:
  paths:
    - "$HOME/.local/bin"
    - path: "$HOME/.cargo/bin"
      tags:
        - rust
        - dev

macos:
  paths:
    - "/opt/homebrew/bin"
    - path: "/opt/homebrew/sbin"
      tags:
        - admin
        - homebrew
    - path: "/Applications/Docker.app/Contents/Resources/bin"
      tags:
        - docker
        - work

linux:
  paths:
    - "/home/linuxbrew/.linuxbrew/bin"
    - path: "/home/linuxbrew/.linuxbrew/sbin"
      tags:
        - admin
        - homebrew
```

### Shell-specific Configuration

PowerShell on macOS doesn't automatically load system paths from `/etc/paths` and `/etc/paths.d/` like Unix shells do. You can enable this with:

```yaml
macos:
  powershell:
    include_system_paths: true # Loads system paths for PowerShell
```

With this setting, PowerShell will get the same comprehensive PATH that zsh/bash get automatically, including standard system directories like `/usr/bin`, `/bin`, etc.

### Tag-based Path Filtering

You can filter paths by tags using `--tags-include` and `--tags-exclude` flags. Tags support both OR logic (comma-separated) and AND logic (plus-separated):

```bash
# Include only paths tagged with 'dev'
pathuni --tags-include=dev

# Include paths tagged with 'dev' OR 'work'
pathuni --tags-include=dev,work

# Include paths that have BOTH 'work' AND 'admin' tags
pathuni --tags-include=work+admin

# Exclude paths tagged with 'docker'
pathuni --tags-exclude=docker

# Exclude paths tagged with 'docker' OR 'gaming'
pathuni --tags-exclude=docker,gaming

# Exclude paths that have BOTH 'work' AND 'admin' tags
pathuni --tags-exclude=work+admin

# Complex: include 'dev' paths but exclude 'work' paths
pathuni --tags-include=dev --tags-exclude=work
```

**Important:**

- **Untagged paths** are immune to tag filtering and are always included
- **Exclude wins** - if a path matches both include and exclude conditions, it's excluded
- **No tag flags** - all paths (tagged and untagged) are included

**Tag naming rules:**

- 3-20 characters
- Start with a letter
- Only letters, numbers, and underscores
- Examples: `dev`, `work_laptop`, `gaming2`, `MyProject`

### Generate PATH export

```bash
# Auto-detect shell and OS (default command)
pathuni
pathuni init

# Specify shell explicitly
pathuni init --shell=fish
pathuni --shell=powershell  # shortcut: global flags work on root command
```

### Preview what will be included

```bash
pathuni dry-run
pathuni n  # shortcut

# With specific shell
pathuni dry-run --shell=bash
```

### Inspect current PATH

```bash
# Show all current PATH entries
pathuni dump
pathuni d  # shortcut

# Show only what pathuni would add
pathuni dump --include=pathuni

# Different output formats
pathuni dump --format=json
pathuni dump --format=yaml --include=pathuni
pathuni d -f json -i all  # using shortcuts and short flags
```

**Example dry-run outputs:**

```bash
# All paths included (default behavior)
$ pathuni dry-run --shell=zsh
Evaluating: /Users/you/.config/pathuni/my_paths.yaml

OS    : macOS
Shell : zsh (specified)

4 Included Paths:
  [+] /Users/you/.local/bin
  [+] /Users/you/.cargo/bin
  [+] /opt/homebrew/bin
  [+] /opt/homebrew/sbin

2 Skipped Paths (not found):
  [!] /nonexistent/missing/bin
  [!] /another/missing/path

4 paths included in total
2 skipped in total

Output would be:
  export PATH="/Users/you/.local/bin:/Users/you/.cargo/bin:/opt/homebrew/bin:/opt/homebrew/sbin"

# With tag filtering and mixed skip reasons
$ pathuni dry-run --tags-exclude=personal --shell=zsh

OS    : macOS
Shell : zsh (specified)

3 Included Paths:
  [+] /Users/you/.local/bin
  [+] /opt/homebrew/bin
  [+] /opt/homebrew/sbin

2 Skipped Paths (not found):
  [!] /nonexistent/missing/bin
  [!] /another/missing/path

1 Skipped Path (filtered by tags):
  [-] /Users/you/Documents

3 paths included in total
3 skipped in total

Output would be:
  export PATH="/Users/you/.local/bin:/opt/homebrew/bin:/opt/homebrew/sbin"
```

**Example dump outputs:**

```bash
$ pathuni dump --include=pathuni
/Users/you/.local/bin
/opt/homebrew/bin
/opt/homebrew/sbin

$ pathuni dump --format=yaml --include=pathuni
PATH:
    - /Users/you/.local/bin
    - /opt/homebrew/bin
    - /opt/homebrew/sbin

$ pathuni dump --format=json --include=all
{"PATH":["/Users/you/.local/bin","/opt/homebrew/bin",...]}
```

## Supported Shells

- **POSIX shells** (sh, ash, bash, dash, ksh, mksh, yash, zsh) - uses `export PATH=`
- **fish** - uses `set -gx PATH`
- **powershell** - uses `$env:PATH =`
  - On macOS, can automatically include system paths from `/etc/paths` and `/etc/paths.d/` using the `include_system_paths` YAML setting (see above under _Shell-specific Configuration_).

## Why Pathuni?

Most dotfiles managers are heavyweight solutions for simple PATH management. Pathuni aims to do one thing well: cross-platform PATH exports with validation and flexible tag-based filtering, perfect for developers juggling multiple environments without wanting full dotfiles orchestration.

**Key features:**

- **Cross-platform**: Works on macOS, Linux with plans for Windows/\*BSD
- **Multi-shell**: bash, zsh, fish, PowerShell support
- **Tag-based filtering**: Include/exclude paths by context (dev, work, gaming, etc.)
- **Path validation**: Only includes directories that actually exist
- **Lightweight**: Single binary, no dependencies
- **Mixed format**: Support both simple strings and tagged path entries

## Contributing

This is a very early release. Pull requests, bug reports, and feature suggestions are welcome!

Areas that could use help:

- Windows support
- \*BSD support
- Additional shell support:
  - C shells (csh, tcsh)
  - Next-gen, post-POSIX shells: elvish, nushell (nu), xonsh
- Performance improvements

## Development

```bash
make build         # Build optimized binary to bin/pathuni
make build-release # Build with maximum optimization + UPX compression (if available)
make cross-compile # Build for multiple platforms (macOS/Linux on ARM64/AMD64)
make test          # Run all tests
make clean         # Clean build artifacts
make dev           # Quick build + run evaluation preview
make install       # Copy binary to ~/.local/bin
```

### Binary Size Optimization

The build system includes several optimizations:

- **Compiler flags**: `-s -w -trimpath` remove debug symbols and build paths
- **UPX compression**: Automatically applied in `build-release` and `cross-compile` if UPX is installed
- **Cross-platform**: The Makefile handles UPX platform differences (macOS requires `--force-macos`)

Size comparison (typical results as of v0.4.0):

- Default Go build: ~6MB
- Optimized build: ~4MB (31% reduction)
- With UPX: ~1.6MB (74% reduction)
