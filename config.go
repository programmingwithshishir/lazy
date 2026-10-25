package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Config struct {
	ProjectsPath      string `json:"projects_path"`
	DefaultVisibility string `json:"default_visibility"` // "public" or "private"
}

// getConfigPath returns the path to ~/.lazy/config.json
func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine user home directory: %w", err)
	}
	configDir := filepath.Join(home, ".lazy")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("unable to create config directory %s: %w", configDir, err)
	}
	return filepath.Join(configDir, "config.json"), nil
}

// LoadConfig reads the configuration file from disk
func LoadConfig() (*Config, error) {
	path, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config format: %w", err)
	}

	// Validate required fields
	if strings.TrimSpace(cfg.ProjectsPath) == "" {
		return nil, fmt.Errorf("configuration is missing projects path")
	}

	if cfg.DefaultVisibility == "" {
		cfg.DefaultVisibility = "public"
	}

	return &cfg, nil
}

// SaveConfig saves the configuration to ~/.lazy/config.json
func SaveConfig(cfg *Config) error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode configuration: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// RunInteractiveSetup prompts the user for configuration settings
func RunInteractiveSetup() (*Config, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("             LAZY CONFIGURATION WIZARD            ")
	fmt.Println("==================================================")
	fmt.Println()

	// 1. Projects Directory (generic default, no usernames leaked)
	defaultDir := `C:\projects`
	if runtime.GOOS != "windows" {
		home, _ := os.UserHomeDir()
		defaultDir = filepath.Join(home, "projects")
	}

	var projectsDir string
	for {
		fmt.Printf("[1/4] Enter projects root directory [%s]: ", defaultDir)
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		input = strings.TrimSpace(input)
		input = strings.Trim(input, "\"'")

		if input == "" {
			projectsDir = defaultDir
		} else {
			if strings.HasPrefix(input, "~") {
				home, _ := os.UserHomeDir()
				input = filepath.Join(home, input[1:])
			}
			projectsDir = filepath.Clean(input)
		}

		if isOneDrivePath(projectsDir) {
			fmt.Println()
			fmt.Println("  [Warning] The path you selected is inside a OneDrive sync folder:")
			fmt.Printf("            %s\n", projectsDir)
			fmt.Println("            OneDrive background sync can lock .git and node_modules files.")
			fmt.Println("            Recommendation: use C:\\projects or a path outside OneDrive.")
			fmt.Print("            Continue with this path anyway? (y/N): ")
			ans, _ := reader.ReadString('\n')
			ans = strings.ToLower(strings.TrimSpace(ans))
			if ans != "y" && ans != "yes" {
				continue
			}
		}

		if err := os.MkdirAll(projectsDir, 0755); err != nil {
			fmt.Printf("Error: Unable to create '%s': %v\nPlease choose a different path.\n", projectsDir, err)
			continue
		}
		fmt.Printf("  -> Projects will be stored in: %s\n\n", projectsDir)
		break
	}

	// 2. Default Visibility
	var visibility string
	for {
		fmt.Printf("[2/4] Default GitHub repo visibility (public/private) [public]: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		input = strings.ToLower(strings.TrimSpace(input))

		if input == "" || input == "public" {
			visibility = "public"
			break
		} else if input == "private" {
			visibility = "private"
			break
		} else {
			fmt.Println("Invalid choice. Please enter 'public' or 'private'.")
		}
	}
	fmt.Printf("  -> Default repository visibility: %s\n\n", visibility)

	// 3. Git Installation & Author Identity
	fmt.Println("[3/4] Checking Git...")
	if _, err := ensureGit(); err != nil {
		fmt.Printf("  -> Git error: %v\n", err)
	} else {
		_ = ensureGitIdentity()
		fmt.Println("  -> [OK] Git is configured and ready.")
	}
	fmt.Println()

	// 4. GitHub CLI Check, Auto-Install & Authentication
	fmt.Println("[4/4] Checking GitHub CLI ('gh')...")
	ghPath, err := ensureGH()
	if err != nil {
		fmt.Printf("  -> Could not automatically install GitHub CLI: %v\n", err)
		fmt.Println("     You can install it manually from https://cli.github.com")
	} else {
		loggedIn, _ := checkGHAuth(ghPath)
		if loggedIn {
			fmt.Println("  -> [OK] GitHub CLI is installed and authenticated!")
		} else {
			fmt.Println("  -> GitHub CLI is installed, but not logged in yet.")
			fmt.Print("     Would you like to log in now with your browser? (Y/n): ")
			ans, _ := reader.ReadString('\n')
			ans = strings.ToLower(strings.TrimSpace(ans))
			if ans == "" || ans == "y" || ans == "yes" {
				_ = ensureGHAuthenticated(ghPath)
			} else {
				fmt.Println("     You can log in later by running: gh auth login")
			}
		}
	}

	cfg := &Config{
		ProjectsPath:      projectsDir,
		DefaultVisibility: visibility,
	}

	if err := SaveConfig(cfg); err != nil {
		return nil, fmt.Errorf("failed to save configuration: %w", err)
	}

	cfgPath, _ := getConfigPath()
	fmt.Println()
	fmt.Println("Configuration successfully saved to:", cfgPath)
	fmt.Println("==================================================")
	fmt.Println()

	return cfg, nil
}

// EnsureConfig checks if valid config exists. If not, forces interactive setup.
func EnsureConfig() (*Config, error) {
	cfg, err := LoadConfig()
	if err == nil && cfg != nil {
		return cfg, nil
	}

	return RunInteractiveSetup()
}

// DisplayConfig prints current configuration settings
func DisplayConfig() {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Println("No valid configuration found. Run 'lazy config' to set up.")
		return
	}

	cfgPath, _ := getConfigPath()
	ghStatus := "Not installed"
	ghPath := getGHExecutable()
	if ghPath != "" {
		loggedIn, statusMsg := checkGHAuth(ghPath)
		if loggedIn {
			ghStatus = "Installed & Authenticated"
		} else {
			ghStatus = statusMsg
		}
	}

	gitStatus := "Not installed"
	if _, err := exec.LookPath("git"); err == nil {
		gitStatus = "Installed"
	}

	nodeStatus := "Not installed"
	if _, err := exec.LookPath("node"); err == nil {
		nodeStatus = "Installed"
	}

	fmt.Println()
	fmt.Println("Current Lazy Configuration & Environment:")
	fmt.Println("  Config File:        ", cfgPath)
	fmt.Println("  Projects Directory: ", cfg.ProjectsPath)
	fmt.Println("  Default Visibility: ", cfg.DefaultVisibility)
	fmt.Println("  Git Status:         ", gitStatus)
	fmt.Println("  Node.js Status:     ", nodeStatus)
	fmt.Println("  GitHub CLI Status:  ", ghStatus)
	fmt.Println()
}
