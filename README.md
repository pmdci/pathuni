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
# Auto-detect shell and OS
pathuni

# Specify shell explicitly
pathuni --shell=fish
pathuni --shell=powershell
```

### Preview what will be included

```bash
pathuni --eval
```

Output:

```
Evaluating: /Users/you/.config/pathuni/my_paths.yaml

OS    : macOS
Shell : zsh (inferred)

Included Paths:
  [+] /Users/you/.local/bin
  [+] /opt/homebrew/bin
  [+] /opt/homebrew/sbin

Skipped (not found):
  [-] /some/missing/path

3 paths included
1 skipped
```

## Supported Shells

- bash, zsh, sh (uses `export PATH=`)
- fish (uses `set -gx PATH`)
- powershell (uses `$env:PATH =`)

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
make build    # Build binary
make test     # Run tests
make clean    # Clean build artifacts
make dev      # Quick build + eval
```
