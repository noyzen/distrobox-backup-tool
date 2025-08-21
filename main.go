// Filename: distrobox-backup-tool.go
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// --- Configuration & Constants ---

// ANSI color codes for beautiful output
const (
	colorReset     = "\033[0m"
	colorRed       = "\033[31m"
	colorGreen     = "\033[32m"
	colorYellow    = "\033[33m"
	colorBlue      = "\033[34m"
	colorMagenta   = "\033[35m"
	colorCyan      = "\033[36m"
	colorWhite     = "\033[37m"
	colorBold      = "\033[1m"
	colorUnderline = "\033[4m"
)

// Container represents a distrobox container
type Container struct {
	Name  string
	ID    string
	Image string
}

var (
	containerRuntime string // Will be "podman" or "docker"
	guiFilePicker    string // Will be "zenity" or "kdialog"
	distroboxVersion string
	hostDistroName   string
)

// --- Main Application Logic ---

func main() {
	clearScreen()
	printHeader()

	checkDependencies()

	// Main application loop
	for {
		containers, err := getContainers()
		if err != nil {
			logError("Could not list Distrobox containers. Is distrobox installed and running correctly?")
			logError(err.Error())
			os.Exit(1)
		}

		displayMenu(containers)
		if !handleUserChoice(containers) {
			return // Exit if user chooses to
		}
	}
}

// --- Core Feature Handlers ---

// handleUserChoice processes the main menu selection.
func handleUserChoice(containers []Container) bool {
	fmt.Printf("%s> Select an option: %s", colorBold, colorReset)
	choiceStr := readUserInput()
	if choiceStr == "" {
		return true // Go back to main menu
	}
	choice, err := strconv.Atoi(choiceStr)
	if err != nil {
		logWarning("Invalid option. Please enter a number.")
		time.Sleep(2 * time.Second)
		return true
	}

	switch choice {
	case 1:
		if len(containers) == 0 {
			logWarning("No containers available to backup.")
			time.Sleep(2 * time.Second)
			return true
		}
		handleBackup(containers)
	case 2:
		handleRestore()
	case 3:
		if len(containers) == 0 {
			logWarning("No containers available to delete.")
			time.Sleep(2 * time.Second)
			return true
		}
		handleDelete(containers)
	case 4:
		if len(containers) == 0 {
			logWarning("No containers available to edit.")
			time.Sleep(2 * time.Second)
			return true
		}
		handleEdit(containers)
	case 5:
		fmt.Printf("\n%s👋 Goodbye!%s\n", colorCyan, colorReset)
		return false // Exit the loop
	default:
		logWarning("Invalid option. Please try again.")
		time.Sleep(2 * time.Second)
	}
	return true
}

