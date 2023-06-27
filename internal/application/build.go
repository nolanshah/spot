package application

import (
	"main/internal/converters"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

func ResetDirectory(dirPath string) error {
	// Check if the directory exists
	_, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		// Directory doesn't exist, create it
		err := os.Mkdir(dirPath, os.ModePerm)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create directory")
			return err
		}
	} else if err != nil {
		log.Error().Err(err).Msg("Failed to check directory existence")
		return err
	} else {
		// Directory exists, delete its contents
		err := os.RemoveAll(dirPath)
		if err != nil {
			log.Error().Err(err).Msg("Failed to delete directory")
			return err
		}

		// Recreate the directory
		err = os.Mkdir(dirPath, os.ModePerm)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create directory")
			return err
		}
	}

	return nil
}

func ProcessFiles(config Config) error {
	ResetDirectory(config.BuildPath)

	CopyDir(config.StaticPath, config.BuildPath)

	err := filepath.Walk(config.ContentPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error().Err(err).Str("file", filePath).Msg("Error accessing file")
			return err
		}

		if info.IsDir() {
			// Skip directories
			return nil
		}

		extension := filepath.Ext(info.Name())
		fileName := strings.TrimSuffix(info.Name(), extension)

		// Get the relative path of the input file
		relativePath, err := filepath.Rel(config.ContentPath, filePath)
		if err != nil {
			log.Error().Err(err).Str("file", filePath).Msg("Failed to get relative path")
			return err
		}

		absolutePath := filepath.Join(config.ContentPath, relativePath)

		contentEntry := MatchContentEntry(config, absolutePath)
		if contentEntry == nil {
			log.Error().Err(err).Str("absolutePath", absolutePath).Msg("No content entry")
			return err
		}

		// Create the output directory structure
		outputPath := filepath.Join(config.BuildPath, filepath.Dir(relativePath))
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			log.Error().Err(err).Str("file", filePath).Msg("Failed to create output directory structure")
			return err
		}

		if extension == ".docx" || extension == ".md" || extension == ".txt" || extension == ".ipynb" {
			outputFilePath, err := converters.ConvertFileToHTML(config.ContentPath, relativePath, config.BuildPath, fileName)
			if err != nil {
				log.Error().Err(err).Str("input", absolutePath).Str("output", outputFilePath).Msg("Failed to convert file to HTML")
				return err
			}
			err = ApplyTemplateToFile(outputFilePath, contentEntry.Template)
			if err != nil {
				log.Error().Err(err).Str("template", contentEntry.Template).Str("file", outputFilePath).Msg("Failed to apply template to file")
				return err
			}
		} else if extension == ".html" {
			err = ApplyTemplateToFile(absolutePath, contentEntry.Template)
			if err != nil {
				log.Error().Err(err).Str("template", contentEntry.Template).Str("file", absolutePath).Msg("Failed to apply template to file")
				return err
			}
		} else if extension == ".webloc" {
			link, err := converters.ExtractLinkFromWebloc(config.ContentPath, relativePath)
			if err != nil {
				log.Error().Err(err).Str("input", absolutePath).Msg("Failed to extract link from webloc")
				return err
			}
			log.Info().Str("file", relativePath).Str("link", link).Msg("Found a webloc link, not doing anything it with.")
		} else if extension == ".lnk" {
			link, err := converters.ExtractLinkFromShortcut(config.ContentPath, relativePath)
			if err != nil {
				log.Error().Err(err).Str("input", absolutePath).Msg("Failed to extract link from lnk")
				return nil // TODO: don't ignore the error
			}
			log.Info().Str("file", relativePath).Str("link", link).Msg("Found a webloc link, not doing anything it with.")
		} else {
			log.Info().Str("extension", extension).Str("file", relativePath).Msg("Skipping file since extension is not supported")
		}

		return nil
	})

	if err != nil {
		log.Error().Err(err).Msg("Error walking through input directory")
		return err
	}

	return nil
}
