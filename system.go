package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// downloadFile downloads a remote URL to a local destination file
func downloadFile(url, destPath string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "lazy-cli/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with HTTP %s", resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// expandWinEnv expands %VARIABLE% patterns on Windows
func expandWinEnv(s string) string {
	for {
		start := strings.Index(s, "%")
		if start == -1 {
			break
		}
		end := strings.Index(s[start+1:], "%")
		if end == -1 {
			break
		}
		end = start + 1 + end
		varName := s[start+1 : end]
		val := os.Getenv(varName)
		s = s[:start] + val + s[end+1:]
	}
	return s
}

// getMsiExecPath locates msiexec.exe reliably on Windows
func getMsiExecPath() string {
	if p, err := exec.LookPath("msiexec"); err == nil {
		return p
	}
	if p, err := exec.LookPath("msiexec.exe"); err == nil {
		return p
	}

	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		systemRoot = os.Getenv("windir")
	}
	if systemRoot == "" {
		systemRoot = `C:\Windows`
	}

	candidates := []string{
		filepath.Join(systemRoot, "System32", "msiexec.exe"),
		filepath.Join(systemRoot, "SysWOW64", "msiexec.exe"),
		`C:\Windows\System32\msiexec.exe`,
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	return "msiexec.exe"
}

// refreshWindowsPath re-reads system and user PATH from the Windows registry
// and updates the current process environment so newly installed tools are found immediately.
func refreshWindowsPath() {
	if runtime.GOOS != "windows" {
		return
	}

	var newPaths []string

	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		systemRoot = os.Getenv("windir")
	}
	if systemRoot == "" {
		systemRoot = `C:\Windows`
	}

	// Always ensure core Windows system folders are in PATH
	newPaths = append(newPaths,
		filepath.Join(systemRoot, "System32"),
		systemRoot,
		filepath.Join(systemRoot, "System32", "Wbem"),
		filepath.Join(systemRoot, "System32", "WindowsPowerShell", "v1.0"),
	)

	// Helper to extract Path value using 'reg query'
	queryReg := func(key, value string) string {
		cmd := exec.Command(getMsiExecPath()[:strings.LastIndex(getMsiExecPath(), `\`)+1]+"reg.exe", "query", key, "/v", value)
		out, err := cmd.Output()
		if err != nil {
			// Fallback to plain reg query
			cmd = exec.Command("reg", "query", key, "/v", value)
			out, err = cmd.Output()
			if err != nil {
				return ""
			}
		}
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(line, value) {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					return strings.Join(parts[2:], " ")
				}
			}
		}
		return ""
	}

	// 1. Read Machine Path (HKLM)
	machinePath := queryReg(`HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment`, "Path")
	if machinePath != "" {
		for _, p := range filepath.SplitList(expandWinEnv(machinePath)) {
			if p != "" {
				newPaths = append(newPaths, p)
			}
		}
	}

	// 2. Read User Path (HKCU)
	userPath := queryReg(`HKCU\Environment`, "Path")
	if userPath != "" {
		for _, p := range filepath.SplitList(expandWinEnv(userPath)) {
			if p != "" {
				newPaths = append(newPaths, p)
			}
		}
	}

	// 3. Guarantee standard installation directories are included
	standardDirs := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Git", "cmd"),
		filepath.Join(os.Getenv("ProgramFiles"), "Git", "bin"),
		filepath.Join(os.Getenv("ProgramFiles"), "GitHub CLI"),
		filepath.Join(os.Getenv("ProgramFiles"), "nodejs"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "WinGet", "Links"),
		filepath.Join(os.Getenv("APPDATA"), "npm"),
	}

	for _, dir := range standardDirs {
		if dir != "" {
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				newPaths = append(newPaths, dir)
			}
		}
	}

	// Deduplicate and set environment
	seen := make(map[string]bool)
	var finalPaths []string
	for _, p := range newPaths {
		clean := filepath.Clean(strings.TrimSpace(p))
		if clean != "" && !seen[strings.ToLower(clean)] {
			seen[strings.ToLower(clean)] = true
			finalPaths = append(finalPaths, clean)
		}
	}

	if len(finalPaths) > 0 {
		_ = os.Setenv("PATH", strings.Join(finalPaths, string(os.PathListSeparator)))
	}
}

// isOneDrivePath checks if a given directory path is located inside a OneDrive folder
func isOneDrivePath(path string) bool {
	lower := strings.ToLower(filepath.Clean(path))
	return strings.Contains(lower, "onedrive")
}

// ensureGit verifies Git is installed; if not, attempts automatic silent installation
func ensureGit() (string, error) {
	refreshWindowsPath()
	if path, err := exec.LookPath("git"); err == nil {
		return path, nil
	}

	fmt.Println()
	fmt.Println("[Notice] Git is not installed on this device.")
	fmt.Println("Installing Git for Windows automatically...")

	if runtime.GOOS == "windows" {
		// 1. Try winget first
		if _, err := exec.LookPath("winget"); err == nil {
			fmt.Println("  -> Attempting install via winget (Git.Git)...")
			cmd := exec.Command("winget", "install", "--id", "Git.Git", "-e", "--silent", "--accept-source-agreements", "--accept-package-agreements")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err == nil {
				refreshWindowsPath()
				if path, err := exec.LookPath("git"); err == nil {
					fmt.Println("  -> Git installed successfully!")
					return path, nil
				}
			}
		}

		// 2. Fallback: Download official standalone installer
		fmt.Println("  -> Downloading Git standalone installer...")
		tempInstaller := filepath.Join(os.TempDir(), "git_installer.exe")
		defer os.Remove(tempInstaller)

		gitURL := "https://github.com/git-for-windows/git/releases/download/v2.48.1.windows.1/Git-2.48.1-64-bit.exe"
		if err := downloadFile(gitURL, tempInstaller); err != nil {
			return "", fmt.Errorf("failed to download Git installer: %w (please install Git manually from https://git-scm.com)", err)
		}

		fmt.Println("  -> Installing Git silently (this may take a minute)...")
		cmd := exec.Command(tempInstaller, "/VERYSILENT", "/NORESTART", "/NOCANCEL", "/SP-", "/CLOSEAPPLICATIONS", "/RESTARTAPPLICATIONS")
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("Git installer failed: %w", err)
		}

		refreshWindowsPath()
		if path, err := exec.LookPath("git"); err == nil {
			fmt.Println("  -> Git installed successfully!")
			return path, nil
		}
	}

	return "", fmt.Errorf("Git could not be installed automatically. Please install Git manually from https://git-scm.com")
}

// ensureGitIdentity verifies user.name and user.email exist to prevent 'Author identity unknown' errors
func ensureGitIdentity() error {
	nameCmd := exec.Command("git", "config", "--global", "user.name")
	nameOut, _ := nameCmd.Output()
	userName := strings.TrimSpace(string(nameOut))

	emailCmd := exec.Command("git", "config", "--global", "user.email")
	emailOut, _ := emailCmd.Output()
	userEmail := strings.TrimSpace(string(emailOut))

	if userName != "" && userEmail != "" {
		return nil
	}

	fmt.Println()
	fmt.Println("[Notice] Git author identity is not configured yet.")

	// Try to get username from gh CLI if available
	if isGHInstalled() {
		userCmd := exec.Command("gh", "api", "user", "--jq", ".login")
		if out, err := userCmd.Output(); err == nil && len(out) > 0 {
			ghUser := strings.TrimSpace(string(out))
			if userName == "" {
				userName = ghUser
				_ = exec.Command("git", "config", "--global", "user.name", userName).Run()
			}
			if userEmail == "" {
				userEmail = ghUser + "@users.noreply.github.com"
				_ = exec.Command("git", "config", "--global", "user.email", userEmail).Run()
			}
			fmt.Printf("  -> Configured Git author automatically: %s <%s>\n", userName, userEmail)
			return nil
		}
	}

	// Prompt user for Git identity
	reader := bufio.NewReader(os.Stdin)
	if userName == "" {
		fmt.Print("Enter your Git author name (e.g., John Doe): ")
		input, _ := reader.ReadString('\n')
		userName = strings.TrimSpace(input)
		if userName == "" {
			userName = "Lazy Developer"
		}
		_ = exec.Command("git", "config", "--global", "user.name", userName).Run()
	}

	if userEmail == "" {
		fmt.Print("Enter your Git author email (e.g., john@example.com): ")
		input, _ := reader.ReadString('\n')
		userEmail = strings.TrimSpace(input)
		if userEmail == "" {
			userEmail = "developer@lazy.local"
		}
		_ = exec.Command("git", "config", "--global", "user.email", userEmail).Run()
	}

	fmt.Printf("  -> Git author set to: %s <%s>\n", userName, userEmail)
	return nil
}

// ensureNode verifies that Node.js and npm are installed
func ensureNode() error {
	refreshWindowsPath()
	_, nodeErr := exec.LookPath("node")
	_, npmErr := exec.LookPath("npm")

	if nodeErr == nil && npmErr == nil {
		return nil
	}

	fmt.Println()
	fmt.Println("[Notice] Node.js and npm are required for this template but were not found.")
	fmt.Println("Installing Node.js (LTS) automatically...")

	if runtime.GOOS == "windows" {
		// 1. Try winget first
		if _, err := exec.LookPath("winget"); err == nil {
			fmt.Println("  -> Attempting install via winget (OpenJS.NodeJS.LTS)...")
			cmd := exec.Command("winget", "install", "--id", "OpenJS.NodeJS.LTS", "-e", "--silent", "--accept-source-agreements", "--accept-package-agreements")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err == nil {
				refreshWindowsPath()
				if _, err := exec.LookPath("npm"); err == nil {
					fmt.Println("  -> Node.js installed successfully!")
					return nil
				}
			}
		}

		// 2. Fallback: Download official LTS MSI installer
		fmt.Println("  -> Downloading official Node.js installer...")
		tempMsi := filepath.Join(os.TempDir(), "node_installer.msi")
		defer os.Remove(tempMsi)

		nodeURL := "https://nodejs.org/dist/v22.14.0/node-v22.14.0-x64.msi"
		if err := downloadFile(nodeURL, tempMsi); err != nil {
			return fmt.Errorf("failed to download Node.js installer: %w (please install from https://nodejs.org)", err)
		}

		fmt.Println("  -> Installing Node.js silently...")
		cmd := exec.Command(getMsiExecPath(), "/i", tempMsi, "/quiet", "/qn", "/norestart")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("Node.js installation failed: %w", err)
		}

		refreshWindowsPath()
		if _, err := exec.LookPath("npm"); err == nil {
			fmt.Println("  -> Node.js installed successfully!")
			return nil
		}
	}

	return fmt.Errorf("Node.js is not installed. Please install it from https://nodejs.org and re-run this command")
}

// openInEditor opens project in editor if available or prints friendly guidance
func openInEditor(projectPath string) {
	if _, err := exec.LookPath("code"); err == nil {
		cmd := exec.Command("code", ".")
		cmd.Dir = projectPath
		_ = cmd.Run()
	} else {
		fmt.Println("Tip: Visual Studio Code ('code') was not found in PATH.")
		fmt.Printf("     You can open your project manually at: %s\n", projectPath)
	}
}