// handleBackup guides the user through backing up a container.
func handleBackup(containers []Container) {
	clearScreen()
	fmt.Printf("%s%s📦 Backup Container%s\n\n", colorBold, colorGreen, colorReset)
	printContainerList(containers)
	fmt.Printf("%s%sHint:%s Use 'Ctrl+C' to return to the main menu at any time.\n\n", colorYellow, colorUnderline, colorReset)

	// 1. Select Container from main menu list
	containerIndex := selectItem("Enter the number of the container to backup", len(containers))
	if containerIndex == 0 {
		return
	}
	selectedContainer := containers[containerIndex-1]

	// 2. Select Destination
	fmt.Println()
	logInfo("Please choose a backup destination folder.")
	destDir, err := selectDirectory("Select Backup Folder")
	if err != nil || destDir == "" {
		logError("No valid destination directory selected. Aborting.")
		time.Sleep(2 * time.Second)
		return
	}

	// 3. Get Backup Name
	fmt.Println()
	fmt.Printf("%s> Enter a name for the backup file (e.g., 'ubuntu-dev-backup'): %s", colorBold, colorReset)
	backupName := readUserInput()
	if backupName == "" {
		logWarning("Backup name cannot be empty. Aborting.")
		time.Sleep(2 * time.Second)
		return
	}
	backupFile := filepath.Join(destDir, backupName+".tar")

	// 4. Check for Overwrite
	if _, err := os.Stat(backupFile); err == nil {
		fmt.Printf("%s⚠️  File '%s' already exists. Overwrite? (y/N): %s", colorYellow, backupFile, colorReset)
		if !confirmAction() {
			logInfo("Backup cancelled by user.")
			time.Sleep(2 * time.Second)
			return
		}
	}

	// 5. Perform Backup
	logInfo(fmt.Sprintf("Backing up '%s' to '%s'...", selectedContainer.Name, backupFile))

	tempImageName := fmt.Sprintf("distrobox-backup-%s:%d", selectedContainer.ID, time.Now().Unix())

	done := make(chan bool)
	go showSpinner("Processing...", done)

	// Commit container to a temporary image
	_, err = runCommand(containerRuntime, "commit", selectedContainer.Name, tempImageName)
	if err != nil {
		done <- true
		logError("Failed to commit container.")
		logError(err.Error())
		time.Sleep(5 * time.Second)
		return
	}

	// Save the image to a tar file
	_, err = runCommand(containerRuntime, "save", "-o", backupFile, tempImageName)
	if err != nil {
		done <- true
		logError("Failed to save image to tar file.")
		// Attempt cleanup even on failure
		runCommand(containerRuntime, "rmi", tempImageName)
		time.Sleep(5 * time.Second)
		return
	}

	// Cleanup temporary image
	_, err = runCommand(containerRuntime, "rmi", tempImageName)
	if err != nil {
		done <- true
		// This is not a fatal error for the backup itself
		logWarning(fmt.Sprintf("Could not clean up temporary image '%s'. You may want to remove it manually.", tempImageName))
	}

	done <- true
	logSuccess(fmt.Sprintf("✅ Backup for '%s' completed successfully!", selectedContainer.Name))
	time.Sleep(3 * time.Second)
}

// handleRestore guides the user through restoring a container from a backup.
func handleRestore() {
	clearScreen()
	fmt.Printf("%s%s📦 Restore Container%s\n\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s%sHint:%s Select a backup file to restore from. 'Ctrl+C' to return.\n\n", colorYellow, colorUnderline, colorReset)

	// 1. Select Backup File
	logInfo("Please choose a backup file (.tar) to restore.")
	backupFile, err := selectFile("Select Backup File", "*.tar")
	if err != nil || backupFile == "" {
		logError("No backup file selected. Aborting.")
		time.Sleep(2 * time.Second)
		return
	}

	// 2. Load Image
	logInfo(fmt.Sprintf("Loading image from '%s'...", backupFile))
	done := make(chan bool)
	go showSpinner("Loading...", done)

	output, err := runCommand(containerRuntime, "load", "-i", backupFile)
	done <- true
	if err != nil {
		logError("Failed to load image from backup file.")
		logError(err.Error())
		time.Sleep(5 * time.Second)
		return
	}

	// Robustly extract loaded image name, including the tag.
	loadedImage := ""
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Loaded image:") {
			parts := strings.SplitN(line, "Loaded image:", 2)
			if len(parts) == 2 {
				loadedImage = strings.TrimSpace(parts[1])
				break // Found it
			}
		}
	}

	if loadedImage == "" {
		logError("Could not determine the name of the loaded image. Aborting.")
		time.Sleep(3 * time.Second)
		return
	}
	logSuccess(fmt.Sprintf("Image '%s' loaded successfully.", loadedImage))

	// 3. Get New Container Name
	fmt.Println()
	fmt.Printf("%s> Enter a name for the new container: %s", colorBold, colorReset)
	containerName := readUserInput()
	if containerName == "" {
		logWarning("Container name cannot be empty. Aborting.")
		runCommand(containerRuntime, "rmi", loadedImage) // Cleanup loaded image
		time.Sleep(2 * time.Second)
		return
	}

	// 4. Choose Isolation Type
	fmt.Println()
	fmt.Printf("%s%sHow would you like to restore this container?%s\n", colorBold, colorUnderline, colorReset)
	fmt.Printf("  %s1)%s Standard Box (Shares your host Home directory)\n", colorGreen, colorReset)
	fmt.Printf("  %s2)%s Isolated Box (Has its own separate Home directory)\n", colorBlue, colorReset)
	restoreType := selectItem("Select type", 2)
	if restoreType == 0 {
		runCommand(containerRuntime, "rmi", loadedImage)
		return
	}

	// 5. Create Distrobox
	args := []string{"--name", containerName, "--image", loadedImage}

	done = make(chan bool)
	go showSpinner("Creating container...", done)

	if restoreType == 2 {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			done <- true
			logError("Could not determine user home directory. Aborting isolated restore.")
			runCommand(containerRuntime, "rmi", loadedImage) // Cleanup
			time.Sleep(3 * time.Second)
			return
		}
		isolatedHomePath := filepath.Join(homeDir, ".local", "share", "distrobox", "homes", containerName)
		args = append(args, "--home", isolatedHomePath)
		logInfo(fmt.Sprintf("Creating new %sISOLATED%s container '%s'...", colorBold, colorReset, containerName))
		logInfo(fmt.Sprintf("Container home will be at: %s", isolatedHomePath))
	} else {
		logInfo(fmt.Sprintf("Creating new %sSTANDARD%s container '%s'...", colorBold, colorReset, containerName))
	}

	_, err = runCommand("distrobox-create", args...)
	done <- true

	if err != nil {
		logError(fmt.Sprintf("Failed to create container '%s'.", containerName))
		logError(err.Error())
		logInfo(fmt.Sprintf("The loaded image '%s' was kept. You can try creating the container again manually or remove the image.", loadedImage))
		time.Sleep(5 * time.Second)
		return
	}

	runCommand(containerRuntime, "rmi", loadedImage) // Cleanup loaded image after successful restore

	logSuccess(fmt.Sprintf("✅ Container '%s' restored successfully!", containerName))
	time.Sleep(3 * time.Second)
}

