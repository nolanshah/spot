package application

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
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
	configFile := fmt.Sprintf("%s/config.yaml", dir)
	config := Config{
		ContentPath:   "content/",
		StaticPath:    "static/",
		TemplatesPath: "templates/",
		BuildPath:     "dist/",
		Content: []ContentEntry{
			{
				InputPath:  "index.md",
				OutputPath: "index.html",
				Converter:  nil,
				Template:   "main.html",
			},
		},
	}
	configData, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	if err := os.WriteFile(configFile, configData, 0666); err != nil {
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
	if err := os.WriteFile(indexFile, []byte(contentContentIndexMd), 0666); err != nil {
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
	if err := os.WriteFile(mainFile, []byte(contentTemplateMainHtml), 0666); err != nil {
		return err
	}

	return nil
}

const contentTemplateMainHtml = `
<html>
	<head>
		<title>Your First Bloop Template!</title>
	</head>
	<body>
		<h1>Your First Bloop Template!</h1>
		<main>
			{{ .Contents }}
		</main>
	</body>
</html>
`

const contentContentIndexMd = `
# Hello world!

Welcome to bloop!
`
