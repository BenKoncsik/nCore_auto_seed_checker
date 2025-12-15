package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// CheckAndInstallDependencies checks for curl, which is required.
func CheckAndInstallDependencies() error {
	path, err := exec.LookPath("curl")
	if err == nil {
		fmt.Printf("Curl found at: %s\n", path)
		return nil
	}

	fmt.Println("Curl not found. Attempting to install...")

	switch runtime.GOOS {
	case "darwin":
		return installMacOS()
	case "linux":
		return installLinux()
	case "windows":
		return fmt.Errorf("automatic installation on Windows is not supported. Please install cURL manually (likely already installed on modern Windows 10/11)")
	default:
		return fmt.Errorf("unsupported platform: %s. Please install cURL manually", runtime.GOOS)
	}
}

func installMacOS() error {
	// macOS usually has curl.
	_, err := exec.LookPath("brew")
	if err != nil {
		return fmt.Errorf("brew not found, cannot install curl automatically")
	}

	fmt.Println("Running: brew install curl")
	cmd := exec.Command("brew", "install", "curl")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install curl: %w", err)
	}
	return nil
}

func installLinux() error {
	pkgManagers := []struct {
		name string
		args []string
	}{
		{"apt-get", []string{"install", "-y", "curl"}},
		{"apk", []string{"add", "curl"}},
		{"dnf", []string{"install", "-y", "curl"}},
		{"yum", []string{"install", "-y", "curl"}},
		{"pacman", []string{"-S", "--noconfirm", "curl"}},
	}

	for _, pm := range pkgManagers {
		_, err := exec.LookPath(pm.name)
		if err == nil {
			fmt.Printf("Found package manager: %s. Attempting installation...\n", pm.name)
			cmd := exec.Command(pm.name, pm.args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to install curl using %s: %w", pm.name, err)
			}
			return nil
		}
	}

	return fmt.Errorf("no supported package manager found to install curl")
}
