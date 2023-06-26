package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"main/internal/application"
	"main/internal/converters"
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

func processFiles(config application.Config) error {
	ResetDirectory(config.BuildPath)

	application.CopyDir(config.StaticPath, config.BuildPath)

	err := filepath.Walk(config.ContentPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error().Err(err).Str("file", filePath).Msg("Error accessing file")
			return err
		}

		if info.IsDir() {
			// Skip directories
			return nil
		}

		// Get the relative path of the input file
		relativePath, err := filepath.Rel(config.ContentPath, filePath)
		if err != nil {
			log.Error().Err(err).Str("file", filePath).Msg("Failed to get relative path")
			return err
		}

		absolutePath := filepath.Join(config.ContentPath, relativePath)

		contentEntry := application.MatchContentEntry(config, absolutePath)
		if contentEntry == nil {
			log.Error().Err(err).Str("absolutePath", absolutePath).Msg("No matching content entry")
			return err
		}

		templateAbsolutePath := filepath.Join(config.TemplatesPath, contentEntry.Template)

		// Create the output directory structure
		outputPath := filepath.Join(config.BuildPath, filepath.Dir(relativePath))
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			log.Error().Err(err).Str("file", filePath).Msg("Failed to create output directory structure")
			return err
		}

		extension := filepath.Ext(info.Name())
		fileName := strings.TrimSuffix(info.Name(), extension)

		if extension == ".docx" || extension == ".md" || extension == ".txt" || extension == ".ipynb" {
			outputFilePath, err := converters.ConvertFileToHTML(config.ContentPath, relativePath, config.BuildPath, fileName)
			if err != nil {
				log.Error().Err(err).Str("input", absolutePath).Str("output", outputFilePath).Msg("Failed to convert file to HTML")
				return err
			}
			err = application.ApplyTemplateToFile(outputFilePath, templateAbsolutePath)
			if err != nil {
				log.Error().Err(err).Str("template", templateAbsolutePath).Str("file", outputFilePath).Msg("Failed to apply template to file")
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

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	app := &cli.App{
		Name:  "bloop",
		Usage: "build static websites from unstructured docs",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:   "debug",
				Value:  false,
				Hidden: true,
			},
		},
		Before: func(cCtx *cli.Context) error {
			if cCtx.Bool("debug") {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
				log.Debug().Msg("Debug logging enabled.")
			}
			return nil
		},
	}

	initCmd := &cli.Command{
		Name:  "init",
		Usage: "Initialize project",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "dir",
				Usage: "project directory",
			},
		},
		Action: func(cCtx *cli.Context) error {
			dir := cCtx.String("dir")
			err := application.CreateProjectLayout(dir)
			if err != nil {
				log.Error().Err(err).Msg("Failed to initialize project.")
				return err
			} else {
				log.Info().Msg("Successfully initialized project.")
				return nil
			}
		},
	}

	buildCommand := &cli.Command{
		Name:  "build",
		Usage: "Build project",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:     "config",
				Usage:    "config file path",
				Value:    "config.yaml",
				Required: false,
			},
			&cli.BoolFlag{
				Name:  "watch",
				Value: false,
			},
			&cli.StringFlag{
				Name:     "addr",
				Usage:    "Address to serve, defaults to `:8080`",
				Value:    ":8080",
				Required: false,
			},
		},
		Action: func(cCtx *cli.Context) error {
			configFile := cCtx.Path("config")
			config, err := application.ParseConfig(configFile)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to get config")
				return err
			}

			if cCtx.Bool("watch") {

				wg := sync.WaitGroup{}
				wg.Add(1) // Add the server to wait group

				// Run the file watcher in a separate goroutine
				go func() {
					err := application.WatchInputDirectory(config, processFiles)
					if err != nil {
						if errors.Is(err, os.ErrPermission) {
							log.Fatal().Err(err).Msg("Insufficient permissions")
						} else {
							log.Fatal().Err(err).Msg("Error while watching input directory")
						}
					}
				}()

				addr := cCtx.String("addr")

				err := application.ServeOutputDirectory(config.BuildPath, addr, &wg)
				if err != nil {
					if errors.Is(err, os.ErrPermission) {
						log.Fatal().Err(err).Msg("Insufficient permissions")
					} else {
						log.Fatal().Err(err).Msg("Failed to serve output directory")
					}
				}

				wg.Wait()
			} else {
				err := processFiles(config)
				if err != nil {
					if errors.Is(err, os.ErrPermission) {
						log.Fatal().Err(err).Msg("Insufficient permissions")
					} else {
						log.Fatal().Err(err).Msg("Conversion failed")
					}
				}
			}

			return nil
		},
	}

	app.Commands = []*cli.Command{initCmd, buildCommand}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err)
	}
}
