package recipes

import (
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

// downloadInstaller downloads a remote URL to a local destination file
func downloadInstaller(url, destPath string) error {
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

// refreshProcessPath updates the in-memory PATH environment variable from the Windows registry
func refreshProcessPath() {
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

	queryReg := func(key, value string) string {
		cmd := exec.Command("reg", "query", key, "/v", value)
		out, err := cmd.Output()
		if err != nil {
			return ""
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

	// 1. Machine Path
	if machinePath := queryReg(`HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment`, "Path"); machinePath != "" {
		for _, p := range filepath.SplitList(expandWinEnv(machinePath)) {
			if p != "" {
				newPaths = append(newPaths, p)
			}
		}
	}

	// 2. User Path
	if userPath := queryReg(`HKCU\Environment`, "Path"); userPath != "" {
		for _, p := range filepath.SplitList(expandWinEnv(userPath)) {
			if p != "" {
				newPaths = append(newPaths, p)
			}
		}
	}

	// 3. Common known directories
	home, _ := os.UserHomeDir()
	knownDirs := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "nodejs"),
		filepath.Join(os.Getenv("ProgramFiles"), "Go", "bin"),
		filepath.Join(os.Getenv("ProgramFiles"), "Git", "cmd"),
		filepath.Join(os.Getenv("ProgramFiles"), "Git", "bin"),
		filepath.Join(os.Getenv("ProgramFiles"), "CMake", "bin"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Python", "Python312"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Python", "Python312", "Scripts"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Python", "Python311"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Python", "Python310"),
		filepath.Join(home, ".cargo", "bin"),
		filepath.Join(home, "go", "bin"),
		filepath.Join(os.Getenv("APPDATA"), "npm"),
		`C:\MinGW\bin`,
		`C:\msys64\ucrt64\bin`,
		`C:\msys64\mingw64\bin`,
	}

	for _, dir := range knownDirs {
		if dir != "" {
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				newPaths = append(newPaths, dir)
			}
		}
	}

	// Deduplicate
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

// EnsureNode checks if Node.js and npm are installed; installs them if missing
func EnsureNode() error {
	refreshProcessPath()
	_, nodeErr := exec.LookPath("node")
	_, npmErr := exec.LookPath("npm")

	if nodeErr == nil && npmErr == nil {
		return nil
	}

	fmt.Println()
	fmt.Println("[Prerequisite] Node.js & npm are required for this recipe.")
	fmt.Println("Installing Node.js (LTS) automatically...")

	if runtime.GOOS == "windows" {
		// Try winget
		if _, err := exec.LookPath("winget"); err == nil {
			fmt.Println("  -> Installing via winget (OpenJS.NodeJS.LTS)...")
			cmd := exec.Command("winget", "install", "--id", "OpenJS.NodeJS.LTS", "-e", "--silent", "--accept-source-agreements", "--accept-package-agreements")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err == nil {
				refreshProcessPath()
				if _, err := exec.LookPath("npm"); err == nil {
					fmt.Println("  -> [OK] Node.js installed successfully!")
					return nil
				}
			}
		}

		// Fallback MSI
		fmt.Println("  -> Downloading official Node.js installer...")
		tempMsi := filepath.Join(os.TempDir(), "node_lts.msi")
		defer os.Remove(tempMsi)

		if err := downloadInstaller("https://nodejs.org/dist/v22.14.0/node-v22.14.0-x64.msi", tempMsi); err == nil {
			fmt.Println("  -> Installing Node.js silently...")
			cmd := exec.Command(getMsiExecPath(), "/i", tempMsi, "/quiet", "/qn", "/norestart")
			if err := cmd.Run(); err == nil {
				refreshProcessPath()
				if _, err := exec.LookPath("npm"); err == nil {
					fmt.Println("  -> [OK] Node.js installed successfully!")
					return nil
				}
			}
		}
	}

	return fmt.Errorf("Node.js could not be installed automatically. Please install it from https://nodejs.org")
}

// EnsurePython checks if Python is installed; installs it if missing
func EnsurePython() error {
	refreshProcessPath()
	for _, cmdName := range []string{"python", "python3", "py"} {
		if _, err := exec.LookPath(cmdName); err == nil {
			return nil
		}
	}

	fmt.Println()
	fmt.Println("[Prerequisite] Python is required for this recipe.")
	fmt.Println("Installing Python 3 automatically...")

	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("winget"); err == nil {
			fmt.Println("  -> Installing via winget (Python.Python.3.12)...")
			cmd := exec.Command("winget", "install", "--id", "Python.Python.3.12", "-e", "--silent", "--accept-source-agreements", "--accept-package-agreements")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err == nil {
				refreshProcessPath()
				for _, cmdName := range []string{"python", "python3", "py"} {
					if _, err := exec.LookPath(cmdName); err == nil {
						fmt.Println("  -> [OK] Python installed successfully!")
						return nil
					}
				}
			}
		}
	}

	return fmt.Errorf("Python is not installed. Please install Python 3 from https://www.python.org/downloads/")
}

// EnsureGo checks if Go is installed; installs it if missing
func EnsureGo() error {
	refreshProcessPath()
	if _, err := exec.LookPath("go"); err == nil {
		return nil
	}

	fmt.Println()
	fmt.Println("[Prerequisite] Go is required for this recipe.")
	fmt.Println("Installing Go automatically...")

	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("winget"); err == nil {
			fmt.Println("  -> Installing via winget (GoLang.Go)...")
			cmd := exec.Command("winget", "install", "--id", "GoLang.Go", "-e", "--silent", "--accept-source-agreements", "--accept-package-agreements")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err == nil {
				refreshProcessPath()
				if _, err := exec.LookPath("go"); err == nil {
					fmt.Println("  -> [OK] Go installed successfully!")
					return nil
				}
			}
		}

		// Fallback MSI
		fmt.Println("  -> Downloading official Go installer...")
		tempMsi := filepath.Join(os.TempDir(), "go_installer.msi")
		defer os.Remove(tempMsi)

		if err := downloadInstaller("https://go.dev/dl/go1.24.0.windows-amd64.msi", tempMsi); err == nil {
			fmt.Println("  -> Installing Go silently...")
			cmd := exec.Command(getMsiExecPath(), "/i", tempMsi, "/quiet", "/qn", "/norestart")
			if err := cmd.Run(); err == nil {
				refreshProcessPath()
				if _, err := exec.LookPath("go"); err == nil {
					fmt.Println("  -> [OK] Go installed successfully!")
					return nil
				}
			}
		}
	}

	return fmt.Errorf("Go is not installed. Please install Go from https://go.dev/dl/")
}

