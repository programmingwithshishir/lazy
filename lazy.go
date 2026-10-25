package main

import (
	"bufio"
	"fmt"
	"lazy/recipes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Displays the help prompt message
func displayHelp() {
	cfg, _ := LoadConfig()
	projectsPath := "Not configured (run 'lazy config')"
	if cfg != nil && cfg.ProjectsPath != "" {
		projectsPath = cfg.ProjectsPath
	}

	fmt.Println()
	fmt.Println("Lazy - Universal Project Scaffolding CLI")
	fmt.Println("Configured Projects Directory:", projectsPath)
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  lazy create <project-name> <--template>")
	fmt.Println("  lazy list")
	fmt.Println("  lazy config            Configure or update settings")
	fmt.Println("  lazy config --show     View current settings & environment health")
	fmt.Println("  lazy --help")
	fmt.Println()
	fmt.Println("Available Recipes:")
	recipes.PrintAvailable()
	fmt.Println()
	fmt.Println("Atomic Project Creation:")
	fmt.Println("  Lazy guarantees transactional safety. If any step fails, all")
	fmt.Println("  partial files and remote repositories are automatically cleaned up.")
	fmt.Println()
}

// Lists the projects in the projects folder
func listProjects(cfg *Config) {
	if len(os.Args) > 2 {
		fmt.Println("Invalid arguments!")
		fmt.Println("Usage: lazy list")
		return
	}

	entries, err := os.ReadDir(cfg.ProjectsPath)
	if err != nil {
		fmt.Printf("Projects directory does not exist yet: %s\n", cfg.ProjectsPath)
		return
	}

	fmt.Printf("\nLocal Projects (%s):\n", cfg.ProjectsPath)
	found := false
	for _, entry := range entries {
		if entry.IsDir() {
			fmt.Println("  - " + entry.Name())
			found = true
		}
	}
	if !found {
		fmt.Println("  (No projects found)")
	}
	fmt.Println()
}

// Checks if a folder exists
func folderExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil && info.IsDir()
}

