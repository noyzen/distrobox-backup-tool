📦 Distrobox Backup Tool

This is a command-line tool written in Go to simplify the management of your Distrobox containers. It provides an interactive menu to easily back up, restore, delete, and edit your containers.

✨ Features

    Backup: Create a .tar archive of your selected container, which can be stored as a backup.

    Restore: Restore a container from a .tar backup file. You can choose to restore it as a standard or isolated container.

    Delete: Permanently remove an existing Distrobox container.

    Edit: Convert a container from a "standard" configuration (sharing the host's home directory) to an "isolated" one (with a dedicated home directory), or vice-versa.

🛠️ Prerequisites

To use this tool, you must have the following dependencies installed on your system:

    Go: The Go programming language is required to compile and run the tool.

    Distrobox: The core tool for creating and managing containers.

    Container Runtime: Either Podman or Docker must be installed and running. The tool will automatically detect which one you have.

Optional Dependencies:
For a more user-friendly experience with a graphical file picker, you can install either zenity (common on GNOME/GTK-based desktops) or kdialog (common on KDE/Qt-based desktops). If neither is found, the tool will fall back to a terminal-based file path entry.

🚀 How to Use

    Save the Code: Save the provided Go code into a file named distrobox-backup-tool.go.

    Compile and Run: Open a terminal in the same directory as the file and run the following command. The go run command will compile and execute the program for you.
    Bash

    go run distrobox-backup-tool.go

    Navigate the Menu: The tool will present you with a main menu. Use the numbers to select an option and follow the on-screen prompts.

⚠️ Important Notes

    Irreversible Actions: The Delete and Edit functions are powerful. Converting a container from an isolated to a standard type will permanently delete the isolated home directory and all data within it. Always be sure before proceeding.

    Backup Naming: When creating a backup, use a descriptive name without file extensions. The tool will automatically append .tar to the filename.

    Ctrl+C: You can use Ctrl+C at any time to exit the tool.