// handleEdit allows the user to change container properties.
func handleEdit(containers []Container) {
	clearScreen()
	fmt.Printf("%s%s🔧 Edit Container%s\n\n", colorBold, colorMagenta, colorReset)
	printContainerList(containers)
	fmt.Printf("%s%sHint:%s This tool can convert a container from Standard to Isolated, or vice-versa.\n\n", colorYellow, colorUnderline, colorReset)

	// 1. Select Container
	containerIndex := selectItem("Enter the number of the container to edit", len(containers))
	if containerIndex == 0 {
		return
	}
	selectedContainer := containers[containerIndex-1]

	// 2. Detect Container Type
	isIsolated, isolatedHomePath := isContainerIsolated(selectedContainer.Name)

	var targetType string
	var prompt string
	if isIsolated {
		targetType = "STANDARD"
		prompt = fmt.Sprintf("Container '%s' is currently ISOLATED. Convert to STANDARD?", selectedContainer.Name)
	} else {
		targetType = "ISOLATED"
		prompt = fmt.Sprintf("Container '%s' is currently STANDARD. Convert to ISOLATED?", selectedContainer.Name)
	}

	logInfo(prompt)
	fmt.Printf("This involves recreating the container. Continue? (y/N): ")
	if !confirmAction() {
		logInfo("Edit cancelled.")
		time.Sleep(2 * time.Second)
		return
	}

	// 3. Specific Warning for Isolated -> Standard
	if isIsolated {
		logWarning(fmt.Sprintf("Converting to STANDARD will PERMANENTLY DELETE the isolated home directory:"))
		logWarning(isolatedHomePath)
		logWarning("All data inside will be lost. The container will use your host's home directory instead.")
		fmt.Printf("%sAre you absolutely sure? (y/N): %s", colorRed, colorReset)
		if !confirmAction() {
			logInfo("Edit cancelled.")
			time.Sleep(2 * time.Second)
			return
		}
	}

	// 4. Perform Conversion
	done := make(chan bool)
	go showSpinner("Converting container...", done)

	// a. Stop the container
	_, err := runCommand(containerRuntime, "stop", selectedContainer.Name)
	if err != nil {
		done <- true
		logError(fmt.Sprintf("Failed to stop container '%s'. Aborting.", selectedContainer.Name))
		time.Sleep(5 * time.Second)
		return
	}

	// b. Commit to a temporary image
	tempImageName := fmt.Sprintf("distrobox-convert-%s:%d", selectedContainer.ID, time.Now().Unix())
	_, err = runCommand(containerRuntime, "commit", selectedContainer.Name, tempImageName)
	if err != nil {
		done <- true
		logError("Failed to commit container to a temporary image. Aborting.")
		time.Sleep(5 * time.Second)
		return
	}

	// c. Remove the old container
	_, err = runCommand("distrobox-rm", selectedContainer.Name, "--force")
	if err != nil {
		done <- true
		logError("Failed to remove the old container. You may need to clean up manually. Aborting.")
		runCommand(containerRuntime, "rmi", tempImageName) // cleanup temp image
		time.Sleep(5 * time.Second)
		return
	}

	// d. Create the new container
	args := []string{"--name", selectedContainer.Name, "--image", tempImageName}
	if targetType == "ISOLATED" {
		newIsolatedHome, _ := getIsolatedHomePath(selectedContainer.Name)
		args = append(args, "--home", newIsolatedHome)
	}

	_, err = runCommand("distrobox-create", args...)
	if err != nil {
		done <- true
		logError(fmt.Sprintf("Failed to create the new %s container.", targetType))
		logError("The temporary image and data have been kept for manual recovery.")
		logInfo(fmt.Sprintf("Temporary image: %s", tempImageName))
		time.Sleep(5 * time.Second)
		return
	}

	// e. Cleanup
	if isIsolated {
		err = os.RemoveAll(isolatedHomePath)
		if err != nil {
			logWarning(fmt.Sprintf("Failed to delete the old isolated home directory: %s", isolatedHomePath))
			logWarning("You may want to remove it manually.")
		}
	}
	runCommand(containerRuntime, "rmi", tempImageName)

	done <- true
	logSuccess(fmt.Sprintf("✅ Container '%s' successfully converted to %s!", selectedContainer.Name, targetType))
	time.Sleep(3 * time.Second)
}

