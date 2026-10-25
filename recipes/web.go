package recipes

import (
	"fmt"
	"os"
	"path/filepath"
)

func init() {
	Register(Recipe{
		Flag:        "--web",
		Name:        "HTML / CSS / JS",
		Description: "Vanilla Web starter with index.html, style.css, and app.js",
		Execute: func(projectPath, projectName string) error {
			return SetupWeb(projectPath, projectName)
		},
	})
}

// SetupWeb scaffolds a clean frontend starter
func SetupWeb(projectPath, projectName string) error {
	dirs := []string{"css", "js", "assets"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(projectPath, dir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	htmlContent := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + projectName + `</title>
    <link rel="stylesheet" href="css/style.css">
</head>
<body>
    <div class="container">
        <h1>Welcome to ` + projectName + `</h1>
        <p>Your web project is ready to go!</p>
    </div>
    <script src="js/app.js"></script>
</body>
</html>
`
	if err := os.WriteFile(filepath.Join(projectPath, "index.html"), []byte(htmlContent), 0644); err != nil {
		return err
	}

	cssContent := `* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 100vh;
    background-color: #0f172a;
    color: #f8fafc;
}

.container {
    text-align: center;
    padding: 2rem;
    border-radius: 12px;
    background: #1e293b;
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.5);
}

h1 {
    margin-bottom: 1rem;
    color: #38bdf8;
}
`
	if err := os.WriteFile(filepath.Join(projectPath, "css", "style.css"), []byte(cssContent), 0644); err != nil {
		return err
	}

	jsContent := `document.addEventListener('DOMContentLoaded', () => {
    console.log('` + projectName + ` loaded successfully!');
});
`
	if err := os.WriteFile(filepath.Join(projectPath, "js", "app.js"), []byte(jsContent), 0644); err != nil {
		return err
	}

	gitIgnoreContent := `.DS_Store
Thumbs.db
.vscode/
.idea/
`
	_ = os.WriteFile(filepath.Join(projectPath, ".gitignore"), []byte(gitIgnoreContent), 0644)
	return nil
}
