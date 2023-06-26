package application

import (
	"fmt"
	"os"
	"path/filepath"
)

func CreateProjectLayout(dir string) error {

	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	// Create the main directory
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create the config file
	configFile := fmt.Sprintf("%s/config.yml", dir)
	if _, err := os.Create(configFile); err != nil {
		return err
	}

	// Create the static directory
	staticDir := fmt.Sprintf("%s/static", dir)
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		return err
	}

	// Create the styles.css file
	stylesFile := fmt.Sprintf("%s/static/styles.css", dir)
	if _, err := os.Create(stylesFile); err != nil {
		return err
	}

	// Create the content directory
	contentDir := fmt.Sprintf("%s/content", dir)
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		return err
	}

	// Create the index.md file
	indexFile := fmt.Sprintf("%s/content/index.md", dir)
	if _, err := os.Create(indexFile); err != nil {
		return err
	}
	if err := os.WriteFile(indexFile, []byte("# Hello world!\n\nWelcome to bloop!"), 0666); err != nil {
		return err
	}

	// Create the templates directory
	templatesDir := fmt.Sprintf("%s/templates", dir)
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return err
	}

	// Create the main.html file
	mainFile := fmt.Sprintf("%s/templates/main.html", dir)
	if _, err := os.Create(mainFile); err != nil {
		return err
	}
	if err := os.WriteFile(mainFile, []byte("<html>\n<head>\n</head>\n<body>\n<main>\n{{ .content }}\n</main>\n</body>\n</html>\n"), 0666); err != nil {
		return err
	}

	return nil
}