// handleDelete guides the user through deleting a container.
func handleDelete(containers []Container) {
	clearScreen()
	fmt.Printf("%s%s🗑️ Delete Container%s\n\n", colorBold, colorRed, colorReset)
	printContainerList(containers)
	fmt.Printf("%s%sHint:%s This action is irreversible. Be sure before you delete.\n\n", colorYellow, colorUnderline, colorReset)

	// 1. Select Container from main menu list
	containerIndex := selectItem("Enter the number of the container to DELETE", len(containers))
	if containerIndex == 0 {
		return
	}
	selectedContainer := containers[containerIndex-1]

	// 2. Confirmation
	logWarning(fmt.Sprintf("You are about to permanently delete the container '%s'.", selectedContainer.Name))
	fmt.Printf("%sThis action cannot be undone. Are you sure? (y/N): %s", colorRed, colorReset)
	if !confirmAction() {
		logInfo("Deletion cancelled by user.")
		time.Sleep(2 * time.Second)
		return
	}

	// 3. Perform Deletion
	logInfo(fmt.Sprintf("Deleting '%s'...", selectedContainer.Name))
	done := make(chan bool)
	go showSpinner("Deleting...", done)
	_, err := runCommand("distrobox-rm", selectedContainer.Name, "--force")
	done <- true

	if err != nil {
		logError(fmt.Sprintf("Failed to delete container '%s'.", selectedContainer.Name))
		logError(err.Error())
		time.Sleep(5 * time.Second)
		return
	}

	logSuccess(fmt.Sprintf("🗑️  Container '%s' has been deleted.", selectedContainer.Name))
	time.Sleep(3 * time.Second)
}

// --- UI & Display Functions ---

// printHeader displays the main application header with a simple text-based title.
func printHeader() {
	fmt.Printf("%s%sDistrobox Backup Tool%s\n", colorBold, colorYellow, colorReset)
	fmt.Printf("Version: %s | Host OS: %s\n\n", distroboxVersion, hostDistroName)
}

