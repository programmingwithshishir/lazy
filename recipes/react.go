package recipes

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func init() {
	Register(Recipe{
		Flag:               "--node-react-vite",
		Name:               "React (Vite)",
		Description:        "React web application scaffolded with Vite and npm install",
		CheckPrerequisites: EnsureNode,
		Execute: func(projectPath, projectName string) error {
			return SetupReactVite(projectPath, projectName)
		},
	})
}

// SetupReactVite scaffolds a React frontend using Vite
func SetupReactVite(projectPath, projectName string) error {
	parentDir := filepath.Dir(projectPath)

	fmt.Println("Running: npm create vite@latest ...")
	cmdStr := fmt.Sprintf("npm create vite@latest %s -- --template react && cd %s && npm install", projectName, projectName)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", cmdStr)
	} else {
		cmd = exec.Command("sh", "-c", cmdStr)
	}
	cmd.Dir = parentDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}
