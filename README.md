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

**macOS Users:** Downloaded binaries may be blocked by Gatekeeper. After downloading, run:

```bash
xattr -d com.apple.quarantine pathuni
```

Or alternatively:

```bash
codesign -s - pathuni # You might need additional flags
```

## Usage

Create `~/.config/pathuni/my_paths.yaml`:

```yaml
all:
  tags: [base, essential]                     # Platform-level tags (NEW in v0.4.5)
  paths:
    - "$HOME/.local/bin"                      # Inherits: [base, essential]
    - path: "$HOME/.cargo/bin"                # Explicit tags override inheritance
      tags: [rust, dev]

macos:
  tags: [mac, gui]                            # Platform-level tags for macOS
  paths:
    - "/opt/homebrew/bin"                     # Inherits: [mac, gui]
    - path: "/opt/homebrew/sbin"              # Explicit tags override
      tags: [admin, homebrew]
    - path: "/Applications/Docker.app/Contents/Resources/bin"
      tags: [docker, work]
    - path: "/usr/local/special"              # Explicit empty override
      tags: []                                # No tags (breaks inheritance)

linux:
  # No tags field - platform has no tags to inherit
  paths:
    - "/home/linuxbrew/.linuxbrew/bin"        # No inheritance = no tags
    - "/usr/local/bin"                        # No inheritance = no tags
    - path: "/home/linuxbrew/.linuxbrew/sbin" # Explicit tags still work
      tags: [admin, homebrew]
    - path: "/opt/simple/bin"                 # Path object without tags field
      # Missing tags field + no platform tags = no tags
```

### Shell-specific Configuration

PowerShell on macOS doesn't automatically load system paths from `/etc/paths` and `/etc/paths.d/` like Unix shells do. You can enable this with:

```yaml
macos:
  powershell:
    include_system_paths: true # Loads system paths for PowerShell
```

With this setting, PowerShell will get the same comprehensive PATH that zsh/bash get automatically, including standard system directories like `/usr/bin`, `/bin`, etc.

### Platform-Level Tag Inheritance (NEW in v0.4.5)

You can now define tags at the platform level (`all`, `macos`, `linux`) that are automatically inherited by simple string paths. This reduces repetition and makes configuration more maintainable:

```yaml
all:
  tags: [base, essential]                     # All simple paths inherit these tags
  paths:
    - "/usr/local/bin"                        # Gets tags: [base, essential]
    - "/usr/bin"                              # Gets tags: [base, essential]
    - path: "/special/bin"                    # Explicit tags override inheritance
      tags: [admin, work]                     # Gets tags: [admin, work] - no inheritance
    - path: "/no/tags/bin"                    # Explicit empty array breaks inheritance
      tags: []                                # Gets no tags (not [base, essential])

macos:
  tags: [mac, desktop]                        # macOS-specific inheritance
  paths:
    - "/opt/homebrew/bin"                     # Gets tags: [mac, desktop]
    - path: "/Applications/Docker.app/Contents/Resources/bin"
      tags: [docker]                          # Gets tags: [docker] - overrides inheritance
```

**Key inheritance rules:**

- **Simple string paths** (like `"/usr/local/bin"`) inherit platform tags
- **Explicit path objects** with `tags:` field override inheritance completely
- **Empty tags array** (`tags: []`) explicitly means "no tags" (breaks inheritance)
- **Missing tags field** means "inherit platform tags"

This is especially powerful for filtering:

```bash
# Include only paths with platform-specific tags
pathuni dry-run --tags-include=mac   # Only macOS-tagged paths
pathuni dry-run --tags-include=base  # Only base-tagged paths from 'all'

# Exclude specific platforms
pathuni dry-run --tags-exclude=linux # Exclude Linux-tagged paths
```

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

- **Exact tags**: 3-20 characters, start with a letter, only letters/numbers/underscores
  - Examples: `dev`, `work_laptop`, `gaming2`, `MyProject`
- **Wildcard patterns**: Support glob-style patterns using `*`, `?`, `[...]` syntax
  - Examples: `work_*`, `server?`, `[abc]*`, `*_temp`

### Wildcard Tag Patterns (NEW in v0.5.0)

You can use glob-style wildcard patterns for flexible tag matching, perfect for hierarchical tag structures:

```bash
# Wildcard patterns using *
pathuni --tags-include="work_*"     # Matches: work_prod, work_dev, work_staging
pathuni --tags-exclude="*_temp"    # Matches: build_temp, cache_temp, any_temp

# Single character wildcards using ?  
pathuni --tags-include="dev?"       # Matches: dev1, dev2, devA (exactly 4 chars)
pathuni --tags-exclude="?unt"       # Matches: hunt, punt, bunt (exactly 4 chars)

# Character classes using [...]
pathuni --tags-include="server[123]"    # Matches: server1, server2, server3
pathuni --tags-exclude="[abc]*"         # Matches: app, audio, build, cache...
pathuni --tags-include="[a-z]*"         # Matches: any tag starting with a-z
pathuni --tags-exclude="[^test]*"       # Matches: any tag NOT starting with t,e,s

# Complex combinations
pathuni --tags-include="work_*,server*" --tags-exclude="*_temp"
# Include work_* OR server* patterns, but exclude anything ending in _temp

# Case-insensitive matching  
pathuni --tags-exclude="MA?OS"     # Matches: macos, MACOS, MacOS, etc.
```

**Supported wildcard syntax:**

- `*` - matches any sequence of characters (zero or more)
- `?` - matches exactly one character
- `[abc]` - matches any character in the set (a, b, or c)
- `[a-z]` - matches any character in the range (a through z)
- `[^abc]` - matches any character NOT in the set (anything except a, b, c)