// displayMenu prints the main menu to the console.
func displayMenu(containers []Container) {
	clearScreen()
	printHeader()
	fmt.Printf("%s=== Distrobox Containers =================================%s\n", colorBlue, colorReset)
	if len(containers) == 0 {
		fmt.Printf("  %sNo Distrobox containers found.%s\n", colorYellow, colorReset)
	} else {
		printContainerList(containers)
	}
	fmt.Printf("%s==========================================================%s\n", colorBlue, colorReset)
	fmt.Printf(" %s1)%s Backup   %s2)%s Restore   %s3)%s Delete   %s4)%s Edit   %s5)%s Exit\n",
		colorGreen, colorReset, colorCyan, colorReset, colorRed, colorReset, colorMagenta, colorReset, colorWhite, colorReset)
	fmt.Println()
	fmt.Printf("%s%sHint:%s Choose an action to perform on your containers.\n", colorYellow, colorUnderline, colorReset)
}

// printContainerList displays the formatted list of containers.
func printContainerList(containers []Container) {
	for i, c := range containers {
		isIsolated, _ := isContainerIsolated(c.Name)
		statusColor := colorGreen
		if isIsolated {
			statusColor = colorBlue
		}
		status := "Standard"
		if isIsolated {
			status = "Isolated"
		}
		fmt.Printf("  %s%d.%s %-25s  %s(%s)%s\n", colorBold, i+1, colorReset, c.Name, statusColor, status, colorReset)
	}
}

// showSpinner displays a simple loading animation.
func showSpinner(message string, done chan bool) {
	spinner := []string{"|", "/", "-", "\\"}
	i := 0
	for {
		select {
		case <-done:
			fmt.Printf("\r%s... Done!              \n", message)
			return
		default:
			fmt.Printf("\r%s %s ", message, spinner[i])
			i = (i + 1) % len(spinner)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// --- Helper & Utility Functions ---

// checkDependencies ensures required CLIs are installed and gets system info.
func checkDependencies() {
	if !commandExists("distrobox") {
		logError("FATAL: 'distrobox' command not found. Please install it first to use this tool.")
		os.Exit(1)
	}

	// Get distrobox version
	output, err := runCommand("distrobox", "--version")
	if err == nil {
		distroboxVersion = strings.TrimSpace(output)
	}

	// Get host distro name
	hostDistroName = "Unknown"
	if _, err := os.Stat("/etc/os-release"); err == nil {
		content, _ := os.ReadFile("/etc/os-release")
		re := regexp.MustCompile(`(?m)^NAME="?([^"\n]+)"?`)
		matches := re.FindStringSubmatch(string(content))
		if len(matches) > 1 {
			hostDistroName = matches[1]
		}
	}

	// Check for container runtime
	if commandExists("podman") {
		containerRuntime = "podman"
	} else if commandExists("docker") {
		containerRuntime = "docker"
	} else {
		logError("FATAL: Neither 'podman' nor 'docker' command found.")
		logError("Distrobox requires one of these runtimes to function.")
		os.Exit(1)
	}
	logInfo(fmt.Sprintf("Using '%s' as the container runtime.", containerRuntime))

	// Check for optional GUI dependencies
	if commandExists("zenity") {
		guiFilePicker = "zenity"
	} else if commandExists("kdialog") {
		guiFilePicker = "kdialog"
	} else {
		logWarning("No GUI file picker (zenity/kdialog) found. Falling back to terminal input.")
		logWarning("For a better experience, consider installing one (e.g., 'sudo dnf install zenity').")
	}
}

// getIsolatedHomePath constructs the expected path for an isolated container's home.
func getIsolatedHomePath(containerName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".local", "share", "distrobox", "homes", containerName), nil
}

// isContainerIsolated checks if a container has a dedicated home directory.
func isContainerIsolated(containerName string) (bool, string) {
	isolatedHomePath, err := getIsolatedHomePath(containerName)
	if err != nil {
		return false, ""
	}

	if _, err := os.Stat(isolatedHomePath); err == nil {
		return true, isolatedHomePath
	}

	return false, ""
}

// getContainers fetches the list of available distroboxes.
func getContainers() ([]Container, error) {
	out, err := exec.Command("distrobox-list", "--no-color").Output()
	if err != nil {
		if strings.Contains(string(out), "No distroboxes found") {
			return []Container{}, nil
		}
		return nil, err
	}

	var containers []Container
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if !strings.Contains(line, "|") || strings.Contains(line, "ID") || strings.Contains(line, "NAME") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) >= 4 {
			containers = append(containers, Container{
				ID:    strings.TrimSpace(parts[0]),
				Name:  strings.TrimSpace(parts[1]),
				Image: strings.TrimSpace(parts[3]),
			})
		}
	}
	return containers, nil
}

