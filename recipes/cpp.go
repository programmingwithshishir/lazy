package recipes

import (
	"fmt"
	"os"
	"path/filepath"
)

func init() {
	Register(Recipe{
		Flag:               "--cpp",
		Name:               "C++",
		Description:        "C++ project structure with src, include, and CMakeLists.txt",
		CheckPrerequisites: EnsureCpp,
		Execute: func(projectPath, projectName string) error {
			return SetupCpp(projectPath, projectName)
		},
	})
}

// SetupCpp scaffolds a modern C++ project layout
func SetupCpp(projectPath, projectName string) error {
	dirs := []string{"bin", "src", "include"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(projectPath, dir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	mainContent := `#include <iostream>

int main() {
    std::cout << "Hello from ` + projectName + `!" << std::endl;
    return 0;
}
`
	if err := os.WriteFile(filepath.Join(projectPath, "src", "main.cpp"), []byte(mainContent), 0644); err != nil {
		return fmt.Errorf("failed to write main.cpp: %w", err)
	}

	cmakeContent := `cmake_minimum_required(VERSION 3.15)
project(` + projectName + ` CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

include_directories(include)

add_executable(` + projectName + ` src/main.cpp)
`
	_ = os.WriteFile(filepath.Join(projectPath, "CMakeLists.txt"), []byte(cmakeContent), 0644)

	gitIgnoreContent := `bin/
build/
*.o
*.obj
*.exe
*.out
.vscode/
.idea/
`
	_ = os.WriteFile(filepath.Join(projectPath, ".gitignore"), []byte(gitIgnoreContent), 0644)
	return nil
}
