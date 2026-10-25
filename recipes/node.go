package recipes

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func init() {
	Register(Recipe{
		Flag:               "--node",
		Name:               "Node.js (JavaScript)",
		Description:        "Node.js Express project with JavaScript",
		CheckPrerequisites: EnsureNode,
		Execute: func(projectPath, projectName string) error {
			return SetupNodeJS(projectPath)
		},
	})

	Register(Recipe{
		Flag:               "--node-ts",
		Name:               "Node.js (TypeScript)",
		Description:        "Node.js Express project with TypeScript",
		CheckPrerequisites: EnsureNode,
		Execute: func(projectPath, projectName string) error {
			return SetupNodeTS(projectPath)
		},
	})
}

// SetupNodeJS scaffolds a Node.js JavaScript Express server
func SetupNodeJS(projectPath string) error {
	originalDir, err := os.Getwd()
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(projectPath); err != nil {
		return err
	}

	fmt.Println("Initializing Node package...")
	if err := initNodePackage(); err != nil {
		return fmt.Errorf("npm init failed: %w", err)
	}

	fmt.Println("Creating folder structure...")
	if err := createNodeFolderStruct(); err != nil {
		return err
	}

	fmt.Println("Creating .gitignore...")
	if err := createNodeGitIgnore(); err != nil {
		return err
	}

	fmt.Println("Creating server.js...")
	return createNodeServer()
}

// SetupNodeTS scaffolds a Node.js TypeScript Express server
func SetupNodeTS(projectPath string) error {
	originalDir, err := os.Getwd()
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(projectPath); err != nil {
		return err
	}

	fmt.Println("Initializing Node package...")
	if err := initNodePackage(); err != nil {
		return fmt.Errorf("npm init failed: %w", err)
	}

	fmt.Println("Creating folder structure...")
	if err := createNodeFolderStruct(); err != nil {
		return err
	}

	fmt.Println("Creating .gitignore...")
	if err := createNodeGitIgnore(); err != nil {
		return err
	}

	fmt.Println("Configuring TypeScript dependencies and tsconfig.json...")
	return setupTypeScript()
}

func createNodeGitIgnore() error {
	gitIgnoreCode := `# Node modules
node_modules/

# Logs
logs/
*.log
npm-debug.log*
yarn-debug.log*
yarn-error.log*

# Environment files
.env
.env.local
.env.*.local

# Production build output
dist/
build/

# Optional: Cache directories
.npm/
.eslintcache

# Operating system files
.DS_Store
Thumbs.db

# IDE / Editor directories
.vscode/
.idea/
`
	return os.WriteFile(".gitignore", []byte(gitIgnoreCode), 0644)
}

func initNodePackage() error {
	initCmd := exec.Command("npm", "init", "-y")
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr
	if err := initCmd.Run(); err != nil {
		return fmt.Errorf("npm init failed: %w", err)
	}

	packageCode := `{
  "name": "my-js-app",
  "version": "1.0.0",
  "main": "src/server.js",
  "scripts": {
    "dev": "nodemon src/server.js",
    "start": "node src/server.js"
  },
  "dependencies": {
    "express": "^4.18.2"
  },
  "devDependencies": {
    "nodemon": "^2.0.20"
  }
}
`
	return os.WriteFile("package.json", []byte(packageCode), 0644)
}

func createNodeFolderStruct() error {
	folders := []string{
		"src",
		"src/controllers",
		"src/models",
		"src/routes",
		"src/config",
		"src/middlewares",
		"src/utils",
	}

	for _, folder := range folders {
		if err := os.MkdirAll(folder, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", folder, err)
		}
	}

	envContent := "# Environment Variables\nPORT=3000\n"
	return os.WriteFile(".env", []byte(envContent), 0644)
}

func createNodeServer() error {
	serverCode := `const http = require('http');
const port = process.env.PORT || 3000;

const server = http.createServer((req, res) => {
  res.statusCode = 200;
  res.setHeader('Content-Type', 'text/plain');
  res.end('Hello World\n');
});

server.listen(port, () => {
  console.log('Server is running on port ' + port);
});
`
	return os.WriteFile(filepath.Join("src", "server.js"), []byte(serverCode), 0644)
}

func setupTypeScript() error {
	tsInstallCmd := exec.Command("npm", "install", "--save-dev", "typescript", "@types/node", "ts-node", "@types/express", "nodemon")
	tsInstallCmd.Stdout = os.Stdout
	tsInstallCmd.Stderr = os.Stderr

	if err := tsInstallCmd.Run(); err != nil {
		return fmt.Errorf("failed to install TypeScript dependencies: %w", err)
	}

	tsInitCode := `{
  "compilerOptions": {
    "target": "es2020",
    "module": "commonjs",
    "rootDir": "./src",
    "outDir": "./dist",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist"]
}
`
	if err := os.WriteFile("tsconfig.json", []byte(tsInitCode), 0644); err != nil {
		return fmt.Errorf("failed to write tsconfig.json: %w", err)
	}

	serverCode := `import http from 'http';

const port = process.env.PORT || 3000;

const server = http.createServer((req, res) => {
  res.statusCode = 200;
  res.setHeader('Content-Type', 'text/plain');
  res.end('Hello World\n');
});

server.listen(port, () => {
  console.log('Server is running on port ' + port);
});
`
	_ = os.Remove(filepath.Join("src", "server.js"))
	if err := os.WriteFile(filepath.Join("src", "server.ts"), []byte(serverCode), 0644); err != nil {
		return fmt.Errorf("failed to write server.ts: %w", err)
	}

	tsPackageCode := `{
  "name": "my-ts-app",
  "version": "1.0.0",
  "main": "dist/server.js",
  "scripts": {
    "dev": "ts-node src/server.ts",
    "build": "tsc",
    "start": "node dist/server.js"
  },
  "dependencies": {
    "express": "^4.18.2"
  },
  "devDependencies": {
    "typescript": "^4.9.5",
    "ts-node": "^10.9.1",
    "@types/node": "^18.15.11",
    "@types/express": "^4.17.17",
    "nodemon": "^2.0.20"
  }
}
`
	return os.WriteFile("package.json", []byte(tsPackageCode), 0644)
}