// selectDirectory prompts for a directory, using GUI if available.
func selectDirectory(title string) (string, error) {
	if guiFilePicker != "" {
		var cmd *exec.Cmd
		if guiFilePicker == "zenity" {
			cmd = exec.Command("zenity", "--file-selection", "--directory", "--title="+title)
		} else { // kdialog
			cmd = exec.Command("kdialog", "--getexistingdirectory", ".", "--title", title)
		}
		out, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}
		logWarning("GUI folder picker failed. Falling back to terminal.")
	}

	fmt.Printf("%s> Enter the full path to the directory: %s", colorBold, colorReset)
	path := readUserInput()
	if path == "" {
		return "", nil
	}
	// Expand tilde
	if strings.HasPrefix(path, "~/") {
		homeDir, _ := os.UserHomeDir()
		path = filepath.Join(homeDir, path[2:])
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("invalid or non-existent directory")
	}
	return path, nil
}

// selectFile prompts for a file, using GUI if available.
func selectFile(title, filter string) (string, error) {
	if guiFilePicker != "" {
		var cmd *exec.Cmd
		if guiFilePicker == "zenity" {
			cmd = exec.Command("zenity", "--file-selection", "--title="+title, "--file-filter="+filter)
		} else { // kdialog
			cmd = exec.Command("kdialog", "--getopenfilename", ".", filter, "--title", title)
		}
		out, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}
		logWarning("GUI file picker failed. Falling back to terminal.")
	}
	fmt.Printf("%s> Enter the full path to the backup file (.tar): %s", colorBold, colorReset)
	path := readUserInput()
	if path == "" {
		return "", nil
	}
	if strings.HasPrefix(path, "~/") {
		homeDir, _ := os.UserHomeDir()
		path = filepath.Join(homeDir, path[2:])
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("file not found")
	}
	return path, nil
}

// selectItem prompts the user to select an item from a list by number.
// Returns 0 if the user enters a blank line.
func selectItem(prompt string, max int) int {
	for {
		fmt.Printf("%s> %s: %s", colorBold, prompt, colorReset)
		input := readUserInput()
		if input == "" {
			return 0
		}
		choice, err := strconv.Atoi(input)
		if err == nil && choice > 0 && choice <= max {
			return choice
		}
		logWarning(fmt.Sprintf("Invalid input. Please enter a number between 1 and %d.", max))
	}
}

// runCommand executes a command and returns its output or an error.
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command '%s %s' failed: %v\nOutput: %s", name, strings.Join(args, " "), err, string(output))
	}
	return string(output), nil
}

func readUserInput() string {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

func confirmAction() bool {
	return strings.ToLower(readUserInput()) == "y"
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func clearScreen() {
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

func logError(msg string) {
	fmt.Printf("%s%s❌ ERROR: %s%s\n", colorBold, colorRed, msg, colorReset)
}

func logWarning(msg string) {
	fmt.Printf("%s%s⚠️  WARN: %s%s\n", colorBold, colorYellow, msg, colorReset)
}

func logInfo(msg string) {
	fmt.Printf("%s%sℹ️  INFO: %s%s\n", colorBold, colorCyan, msg, colorReset)
}

func logSuccess(msg string) {
	fmt.Printf("%s%s%s%s\n", colorBold, colorGreen, msg, colorReset)
}
