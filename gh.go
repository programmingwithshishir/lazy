package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// getGHExecutable searches for the gh binary in PATH and common installation directories
func getGHExecutable() string {
	refreshWindowsPath()

	// 1. Check current system PATH
	if path, err := exec.LookPath("gh"); err == nil {
		return path
	}

	// 2. On Windows, check standard installation directories
	if runtime.GOOS == "windows" {
		candidates := []string{
			filepath.Join(os.Getenv("ProgramFiles"), "GitHub CLI", "gh.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "GitHub CLI", "gh.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "GitHub CLI", "gh.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "WinGet", "Links", "gh.exe"),
			filepath.Join(os.Getenv("USERPROFILE"), "scoop", "shims", "gh.exe"),
			`C:\ProgramData\chocolatey\bin\gh.exe`,
		}

		for _, candidate := range candidates {
			if candidate == "" {
				continue
			}
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				dir := filepath.Dir(candidate)
				currentPath := os.Getenv("PATH")
				if !strings.Contains(currentPath, dir) {
					_ = os.Setenv("PATH", dir+string(os.PathListSeparator)+currentPath)
				}
				return candidate
			}
		}
	}

	return ""
}

// isGHInstalled returns true if gh can be located
func isGHInstalled() bool {
	return getGHExecutable() != ""
}

// checkGHAuth checks if GitHub CLI is authenticated
func checkGHAuth(ghPath string) (bool, string) {
	if ghPath == "" {
		ghPath = getGHExecutable()
	}
	if ghPath == "" {
		return false, "GitHub CLI is not installed"
	}

	cmd := exec.Command(ghPath, "auth", "status")
	output, err := cmd.CombinedOutput()
	outStr := string(output)
	if err == nil || strings.Contains(outStr, "Logged in to github.com") {
		return true, "Logged in to GitHub"
	}
	return false, "GitHub CLI installed but not logged in"
}

// checkGHRepoExists checks if a repository with the given name already exists on the user's GitHub account
func checkGHRepoExists(ghPath, repoName string) (bool, error) {
	if ghPath == "" {
		ghPath = getGHExecutable()
	}
	if ghPath == "" {
		return false, nil
	}

	// 1. Get authenticated username
	userCmd := exec.Command(ghPath, "api", "user", "--jq", ".login")
	userOut, err := userCmd.Output()
	if err != nil {
		return false, err
	}
	username := strings.TrimSpace(string(userOut))
	if username == "" {
		return false, nil
	}

	// 2. Check if repo exists under user's account
	checkCmd := exec.Command(ghPath, "api", fmt.Sprintf("repos/%s/%s", username, repoName), "--silent")
	if err := checkCmd.Run(); err == nil {
		return true, nil // Exists!
	}

	return false, nil
}

// installGH installs GitHub CLI using package managers or direct MSI fallback
func installGH() error {
	switch runtime.GOOS {
	case "windows":
		// 1. Try winget first
		if _, err := exec.LookPath("winget"); err == nil {
			fmt.Println("Running: winget install --id GitHub.cli ...")
			cmd := exec.Command("winget", "install", "--id", "GitHub.cli", "--silent", "--accept-source-agreements", "--accept-package-agreements")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err == nil {
				refreshWindowsPath()
				if isGHInstalled() {
					return nil
				}
			}
			fmt.Println("winget installation encountered an issue, trying direct installer...")
		}

		// 2. Try scoop
		if _, err := exec.LookPath("scoop"); err == nil {
			fmt.Println("Running: scoop install gh ...")
			cmd := exec.Command("scoop", "install", "gh")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err == nil {
				refreshWindowsPath()
				if isGHInstalled() {
					return nil
				}
			}
		}

		// 3. Try chocolatey
		if _, err := exec.LookPath("choco"); err == nil {
			fmt.Println("Running: choco install gh -y ...")
			cmd := exec.Command("choco", "install", "gh", "-y")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err == nil {
				refreshWindowsPath()
				if isGHInstalled() {
					return nil
				}
			}
		}

		// 4. Guaranteed Direct MSI Fallback (works on any Windows 10+ system)
		fmt.Println("Downloading official GitHub CLI installer...")
		tempMsi := filepath.Join(os.TempDir(), "gh_installer.msi")
		defer os.Remove(tempMsi)

		ghURL := "https://github.com/cli/cli/releases/download/v2.67.0/gh_2.67.0_windows_amd64.msi"
		if err := downloadFile(ghURL, tempMsi); err != nil {
			return fmt.Errorf("failed to download GitHub CLI installer: %w (visit https://cli.github.com)", err)
		}

		fmt.Println("Installing GitHub CLI silently via msiexec...")
		msiCmd := exec.Command(getMsiExecPath(), "/i", tempMsi, "/quiet", "/qn", "/norestart")
		if err := msiCmd.Run(); err != nil {
			return fmt.Errorf("msiexec installation failed: %w", err)
		}

		refreshWindowsPath()
		if isGHInstalled() {
			return nil
		}

		return fmt.Errorf("GitHub CLI installed but executable could not be verified")

	case "darwin":
		if _, err := exec.LookPath("brew"); err != nil {
			return fmt.Errorf("Homebrew ('brew') not found. Please install gh from https://cli.github.com")
		}
		fmt.Println("Running: brew install gh ...")
		cmd := exec.Command("brew", "install", "gh")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()

	default: // Linux
		if _, err := exec.LookPath("apt-get"); err == nil {
			fmt.Println("Running: sudo apt-get update && sudo apt-get install -y gh ...")
			cmd := exec.Command("sh", "-c", "sudo apt-get update && sudo apt-get install -y gh")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			return cmd.Run()
		}
		if _, err := exec.LookPath("dnf"); err == nil {
			fmt.Println("Running: sudo dnf install -y gh ...")
			cmd := exec.Command("sudo", "dnf", "install", "-y", "gh")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			return cmd.Run()
		}
		if _, err := exec.LookPath("pacman"); err == nil {
			fmt.Println("Running: sudo pacman -S --noconfirm github-cli ...")
			cmd := exec.Command("sudo", "pacman", "-S", "--noconfirm", "github-cli")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			return cmd.Run()
		}
		return fmt.Errorf("no supported Linux package manager found")
	}
}

// ensureGH locates gh or installs it automatically if missing
func ensureGH() (string, error) {
	ghPath := getGHExecutable()
	if ghPath != "" {
		return ghPath, nil
	}

	fmt.Println()
	fmt.Println("[Notice] GitHub CLI ('gh') is not installed on this device.")
	fmt.Println("Installing GitHub CLI automatically now...")
	if err := installGH(); err != nil {
		return "", fmt.Errorf("automatic installation failed: %w", err)
	}

	ghPath = getGHExecutable()
	if ghPath == "" {
		return "", fmt.Errorf("installation completed, but binary could not be found. Try restarting your terminal")
	}

	fmt.Println("GitHub CLI installed successfully!")
	return ghPath, nil
}

// ensureGHAuthenticated ensures the user is logged into GitHub CLI
func ensureGHAuthenticated(ghPath string) error {
	if ghPath == "" {
		ghPath = getGHExecutable()
	}
	loggedIn, _ := checkGHAuth(ghPath)
	if loggedIn {
		return nil
	}

	fmt.Println()
	fmt.Println("Authentication required for GitHub CLI.")
	fmt.Println("Opening GitHub login in your terminal/browser...")
	cmd := exec.Command(ghPath, "auth", "login")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
