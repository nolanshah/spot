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
		ContentPath:     "content/",
		StaticPath:      "static/",
		TemplatesPath:   "templates/",
		BuildPath:       "dist/",
		DefaultTemplate: "",
		SiteTitle:       "My Spot Site",
		SiteDescription: "This website is created with spot!",
		Content: []ContentEntry{
			{
				InputPath:  "index.md",
				OutputPath: "index.html",
				Template:   "main.html",
				Title:      "Home",
			},
			{
				InputPath:  "/blog/",
				OutputPath: "",
				Template:   "main.html",
				Title:      "Blog",
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

	// Create the content/blog/ directory
	blogDir := fmt.Sprintf("%s/content/blog", dir)
	if err := os.MkdirAll(blogDir, 0755); err != nil {
		return err
	}

	// Create the content/blog/first.md file
	blogFirstFile := fmt.Sprintf("%s/content/blog/first.md", dir)
	if _, err := os.Create(blogFirstFile); err != nil {
		return err
	}
	if err := os.WriteFile(blogFirstFile, []byte(contentContentBlogFirstMd), 0666); err != nil {
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
		<title>{{ .Page.Title }} - {{ .Site.Title }}</title>
	</head>
	<body>
		<h1>{{ .Site.Title }}</h1>
		<h2>{{ .Page.Title }}</h2>
		<main>
			<section>
				<ul>
					{{- range $i, $e := .Pages.List -}}
					{{- if HasPrefix "/blog" .UrlPath -}}
					<li>
						<a class="blog-list-link" href="{{ .UrlPath }}">{{ .CreatedAt.Format "02 Jan 2006" }} - {{ .Title }}</a>
					</li>
					{{- end -}}
					{{- end -}}
				</ul>
			</section>
			<section>
				{{block "content" .}}
		
				{{ .Contents }}
		
				{{end}}
			</section>
		</main>
	</body>
</html>
`

const contentContentIndexMd = `
Welcome to spot! This is the index page for your website.
`

const contentContentBlogFirstMd = `
This is the first blog post.
`
