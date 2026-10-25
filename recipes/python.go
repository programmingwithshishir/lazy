package recipes

import (
	"fmt"
	"os"
	"path/filepath"
)

func init() {
	Register(Recipe{
		Flag:               "--python",
		Name:               "Python",
		Description:        "Python project with src/main.py, requirements.txt, and virtualenv setup",
		CheckPrerequisites: EnsurePython,
		Execute: func(projectPath, projectName string) error {
			return SetupPython(projectPath, projectName)
		},
	})
}

// SetupPython scaffolds a clean Python project
func SetupPython(projectPath, projectName string) error {
	dirs := []string{"src", "tests"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(projectPath, dir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	mainPy := `"""Main entry point for ` + projectName + `"""

def main():
    print("Hello from ` + projectName + `!")

if __name__ == "__main__":
    main()
`
	if err := os.WriteFile(filepath.Join(projectPath, "src", "main.py"), []byte(mainPy), 0644); err != nil {
		return err
	}

	reqs := `# Python Dependencies
# Add your packages here (e.g. requests>=2.31.0)
`
	_ = os.WriteFile(filepath.Join(projectPath, "requirements.txt"), []byte(reqs), 0644)

	readme := `# ` + projectName + `

A Python project scaffolded with Lazy.

## Quick Start
` + "```bash\n# Create virtual environment\npython -m venv .venv\n\n# Activate (Windows)\n.venv\\Scripts\\activate\n\n# Run\npython src/main.py\n```" + `
`
	_ = os.WriteFile(filepath.Join(projectPath, "README.md"), []byte(readme), 0644)

	gitIgnore := `__pycache__/
*.py[cod]
*$py.class
*.so
.Python
env/
build/
develop-eggs/
dist/
downloads/
eggs/
.eggs/
lib/
lib64/
parts/
sdist/
var/
wheels/
share/python-wheels/
*.egg-info/
.installed.cfg
*.egg
.venv/
venv/
ENV/
.pytest_cache/
.vscode/
.idea/
`
	_ = os.WriteFile(filepath.Join(projectPath, ".gitignore"), []byte(gitIgnore), 0644)
	return nil
}
