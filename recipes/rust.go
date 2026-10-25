package recipes

import (
	"fmt"
	"os"
	"path/filepath"
)

func init() {
	Register(Recipe{
		Flag:               "--rust",
		Name:               "Rust",
		Description:        "Rust project structure with Cargo.toml and src/main.rs",
		CheckPrerequisites: EnsureRust,
		Execute: func(projectPath, projectName string) error {
			return SetupRust(projectPath, projectName)
		},
	})
}

// SetupRust scaffolds a Cargo-based Rust project layout
func SetupRust(projectPath, projectName string) error {
	srcDir := filepath.Join(projectPath, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return fmt.Errorf("failed to create src directory: %w", err)
	}

	cargoToml := `[package]
name = "` + projectName + `"
version = "0.1.0"
edition = "2021"

[dependencies]
`
	if err := os.WriteFile(filepath.Join(projectPath, "Cargo.toml"), []byte(cargoToml), 0644); err != nil {
		return err
	}

	mainRs := `fn main() {
    println!("Hello from ` + projectName + `!");
}
`
	if err := os.WriteFile(filepath.Join(srcDir, "main.rs"), []byte(mainRs), 0644); err != nil {
		return err
	}

	gitIgnore := `/target
**/*.rs.bk
Cargo.lock
.vscode/
.idea/
`
	_ = os.WriteFile(filepath.Join(projectPath, ".gitignore"), []byte(gitIgnore), 0644)
	return nil
}
