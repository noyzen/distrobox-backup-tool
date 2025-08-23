# Distrobox Management Tool

[![Go](https://img.shields.io/badge/Go-1.18%2B-blue?logo=go)](https://golang.org/) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT) [![Distrobox](https://img.shields.io/badge/Distrobox-Compatible-green)](https://github.com/89luca89/distrobox)

A simple, user-friendly CLI tool for managing [Distrobox](https://github.com/89luca89/distrobox) containers. Distrobox lets you run any Linux distribution inside a container on your host system, and this tool makes it easier to backup, restore, clone, edit, delete, and check the health of your containers. Built in Go for cross-platform compatibility, with colorful output, spinners, and optional GUI file pickers for a smooth experience.

Whether you're a developer juggling multiple environments or a Linux enthusiast experimenting with distros, this tool simplifies container lifecycle management without needing to remember complex commands.

## Features

- **Backup Containers**: Create compressed backups of your containers as `.tar` files. Supports both standard (shared home) and isolated (separate home) containers. For isolated ones, choose between combined or separated backups.
- **Restore Containers**: Load backups and recreate containers with options for systemd init and NVIDIA GPU integration. Automatically detects and handles isolated vs. standard types.
- **Clone Containers**: Make exact copies of existing containers with new names, preserving isolation status.
- **Edit Container Type**: Convert containers between standard (shared host home) and isolated (dedicated home folder) modes.
- **Delete Containers**: Safely remove containers with confirmation prompts.
- **Health Check**: Quickly test if a container is responsive by entering it and running a simple command.
- **User-Friendly Interface**: Interactive menu with colored output, progress spinners, and warnings for disk space or overwrites. Falls back to terminal input if GUI tools aren't available.
- **Dependencies Check**: Automatically detects Podman/Docker, Distrobox version, host OS, and optional tools like `tar`, `zenity`, or `kdialog`.
- **Cross-Platform**: Works on Linux (primary), with potential for macOS/Windows via Distrobox-compatible setups.

## Requirements

- **Distrobox**: Installed and functional. [Installation Guide](https://github.com/89luca89/distrobox#installation).
- **Container Runtime**: Either [Podman](https://podman.io/) (recommended) or [Docker](https://www.docker.com/).
- **Go**: Version 1.18+ to build from source (or download pre-built binaries from releases).
- **Optional**:
  - `tar`: For handling separated backups/restores of isolated home directories.
  - `zenity` (GNOME) or `kdialog` (KDE): For GUI file/folder selection dialogs.
- Sufficient disk space in your container storage path (automatically checked where possible).

This tool assumes you're running on a Linux host, as Distrobox is Linux-focused.

## Installation

### Option 1: Download Pre-Built Binary
Check the [Releases](https://github.com/yourusername/distrobox-management-tool/releases) page for pre-compiled binaries for your architecture (e.g., Linux AMD64). Download, make it executable, and run:

```bash
chmod +x distrobox-tool
./distrobox-tool
```

### Option 2: Build from Source
1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/distrobox-management-tool.git
   cd distrobox-management-tool
   ```
2. Build the binary:
   ```bash
   go build -o distrobox-tool main.go
   ```
3. Run it:
   ```bash
   ./distrobox-tool
   ```

Add it to your PATH for convenience: `mv distrobox-tool /usr/local/bin/`.

## Usage Guide

Run the tool with `./distrobox-tool` (or just `distrobox-tool` if in PATH). It starts with a main menu showing your containers and options.

### Main Menu
```
Distrobox Management Tool
Distrobox v1.7.2 | Host OS: Ubuntu 24.04 | Runtime: podman

=== Your Distrobox Containers ======================================
  1. ubuntu-dev                 Standard
  2. fedora-toolbox             Isolated
====================================================================
 1) Backup    2) Restore   3) Clone
 4) Edit      5) Delete    6) Health Check
 7) Exit

> Select an option:
```

- Enter a number (1-7) to choose an action.
- Press Enter without input to refresh the menu.
- Use `7` or Ctrl+C to exit.

### 1. Backup a Container
- Select a container from the list.
- Choose a destination folder (GUI picker if available, or manual path).
- Enter a base name for the backup file (e.g., `ubuntu-dev`).
- For isolated containers: Choose combined (one `.tar`) or separated (`.tar` for image + `.tar.gz` for home).
- The tool commits the container to a temp image, saves it, and cleans up. Checks for overwrites and space.

Example output file: `ubuntu-dev-isolated.tar`.

### 2. Restore a Container
- Select a `.tar` backup file (GUI or manual).
- Enter a new container name.
- Optionally enable systemd init and NVIDIA integration.
- The tool loads the image, creates the container, and restores home if separated.
- Detects isolated/standard from filename or companion `-home.tar.gz`.

### 3. Clone a Container
- Select a source container.
- Enter a unique new name.
- The tool creates a temp image, clones, and preserves isolation.

### 4. Edit Container Type
- Select a container.
- Confirm conversion: Standard → Isolated (adds dedicated home) or Isolated → Standard (deletes isolated home—careful!).
- The tool stops, commits, removes, and recreates the container with the new type.

**Warning**: Converting from isolated deletes the dedicated home folder permanently.

### 5. Delete a Container
- Select a container.
- Double-confirm to avoid accidents.
- Uses `distrobox-rm -f` for force removal.

### 6. Health Check
- Select a container.
- Runs `distrobox-enter` to execute `whoami` inside.
- Reports PASS/FAIL with error details if failed.

### Tips
- **Isolated vs. Standard**: Isolated containers have a dedicated `~/.local/share/distrobox/homes/<name>` folder. Standard ones share your host home.
- **Disk Space**: Backups/restores check free space in container storage (e.g., `~/.local/share/containers` for Podman).
- **Errors**: The tool logs errors in red and keeps temp images for recovery if something fails.
- **No Containers?** The menu shows "No Distrobox containers found." Create some with `distrobox-create` first.
- **GUI Fallback**: If no `zenity`/`kdialog`, it prompts for paths in the terminal.

## Contributing
Contributions welcome! Fork the repo, make changes, and submit a PR. Ideas:
- Add more edit options (e.g., rename, add flags).
- Support for Windows/macOS via WSL/OrbStack.
- Unit tests for utilities.

Please follow Go best practices and keep the UI simple.

## License
MIT License. See [LICENSE](LICENSE) for details.

## Acknowledgments
- Built on top of [Distrobox](https://github.com/89luca89/distrobox) by Luca Di Maio.
- Inspired by the need for easier container management in daily workflows.

If you find bugs or have suggestions, open an [issue](https://github.com/yourusername/distrobox-management-tool/issues). Happy containering! 🚀
