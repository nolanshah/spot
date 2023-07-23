package application

import (
	"html/template"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

func hasPrefix(prefix string, s string) bool {
	log.Logger.Trace().Str("prefix", prefix).Str("string", s).Msg("Called HasPrefix.")
	return strings.HasPrefix(s, prefix)
}

func loadTemplates(baseTemplateDirPath string, templateName string) *template.Template {
	var paths []string
	err := filepath.Walk(baseTemplateDirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		log.Fatal().Err(err).Str("baseTemplateDirPath", baseTemplateDirPath).Msg("Failed to collect templates from base dir.")
	}

	log.Trace().Any("paths", paths).Msg("Template paths.")

	// Parse all the templates (incl possible deps) with the function map configured
	tmpl, err := template.New("__sentinel").Funcs(template.FuncMap{
		"HasPrefix": hasPrefix,
	}).ParseFiles(paths...)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse templates.")
	}

	// Then re-load the template we care about and make sure it's named appropriately so it can be referenced
	return template.Must(tmpl.New(templateName).ParseFiles(filepath.Join(baseTemplateDirPath, templateName)))
}

func ApplyTemplateToFile(tData TData) error {
	templatePath := tData.Page.TemplatePath
	contentHtmlPath := tData.Page.DestinationPath

	tmpl := loadTemplates(path.Dir(templatePath), path.Base(templatePath))

	// Create a buffer to hold the rendered output
	output, err := os.Create(contentHtmlPath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create output file.")
	}
	defer output.Close()

	log.Trace().Str("templatePath", path.Base(templatePath)).Any("data.Page", tData.Page).Msg("Attempting to apply template with the following data.")

	// Apply the template to the contents and write the output to the file
	if err := tmpl.ExecuteTemplate(output, path.Base(templatePath), tData); err != nil {
		log.Error().Err(err).Msg("Failed to apply template.")
		return err
	}

	log.Trace().Str("template", path.Base(templatePath)).Str("file", contentHtmlPath).Msg("Successfully applied template to file.")

	return nil
}
