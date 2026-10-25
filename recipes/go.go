package recipes

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func init() {
	Register(Recipe{
		Flag:               "--go",
		Name:               "Go",
		Description:        "Go application with go.mod, cmd/main.go, internal/, and pkg/",
		CheckPrerequisites: EnsureGo,
		Execute: func(projectPath, projectName string) error {
			return SetupGo(projectPath, projectName)
		},
	})
}

// SetupGo scaffolds a standard Go application layout
func SetupGo(projectPath, projectName string) error {
	dirs := []string{"cmd", "internal", "pkg"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(projectPath, dir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	mainGo := `package main

import "fmt"

func main() {
	fmt.Println("Hello from ` + projectName + `!")
}
`
	if err := os.WriteFile(filepath.Join(projectPath, "cmd", "main.go"), []byte(mainGo), 0644); err != nil {
		return err
	}

	// Initialize go module
	cmd := exec.Command("go", "mod", "init", projectName)
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod init failed: %w", err)
	}

	gitIgnore := `bin/
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out
.vscode/
.idea/
`
	if err := os.WriteFile(filepath.Join(projectPath, ".gitignore"), []byte(gitIgnore), 0644); err != nil {
		return err
	}
	return nil
}
