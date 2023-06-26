package application

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
)

func ApplyTemplateToFile(contentHtmlPath string, templatePath string) error {
	// Read the contents of the file
	contents, err := ioutil.ReadFile(contentHtmlPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// Parse the template file
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	// Create a buffer to hold the rendered output
	output := &os.File{}
	if output, err = os.Create(contentHtmlPath); err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer output.Close()

	data := struct {
		Contents template.HTML
	}{
		Contents: template.HTML(contents),
	}

	// Apply the template to the contents and write the output to the file
	if err := tmpl.Execute(output, data); err != nil {
		return fmt.Errorf("failed to apply template: %v", err)
	}

	return nil
}