// EnsureRust checks if Rust/Cargo is installed; installs it if missing
func EnsureRust() error {
	refreshProcessPath()
	if _, err := exec.LookPath("cargo"); err == nil {
		return nil
	}

	fmt.Println()
	fmt.Println("[Prerequisite] Rust (cargo) is required for this recipe.")
	fmt.Println("Installing Rust automatically via Rustup...")

	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("winget"); err == nil {
			fmt.Println("  -> Installing via winget (Rustlang.Rustup)...")
			cmd := exec.Command("winget", "install", "--id", "Rustlang.Rustup", "-e", "--silent", "--accept-source-agreements", "--accept-package-agreements")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err == nil {
				refreshProcessPath()
				if _, err := exec.LookPath("cargo"); err == nil {
					fmt.Println("  -> [OK] Rust installed successfully!")
					return nil
				}
			}
		}

		// Fallback: rustup-init
		fmt.Println("  -> Downloading rustup-init...")
		tempExe := filepath.Join(os.TempDir(), "rustup-init.exe")
		defer os.Remove(tempExe)

		if err := downloadInstaller("https://win.rustup.rs/x86_64", tempExe); err == nil {
			fmt.Println("  -> Running rustup-init silently...")
			cmd := exec.Command(tempExe, "-y", "--default-toolchain", "stable")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err == nil {
				refreshProcessPath()
				if _, err := exec.LookPath("cargo"); err == nil {
					fmt.Println("  -> [OK] Rust installed successfully!")
					return nil
				}
			}
		}
	}

	return fmt.Errorf("Rust is not installed. Please install Rust from https://rustup.rs")
}

// EnsureCpp checks if CMake and a C++ compiler are installed
func EnsureCpp() error {
	refreshProcessPath()

	// Check CMake
	if _, err := exec.LookPath("cmake"); err != nil {
		fmt.Println()
		fmt.Println("[Notice] CMake is recommended for C++ projects.")
		if runtime.GOOS == "windows" {
			if _, err := exec.LookPath("winget"); err == nil {
				fmt.Println("  -> Installing CMake via winget (Kitware.CMake)...")
				cmd := exec.Command("winget", "install", "--id", "Kitware.CMake", "-e", "--silent", "--accept-source-agreements", "--accept-package-agreements")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				_ = cmd.Run()
				refreshProcessPath()
			}
		}
	}

	// Check C++ compiler
	hasCompiler := false
	for _, comp := range []string{"g++", "clang++", "cl"} {
		if _, err := exec.LookPath(comp); err == nil {
			hasCompiler = true
			break
		}
	}

	if !hasCompiler {
		fmt.Println()
		fmt.Println("[Notice] No C++ compiler (g++, clang++, or MSVC cl.exe) was detected in PATH.")
		fmt.Println("  You can still scaffold the C++ project layout.")
		fmt.Println("  To compile, install MinGW, LLVM Clang, or Visual Studio C++ Build Tools.")
	}

	return nil
}