**Pattern examples:**

- `work_*` → `work_prod`, `work_dev`, `work_staging`
- `dev?` → `dev1`, `dev2`, `devA` (but not `development`)
- `server[12]` → `server1`, `server2` (but not `server3`)
- `*_temp` → `build_temp`, `work_temp`, `cache_temp`
- `[a-c]*` → `app`, `build`, `cache` (any tag starting with a, b, or c)

**Note**: All wildcard matching is case-insensitive, so `Work_*` matches `work_prod`, `WORK_DEV`, etc.

### Generate PATH export

```bash
# Auto-detect shell and OS (default command)
pathuni
pathuni init

# Specify shell explicitly
pathuni init --shell=fish
pathuni --shell=powershell  # shortcut: global flags work on root command

# Specify OS explicitly (NEW in v0.4.6)
pathuni init --os=linux
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

# Dump output scope options
pathuni dump --scope=pathuni # Show only what pathuni would add
pathuni dump --scope=system  # Show only existing, path without pathuni additions
pathuni dump --scope=full    # Show existing path, plus pathuni additions (default)

# Different output formats
pathuni dump --format=json
pathuni dump --format=yaml --scope=pathuni
pathuni d -f json -s full  # using shortcuts and short flags
```

**Example dry-run outputs (NEW improved tree structure in v0.4.5):**

```bash
# All paths included (no filtering)
$ pathuni dry-run --os=macos
Evaluating: /Users/you/.config/pathuni/my_paths.yaml

OS    : macOS (specified)
Shell : zsh (detected)

5 Included Paths:
  [+] /Users/you/.local/bin
  [+] /Users/you/.cargo/bin
  [+] /opt/homebrew/bin
  [+] /opt/homebrew/sbin
  [+] /usr/local/special

1 Skipped Path:
  [!] /Applications/Docker.app/Contents/Resources/bin (not found)

5 paths included in total
1 skipped in total

# With tag filtering showing detailed skip reasons
$ pathuni dry-run --tags-include=essential
Evaluating: /Users/you/.config/pathuni/my_paths.yaml

OS    : macOS (detected)
Shell : zsh (detected)

1 Included Path:
  [+] /Users/you/.local/bin

4 Skipped Paths:
  [-] /Users/you/.cargo/bin
       └rust,dev != essential
  [-] /opt/homebrew/bin
       └mac != essential
  [-] /opt/homebrew/sbin
       └admin != essential
  [!] /Applications/Docker.app/Contents/Resources/bin (not found)

1 path included in total
4 skipped in total

# Complex filtering with inheritance and explicit empty tags,
# specifying zsh as the shell
$ pathuni dry-run --tags-exclude=gui --shell=zsh
Evaluating: /Users/you/.config/pathuni/my_paths.yaml

OS    : macOS (detected)
Shell : zsh (specified)

3 Included Paths:
  [+] /Users/you/.local/bin     # [base, essential]
  [+] /Users/you/.cargo/bin     # [rust, dev]
  [+] /usr/local/special        # [] (explicit empty - immune to gui filter)

2 Skipped Paths:
  [-] /opt/homebrew/bin
       └mac = gui
  [!] /Applications/Docker.app/Contents/Resources/bin (not found)

3 paths included in total
2 skipped in total
```

**Example dump outputs:**

```bash
$ pathuni dump --scope=pathuni
/Users/you/.local/bin
/opt/homebrew/bin
/opt/homebrew/sbin

$ pathuni dump --format=yaml --scope=pathuni
PATH:
    - /Users/you/.local/bin
    - /opt/homebrew/bin
    - /opt/homebrew/sbin

$ pathuni dump --format=json --scope=full
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
- **Platform-level tag inheritance**: Define tags once per platform, inherit automatically
- **Tag-based filtering**: Include/exclude paths by context (dev, work, gaming, etc.)
- **Wildcard tag patterns**: Use glob-style patterns (`work_*`, `server?`, `[abc]*`) for flexible filtering
- **Improved dry-run output**: Tree-structured output with detailed skip reasons
- **Path validation**: Only includes directories that actually exist
- **Lightweight**: Single binary, no dependencies
- **Mixed format**: Support both simple strings and tagged path entries

## Contributing

Pull requests, bug reports, and feature suggestions are welcome!

Areas that could use help:

- Windows support
- \*BSD support
- Additional shell support:
  - C shells (csh, tcsh, ...)
  - Next-gen, post-POSIX shells (elvish, nushell (nu), xonsh, ...)
- Performance improvements

## Development

```bash
make build         # Build optimised binary to bin/pathuni
make build-release # Build with maximum optimisation + UPX compression (if available)
make cross-compile # Build for multiple platforms (macOS/Linux on ARM64/AMD64)
make test          # Run all tests
make clean         # Clean build artifacts
make dev           # Quick build + run evaluation preview
make install       # Copy binary to ~/.local/bin
```

### Binary Size Optimisation

The build system includes several optimisations:

- **Compiler flags**: `-s -w -trimpath` remove debug symbols and build paths
- **UPX compression**: Automatically applied in `build-release` and `cross-compile` if UPX is installed
- **Cross-platform**: The Makefile handles UPX platform differences.
  - **NOTE**: UPX compression for macOS is officially unsupported until further notice.

Size comparison (typical results as of v0.4.0):

- Default Go build: ~6MB
- Optimised build: ~4MB (31% reduction)
- With UPX: ~1.6MB (74% reduction)