// Creates a new project with transactional atomicity
func createNewProject(cfg *Config) {
	if len(os.Args) < 4 {
		fmt.Println("\nInvalid arguments!")
		fmt.Println("Usage: lazy create <project-name> <--template>")
		fmt.Println("Available templates:")
		recipes.PrintAvailable()
		return
	}

	projectName := strings.TrimSpace(os.Args[2])
	templateFlag := strings.TrimSpace(os.Args[3])

	if projectName == "" {
		fmt.Println("Error: Project name cannot be empty.")
		return
	}

	// 1. Validate Recipe
	recipe, exists := recipes.Get(templateFlag)
	if !exists {
		fmt.Printf("\nUnknown template '%s'.\n", templateFlag)
		fmt.Println("Available templates:")
		recipes.PrintAvailable()
		return
	}

	// 2. Check Prerequisites for the Recipe
	if recipe.CheckPrerequisites != nil {
		if err := recipe.CheckPrerequisites(); err != nil {
			fmt.Printf("\nError: Prerequisite check failed for %s:\n  %v\n", recipe.Name, err)
			return
		}
	}

	// 3. Name Availability Check (Both Locally and on GitHub)
	reader := bufio.NewReader(os.Stdin)
	var localProjectPath string

	for {
		// Local folder collision check
		localProjectPath = filepath.Join(cfg.ProjectsPath, projectName)
		if folderExists(localProjectPath) {
			fmt.Println()
			fmt.Printf("Error: A local project named '%s' already exists at:\n  %s\n", projectName, localProjectPath)
			fmt.Print("Please enter a different project name (or press Enter to cancel): ")
			input, _ := reader.ReadString('\n')
			projectName = strings.TrimSpace(input)
			if projectName == "" {
				fmt.Println("Project creation cancelled.")
				return
			}
			continue
		}

		// Remote GitHub repository collision check
		if ghPath := getGHExecutable(); ghPath != "" {
			exists, err := checkGHRepoExists(ghPath, projectName)
			if err == nil && exists {
				fmt.Println()
				fmt.Printf("Error: A repository named '%s' already exists on your GitHub account!\n", projectName)
				fmt.Print("Please enter a different project name (or press Enter to cancel): ")
				input, _ := reader.ReadString('\n')
				projectName = strings.TrimSpace(input)
				if projectName == "" {
					fmt.Println("Project creation cancelled.")
					return
				}
				continue
			}
		}

		// Name is available both locally and remotely
		break
	}

	// -------------------------------------------------------------------------
	// ATOMIC TRANSACTION MANAGEMENT
	// -------------------------------------------------------------------------
	originalWd, err := os.Getwd()
	if err != nil {
		originalWd = cfg.ProjectsPath
	}

	var cleanupActions []func()
	var success bool

	defer func() {
		if !success {
			fmt.Println()
			fmt.Println("==================================================")
			fmt.Println(" [FAILURE] Project creation could not be completed")
			fmt.Println(" Rolling back to prevent half-baked files...")
			fmt.Println("==================================================")

			// Step away from project directory before removing (critical on Windows)
			_ = os.Chdir(originalWd)

			for i := len(cleanupActions) - 1; i >= 0; i-- {
				cleanupActions[i]()
			}
			fmt.Println("[Rollback] Cleanup complete. No partial files or repositories remain.")
			fmt.Println()
		}
	}()

	// 4. Ensure Root Projects Directory Exists
	if err := os.MkdirAll(cfg.ProjectsPath, 0755); err != nil {
		fmt.Printf("Error: Unable to create projects root directory (%s): %v\n", cfg.ProjectsPath, err)
		return
	}

	// 5. Create Local Project Directory
	if err := os.MkdirAll(localProjectPath, 0755); err != nil {
		fmt.Printf("Error: Unable to create project directory (%s): %v\n", localProjectPath, err)
		return
	}

	// Register local directory rollback
	cleanupActions = append(cleanupActions, func() {
		_ = os.Chdir(originalWd)
		if err := os.RemoveAll(localProjectPath); err != nil {
			fmt.Printf("  -> Warning: Failed to remove directory '%s': %v\n", localProjectPath, err)
		} else {
			fmt.Printf("  -> Removed partial local directory: %s\n", localProjectPath)
		}
	})

	// 6. Execute Recipe Scaffolding
	fmt.Printf("\nScaffolding %s project '%s'...\n", recipe.Name, projectName)
	if err := recipe.Execute(localProjectPath, projectName); err != nil {
		fmt.Printf("\nError: Failed to generate project assets: %v\n", err)
		return
	}
	fmt.Println("  -> Project assets created.")

	// 7. Ensure Git is Available and Author is Configured
	gitPath, err := ensureGit()
	if err != nil {
		fmt.Printf("\nError: Git check failed: %v\n", err)
		return
	}
	if err := ensureGitIdentity(); err != nil {
		fmt.Printf("\nError: Git identity setup failed: %v\n", err)
		return
	}

	// 8. Local Git Initialization
	gitInitCmd := exec.Command(gitPath, "init")
	gitInitCmd.Dir = localProjectPath
	if err := gitInitCmd.Run(); err != nil {
		fmt.Printf("\nError: 'git init' failed: %v\n", err)
		return
	}
	fmt.Println("  -> Initialized Git repository.")

	// Set default branch to main
	if err := exec.Command(gitPath, "-C", localProjectPath, "branch", "-M", "main").Run(); err != nil {
		fmt.Printf("\nError: Failed to set default branch to 'main': %v\n", err)
		return
	}

	// 9. Git Add & Initial Commit
	addCmd := exec.Command(gitPath, "add", "-A")
	addCmd.Dir = localProjectPath
	if err := addCmd.Run(); err != nil {
		fmt.Printf("\nError: 'git add' failed: %v\n", err)
		return
	}

	commitCmd := exec.Command(gitPath, "commit", "-m", "Initial commit")
	commitCmd.Dir = localProjectPath
	if err := commitCmd.Run(); err != nil {
		fmt.Printf("\nError: Initial git commit failed: %v\n", err)
		return
	}
	fmt.Println("  -> Created initial Git commit.")

	// 10. Remote GitHub Integration via 'gh' CLI
	ghPath, err := ensureGH()
	if err != nil {
		fmt.Printf("\nError: GitHub CLI is required but could not be configured: %v\n", err)
		return
	}
	if err := ensureGHAuthenticated(ghPath); err != nil {
		fmt.Printf("\nError: GitHub authentication failed: %v\n", err)
		return
	}

	// Register remote GitHub repository rollback in case creation/push fails midway
	cleanupActions = append(cleanupActions, func() {
		if exists, _ := checkGHRepoExists(ghPath, projectName); exists {
			fmt.Printf("  -> Deleting orphaned remote GitHub repository: %s\n", projectName)
			_ = exec.Command(ghPath, "repo", "delete", projectName, "--yes").Run()
		}
	})

	visibilityFlag := "--public"
	if cfg.DefaultVisibility == "private" {
		visibilityFlag = "--private"
	}

	fmt.Printf("  -> Creating %s GitHub repository via 'gh' CLI...\n", cfg.DefaultVisibility)
	ghCmd := exec.Command(ghPath, "repo", "create", projectName, visibilityFlag, "--source=.", "--remote=origin", "--push")
	ghCmd.Dir = localProjectPath
	ghCmd.Stdout = os.Stdout
	ghCmd.Stderr = os.Stderr
	ghCmd.Stdin = os.Stdin

	if err := ghCmd.Run(); err != nil {
		fmt.Printf("\nError: Failed to create or push repository to GitHub: %v\n", err)
		return
	}
	fmt.Println("  -> Remote GitHub repository synchronized successfully!")

	// 11. Launch Editor
	openInEditor(localProjectPath)

	// Mark transaction as complete
	success = true
	fmt.Printf("\n[Success] Project '%s' successfully created and configured at:\n  %s\n", projectName, localProjectPath)
}

func main() {
	if len(os.Args) < 2 {
		displayHelp()
		return
	}

	command := os.Args[1]

	// Help and Config commands can run directly
	switch command {
	case "--help", "-h", "help":
		displayHelp()
		return
	case "config":
		if len(os.Args) > 2 && (os.Args[2] == "--show" || os.Args[2] == "show") {
			DisplayConfig()
			return
		}
		if _, err := RunInteractiveSetup(); err != nil {
			fmt.Printf("Setup failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// All other commands (create, list) require completed configuration
	cfg, err := EnsureConfig()
	if err != nil {
		fmt.Printf("Configuration error: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "list":
		listProjects(cfg)
	case "create":
		createNewProject(cfg)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		displayHelp()
	}
}
