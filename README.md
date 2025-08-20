# Pathuni

A lightweight, cross-platform PATH management tool for dotfiles that work across multiple operating systems (macOS and Linux for now) and shells.

## What it does

Pathuni reads a simple YAML config file and generates shell-specific PATH export commands. It validates that directories exist before including them and supports platform-specific path lists.

## Installation

```bash
git clone <your-repo>
cd pathuni
make build
make install  # copies to ~/.local/bin
```

## Usage

Create `~/.config/pathuni/my_paths.yaml`:

```yaml
All:
  - "$HOME/.local/bin"

macOS:
  - "/opt/homebrew/bin"
  - "/opt/homebrew/sbin"

Linux:
  - "/home/linuxbrew/.linuxbrew/bin"
  - "/home/linuxbrew/.linuxbrew/sbin"
```

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

**Example dry-run output:**

```
$ pathuni dry-run
Evaluating: /Users/you/.config/pathuni/my_paths.yaml

OS    : macOS
Shell : zsh (inferred)

Included Paths:
  [+] /Users/you/.local/bin
  [+] /opt/homebrew/bin
  [+] /opt/homebrew/sbin

3 paths included
0 skipped

Output:
  export PATH="/Users/you/.local/bin:/opt/homebrew/bin:/opt/homebrew/sbin:$PATH"

To apply: run 'pathuni --shell=zsh'
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

- **bash|zsh|sh** - uses `export PATH=`
- **fish** - uses `set -gx PATH`
- **powershell** - uses `$env:PATH =`

## Why Pathuni?

Most dotfiles managers are heavyweight solutions for simple PATH management. Pathuni aims to do one thing well: cross-platform PATH exports with validation, perfect for developers juggling multiple environments without wanting full dotfiles orchestration.

## Contributing

This is a very early release. Pull requests, bug reports, and feature suggestions are welcome!

Areas that could use help:

- Windows support
- \*BSD support
- Additional shell support
- Test coverage
- Performance improvements

## Development

```bash
make build    # Build binary to bin/pathuni
make test     # Run all tests
make clean    # Clean build artifacts
make dev      # Quick build + run evaluation preview
make install  # Copy binary to ~/.local/bin
```
