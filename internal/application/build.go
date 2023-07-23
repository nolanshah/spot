package application

import (
	"html/template"
	"io/ioutil"
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

type page struct {
	url            string
	relContentPath string
	absContentPath string
	relOutputPath  string
	absOutputPath  string
	contentEntry   ContentEntry
	fileNameNoExt  string
}

func ProcessFiles(config Config) error {
	ResetDirectory(config.BuildPath)

	CopyDir(config.StaticPath, config.BuildPath)

	pages := make([]page, 0)

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
		relContentPath, err := filepath.Rel(config.ContentPath, filePath)
		if err != nil {
			log.Error().Err(err).Str("file", filePath).Msg("Failed to get relative path")
			return err
		}

		absolutePath := filepath.Join(config.ContentPath, relContentPath)

		contentEntry := MatchContentEntry(config, absolutePath)
		if contentEntry == nil {
			log.Error().Err(err).Str("absolutePath", absolutePath).Msg("No content entry")
			return err
		}

		// Create the output directory structure
		outputPath := filepath.Join(config.BuildPath, filepath.Dir(relContentPath))
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			log.Error().Err(err).Str("file", filePath).Msg("Failed to create output directory structure")
			return err
		}

		if extension == ".docx" || extension == ".md" || extension == ".txt" || extension == ".ipynb" {
			outputFilePath, err := converters.ConvertFileToHTML(config.ContentPath, relContentPath, config.BuildPath, fileName)
			if err != nil {
				log.Error().Err(err).Str("input", absolutePath).Str("output", outputFilePath).Msg("Failed to convert file to HTML")
				return err
			}

			relOutputPath, err := filepath.Rel(config.BuildPath, outputFilePath)
			if err != nil {
				log.Error().Err(err).Str("file", filePath).Msg("Failed to get relative path")
				return err
			}

			pages = append(pages, page{
				url:            strings.TrimSuffix(relOutputPath, "index.html"),
				relContentPath: relContentPath,
				absContentPath: contentEntry.InputPath,
				relOutputPath:  relOutputPath,
				absOutputPath:  contentEntry.OutputPath,
				contentEntry:   *contentEntry,
				fileNameNoExt:  fileName,
			})
		} else if extension == ".html" {
			CopyFile(absolutePath, outputPath)

			relOutputPath, err := filepath.Rel(config.BuildPath, outputPath)
			if err != nil {
				log.Error().Err(err).Str("file", filePath).Msg("Failed to get relative path")
				return err
			}

			pages = append(pages, page{
				url:            strings.TrimSuffix(relOutputPath, "index.html"),
				relContentPath: relContentPath,
				absContentPath: contentEntry.InputPath,
				relOutputPath:  relOutputPath,
				absOutputPath:  contentEntry.OutputPath,
				contentEntry:   *contentEntry,
				fileNameNoExt:  fileName,
			})
		} else if extension == ".webloc" {
			link, err := converters.ExtractLinkFromWebloc(config.ContentPath, relContentPath)
			if err != nil {
				log.Error().Err(err).Str("input", absolutePath).Msg("Failed to extract link from webloc")
				return err
			}
			log.Info().Str("file", relContentPath).Str("link", link).Msg("Found a webloc link, not doing anything it with.")
		} else if extension == ".lnk" {
			link, err := converters.ExtractLinkFromShortcut(config.ContentPath, relContentPath)
			if err != nil {
				log.Error().Err(err).Str("input", absolutePath).Msg("Failed to extract link from lnk")
				return nil // TODO: don't ignore the error
			}
			log.Info().Str("file", relContentPath).Str("link", link).Msg("Found a webloc link, not doing anything it with.")
		} else {
			log.Info().Str("extension", extension).Str("file", relContentPath).Msg("Skipping file since extension is not supported")
		}

		return nil
	})

	// transform prior repr of pages into list of TPage
	tPages := make([]TPage, 0, len(pages))
	for _, p := range pages {
		foundTitle := p.contentEntry.Title
		if len(foundTitle) == 0 {
			foundTitleP := GetTitleForHtmlFile(p.absOutputPath)
			if foundTitleP != nil {
				foundTitle = *foundTitleP
			} else {
				log.Warn().Str("file", p.relContentPath).Msg("No title for page")
			}
		}
		foundDesc := p.contentEntry.Description
		if len(foundDesc) == 0 {
			log.Warn().Str("file", p.relContentPath).Msg("No description for page")
		}

		tPages = append(tPages, TPage{
			SourcePath:      p.absContentPath,
			TemplatePath:    p.contentEntry.Template,
			DestinationPath: p.absOutputPath,
			UrlPath:         "/" + p.url,
			Title:           foundTitle,
			Description:     foundDesc,
			CreatedAt:       p.contentEntry.CreatedAt,
			Tags:            p.contentEntry.Tags,
			Metadata:        p.contentEntry.Metadata,
		})
	}

	for _, tPage := range tPages {
		// Read the contents of the file
		contents, err := ioutil.ReadFile(tPage.DestinationPath)
		if err != nil {
			log.Error().Err(err).Msg("Failed to read file")
		}

		tData := TData{
			Page:     tPage,
			Contents: template.HTML(contents),
			Pages: TPageList{
				List: tPages,
			},
		} // page.absOutputPath, page.contentEntry.Template, &pages_urls

		err = ApplyTemplateToFile(tData)
		if err != nil {
			log.Error().Err(err).Any("tPage", tPage).Msg("Failed to apply template to file")
			return err
		}
	}

	if err != nil {
		log.Error().Err(err).Msg("Error walking through input directory")
		return err
	}

	return nil
}
