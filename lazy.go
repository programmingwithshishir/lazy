package main

import (
	"context"
	"fmt"
	nodesetup "lazy/templates"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/go-github/v50/github"
	"golang.org/x/oauth2"
)

type Template struct {
	Cmd          string
	FolderNeeded bool
}

type Templates map[string]Template

var templates = Templates{
	"--cpp": {
		Cmd:          "mkdir bin,src,user && echo //Program Here > main.cpp",
		FolderNeeded: true,
	},
	"--node-react-vite": {
		Cmd:          "npm create vite@latest $ProjectName -- --template react && cd $ProjectName && npm install",
		FolderNeeded: false,
	},
}

const projectsPath = "C:/hackathon/programs"

// Node setup
func nodeSetup() {
	projectName := strings.TrimSpace(os.Args[2])
	wg := sync.WaitGroup{}
	err := os.Mkdir(filepath.Join(projectsPath, projectName), 0755)
	if err != nil {
		fmt.Println("Error creating directory", err)
	}
	os.Chdir(filepath.Join(projectsPath, projectName))
	var goCount = 3
	if os.Args[3] == "--node-ts" {
		goCount++
	}
	wg.Add(goCount)
	nodesetup.InitNode()
	go nodesetup.CtreateNodeServer(&wg)
	go nodesetup.CreateFolderStruct(&wg)
	go nodesetup.CreateGitIgnore(&wg)
	if os.Args[3] == "--node-ts" {
		go nodesetup.SetupTypeScript(&wg)
	}
	wg.Wait()
	fmt.Println("Project setup completed")
}

// Displays the help prompt message
func displayHelp() {
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("	--help		   Show help screen")
	fmt.Println("	create	   	   Create a new project")
	fmt.Println("	list		   Display existing projects")
	fmt.Println("Templates")
	fmt.Println("	--cpp			   	   Sets a project folder path")
	fmt.Println("	--node			   	   Sets a node project with javascript")
	fmt.Println("	--node-ts			   Sets a node project with typescript")
	fmt.Println("	--node-react-vite	   Gets the current project folder path")
}

// Lists the project in the projects folder
func listProjects(argumentsCount int) {
	if argumentsCount > 1 {
		fmt.Println("Invalid arguments!!")
		fmt.Println("Usage: lazy list")
		return
	}

	entries, err := os.ReadDir(projectsPath)
	if err != nil {
		fmt.Println("Could not find projects directory")
	}

	fmt.Println()
	fmt.Printf("Local Projects:\n")
	for _, entry := range entries {
		if entry.IsDir() {
			fmt.Println(entry.Name())
		}
	}
}

// Checks if a folder exists
func folderExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// Create a Repository in Github
func createGitHubRepo(token, name string) (*github.Repository, error) {
	// Create authenticated GitHub client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Create repository request
	repo := &github.Repository{
		Name:    github.String(name),
		Private: github.Bool(false),
	}

	// Create the repository
	createdRepo, _, err := client.Repositories.Create(ctx, "", repo)
	if err != nil {
		return nil, err
	}
	return createdRepo, nil
}

// Connects the local repository to Github
func connectLocalToRemote(localPath, remoteURL string) error {
	cmd := exec.Command("git", "remote", "add", "origin", remoteURL)
	cmd.Dir = localPath
	return cmd.Run()
}

// Creates a new project
func createNewProject(argumentsCount int) {
	projectName := strings.TrimSpace(os.Args[2])
	localProjectPath := filepath.Join(projectsPath, projectName)
	if argumentsCount > 3 || argumentsCount < 3 {
		fmt.Println("\nInvalid arguments!!")
		fmt.Println("Usage: lazy create <project-name> <--template>")
		return
	}
	if os.Args[3] == "--node" || os.Args[3] == "--node-ts" {
		nodeSetup()
	} else {
		if folderExists(localProjectPath) {
			fmt.Println("\nProject already exists!!")
		} else {
			if temp, exists := templates[os.Args[3]]; exists {
				fmt.Println("Template Found")
				if temp.FolderNeeded {
					os.Mkdir(localProjectPath, 0755)
					fmt.Println("Created project directory")
					cmd := exec.Command("cmd", "/C", temp.Cmd)
					cmd.Dir = localProjectPath
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					err := cmd.Run()
					if err != nil {
						fmt.Println("Error:", err)
					}
				} else {
					cmd := exec.Command("cmd", "/C", strings.ReplaceAll(temp.Cmd, "$ProjectName", projectName))
					cmd.Dir = projectsPath
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					err := cmd.Run()
					if err != nil {
						fmt.Println("Error:", err)
					}
				}
				fmt.Println("Created project assets")
			}
		}
	}
	repo, err := createGitHubRepo("", projectName)
	if err != nil {
		fmt.Println("Error creating GitHub repository:", err)
		return
	}
	fmt.Println("Initialized a github repository")
	cmd := exec.Command("git", "init")
	cmd.Dir = localProjectPath
	cmd.Run()
	fmt.Println("Initialized a git repository")
	connectLocalToRemote(localProjectPath, *repo.CloneURL)
	fmt.Println("Linked local repository to remote")

	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = localProjectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		fmt.Println("Error:", err)
	}
	cmd = exec.Command("git", "commit", "-m", "'Initial Commit'")
	cmd.Dir = localProjectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		fmt.Println("Error:", err)
	}
	cmd = exec.Command("git", "push", "--set-upstream", "origin", "master")
	cmd.Dir = localProjectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		fmt.Println("Error:", err)
	}
	cmd = exec.Command("code", ".")
	cmd.Dir = localProjectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		fmt.Println("Error:", err)
	}
}

func main() {
	argumentsCount := len(os.Args) - 1

	// Handling commands based on first argument
	if argumentsCount > 0 {
		if os.Args[1] == "list" {
			listProjects(argumentsCount)
		} else if os.Args[1] == "create" {
			createNewProject(argumentsCount)
		} else {
			displayHelp()
		}
	} else {
		displayHelp()
	}
}
