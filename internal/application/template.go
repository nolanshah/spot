package application

import (
	"html/template"
	"os"
	"path"
	"strings"

	"github.com/rs/zerolog/log"
)

func hasPrefix(prefix string, s string) bool {
	log.Logger.Trace().Str("prefix", prefix).Str("string", s).Msg("Called HasPrefix.")
	return strings.HasPrefix(s, prefix)
}

func ApplyTemplateToFile(tData TData) error {
	templatePath := tData.Page.TemplatePath
	contentHtmlPath := tData.Page.DestinationPath

	// Parse the template file
	tmpl, err := template.New(path.Base(templatePath)).Funcs(template.FuncMap{
		"HasPrefix": hasPrefix,
	}).ParseFiles(templatePath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse template.")
	}

	// Create a buffer to hold the rendered output
	output := &os.File{}
	if output, err = os.Create(contentHtmlPath); err != nil {
		log.Error().Err(err).Msg("Failed to create output file.")
	}
	defer output.Close()

	log.Trace().Str("templatePath", templatePath).Any("data", tData).Msg("Attempting to apply template with the following data.")

	// Apply the template to the contents and write the output to the file
	if err := tmpl.Execute(output, tData); err != nil {
		log.Error().Err(err).Msg("Failed to apply template.")
		return err
	}

	log.Trace().Str("template", templatePath).Str("file", contentHtmlPath).Msg("Successfully applied template to file.")

	return nil
}
