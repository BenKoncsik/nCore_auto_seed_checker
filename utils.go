package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// CheckAndInstallLynx checks if lynx is installed, and if not, attempts to install it.
func CheckAndInstallLynx() error {
	path, err := exec.LookPath("lynx")
	if err == nil {
		fmt.Printf("Lynx found at: %s\n", path)
		return nil
	}

	fmt.Println("Lynx not found. Attempting to install...")

	switch runtime.GOOS {
	case "darwin":
		return installMacOS()
	case "linux":
		return installLinux()
	case "windows":
		return fmt.Errorf("automatic installation on Windows is not supported. Please install Lynx manually (e.g., check https://lynx.invisible-island.net/ or use Chocolatey/WSL)")
	default:
		return fmt.Errorf("unsupported platform: %s. Please install Lynx manually", runtime.GOOS)
	}
}

func installMacOS() error {
	// Check if brew exists
	_, err := exec.LookPath("brew")
	if err != nil {
		return fmt.Errorf("brew not found, cannot install lynx automatically. Please install Homebrew or install lynx manually")
	}

	fmt.Println("Running: brew install lynx")
	cmd := exec.Command("brew", "install", "lynx")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install lynx: %w. Please install it manually", err)
	}
	fmt.Println("Lynx installed successfully.")
	return nil
}

func installLinux() error {
	// Map of package manager name to install arguments
	// Order matters: we check simpler/more common ones first or check specifically
	pkgManagers := []struct {
		name string
		args []string
	}{
		{"apt-get", []string{"install", "-y", "lynx"}},
		{"apk", []string{"add", "lynx"}},
		{"dnf", []string{"install", "-y", "lynx"}},
		{"yum", []string{"install", "-y", "lynx"}},
		{"pacman", []string{"-S", "--noconfirm", "lynx"}},
	}

	for _, pm := range pkgManagers {
		_, err := exec.LookPath(pm.name)
		if err == nil {
			fmt.Printf("Found package manager: %s. Attempting validation...\n", pm.name)

			// Construct command
			// Note: This might require sudo. We try running it directly.
			// If the user is not root, this will fail.
			// We can try to be smart and prepend "sudo" if we are not root, but
			// interactive password entry is tricky.
			// We'll rely on the command failing and the user reading the error.

			fmt.Printf("Running: %s %v\n", pm.name, pm.args)
			cmd := exec.Command(pm.name, pm.args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				// Try with sudo if strictly "apt-get", "dnf", "yum", "pacman"?
				// apk usually runs as root in container or needs sudo.
				return fmt.Errorf("failed to install lynx using %s: %w. Try running with sudo or install manually", pm.name, err)
			}

			fmt.Println("Lynx installed successfully.")
			return nil
		}
	}

	return fmt.Errorf("no supported package manager found (apt-get, apk, dnf, yum, pacman). Please install lynx manually")
}
