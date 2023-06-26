package application

import (
	"html/template"
	"io/ioutil"
	"os"

	"github.com/rs/zerolog/log"
)

func ApplyTemplateToFile(contentHtmlPath string, templatePath string) error {
	// Read the contents of the file
	contents, err := ioutil.ReadFile(contentHtmlPath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read file")
	}

	// Parse the template file
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse template")
	}

	// Create a buffer to hold the rendered output
	output := &os.File{}
	if output, err = os.Create(contentHtmlPath); err != nil {
		log.Error().Err(err).Msg("Failed to create output file")
	}
	defer output.Close()

	data := struct {
		Contents template.HTML
	}{
		Contents: template.HTML(contents),
	}

	// Apply the template to the contents and write the output to the file
	if err := tmpl.Execute(output, data); err != nil {
		log.Error().Err(err).Msg("Failed to apply template")
		return err
	}

	log.Info().Str("template", templatePath).Str("file", contentHtmlPath).Msg("Successfully applied template to file")

	return nil
}
