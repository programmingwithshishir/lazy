package nodesetup

import (
	"os"
	"os/exec"
	"sync"
)

// function to create .gitignore file
func CreateGitIgnore(wg *sync.WaitGroup) {
	defer wg.Done()

	os.Create(".gitignore")

	//gitignore code
	gitIgnoreCode := `
	# Node modules
	node_modules/

	# Logs
	logs/
	*.log
	npm-debug.log*
	yarn-debug.log*
	yarn-error.log*

	# Environment files
	.env	

	# Production build output
	dist/

	# Optional: Cache directories
	.npm/
	.eslintcache

	# Operating system files
	.DS_Store

	# IDE / Editor directories
	.vscode/
	.idea/
	`
	os.WriteFile(".gitignore", []byte(gitIgnoreCode), 0644)
}

// function to create node setup
func InitNode() {
	initCmd := exec.Command("npm", "init", "-y")
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr
	initCmd.Run()

	packageCode := `{
  "name": "my-js-app",
  "version": "1.0.0",
  "main": "server.js",
  "scripts": {
    "dev": "nodemon src/server.js", 
    "start": "node src/server.js",
    "build": "echo \"No build step for plain JavaScript\""
  },
  "dependencies": {
    "express": "^4.18.2"
  },
  "devDependencies": {
    "nodemon": "^2.0.20"
  }
}
`
	//package.json basic code
	os.WriteFile("package.json", []byte(packageCode), 0644)
}

// function to create folder stucture
func CreateFolderStruct(wg *sync.WaitGroup) {
	defer wg.Done()

	//create folders
	os.Mkdir("src", 0755)
	os.Mkdir("src/controllers", 0755)
	os.Mkdir("src/models", 0755)
	os.Mkdir("src/routes", 0755)
	os.Mkdir("src/config", 0755)
	os.Mkdir("src/middlewares", 0755)
	os.Mkdir("src/utils", 0755)

	//create files
	os.Create(".env")

}

// function to create simple node server code with http
func CtreateNodeServer(wg *sync.WaitGroup) {
	defer wg.Done()

	//basic server code
	serverCode := `
	const http = require('http');
	const port = process.env.PORT || 3000;
	
	const server = http.createServer((req, res) => {
	  res.statusCode = 200;
	  res.setHeader('Content-Type', 'text/plain');
	  res.end('Hello World');
	  });
	
	server.listen(port, () => {
	  console.log('Server is running on port ' + port);
	  });
	  `

	os.Create("src/server.js")
	os.WriteFile("src/server.js", []byte(serverCode), 0644)
}

// function to setup typescript
func SetupTypeScript(wg *sync.WaitGroup) {
	defer wg.Done()

	//installation of  typescript
	tsInstallCmd := exec.Command("npm", "install", "typescript", "@types/node", "ts-node", "--save-dev")
	tsInstallCmd.Stdout = os.Stdout
	tsInstallCmd.Stderr = os.Stderr

	if err := tsInstallCmd.Run(); err != nil {
		panic(err)
	}

	//initializing typescript
	tsInitCMd := exec.Command("npx", "tsc", "--init")
	tsInitCMd.Stdout = os.Stdout
	tsInitCMd.Stderr = os.Stderr

	if err := tsInitCMd.Run(); err != nil {
		panic(err)
	}

	//tsconfig.json code
	tsInitCode := `{
  "compilerOptions": {
    "target": "es6",                         
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
	os.WriteFile("tsconfig.json", []byte(tsInitCode), 0644)
	os.Rename("src/server.js", "src/server.ts")

	serverCode := `
	import { IncomingMessage, RequestListener, Server, ServerResponse } from "http";

const http = require("http");
const port = process.env.PORT || 3000;

// Create a server with ts types
const server = http.createServer(
  (req: IncomingMessage, res: ServerResponse) => {
    res.statusCode = 200;
    res.setHeader("Content-Type", "text/plain");
    res.end("Hello World");
  }
);

server.listen(port, () => {
  console.log("Server is running on port " + port);
});
`
	os.WriteFile("src/server.ts", []byte(serverCode), 0644)

	//package.json code
	tsPackageCode := `{
  "name": "my-ts-app",
  "version": "1.0.0",
  "main": "dist/index.js",
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
	os.WriteFile("package.json", []byte(tsPackageCode), 0644)
}

// function to setup ESlint
func SetupESlint() {

	//installation of eslint
	esInstallCmd := exec.Command("npm", "install", "--save-dev", "eslint")
	esInstallCmd.Stdout = os.Stdout
	esInstallCmd.Stderr = os.Stderr

	if err := esInstallCmd.Run(); err != nil {
		panic(err)
	}

	//creating eslint file
	eslintrcCode := `{
  "env": {
    "node": true,
    "es2021": true
  },
  "extends": [
    "eslint:recommended",
  ],
  "parserOptions": {
    "ecmaVersion": 12,
    "sourceType": "module"
  },
  "rules": {
    // You can customize your ESLint rules here
  }
}
`
	os.Create(".eslintrc.json")
	os.WriteFile(".eslintrc.json", []byte(eslintrcCode), 0644)

}
