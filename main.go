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

func processFiles(inputDir, outputDir string) error {
	err := filepath.Walk(inputDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error().Err(err).Str("file", filePath).Msg("Error accessing file")
			return err
		}

		if info.IsDir() {
			// Skip directories
			return nil
		}

		// Get the relative path of the input file
		relativePath, err := filepath.Rel(inputDir, filePath)
		if err != nil {
			log.Error().Err(err).Str("file", filePath).Msg("Failed to get relative path")
			return err
		}

		// Create the output directory structure
		outputPath := filepath.Join(outputDir, filepath.Dir(relativePath))
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			log.Error().Err(err).Str("file", filePath).Msg("Failed to create output directory structure")
			return err
		}

		extension := filepath.Ext(info.Name())
		fileName := strings.TrimSuffix(info.Name(), extension)

		if extension == ".docx" || extension == ".md" || extension == ".txt" || extension == ".ipynb" {
			err = converters.ConvertFileToHTML(inputDir, relativePath, outputDir, fileName)
			if err != nil {
				return err
			}
		} else if extension == ".webloc" {
			link, err := converters.ExtractLinkFromWebloc(inputDir, relativePath)
			if err != nil {
				return err
			}
			log.Info().Str("file", relativePath).Str("link", link).Msg("Found a webloc link, not doing anything it with.")
		} else if extension == ".lnk" {
			link, err := converters.ExtractLinkFromShortcut(inputDir, relativePath)
			if err != nil {
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
			if cCtx.Bool("debug") {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
				log.Debug().Msg("Debug logging enabled.")
			}

			configFile := cCtx.Path("config")
			config, err := application.ParseConfig(configFile)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to get config")
				return err
			}

			inputDir := config.ContentPath
			outputDir := config.BuildPath

			// copy static
			// TODO: support watching on static
			application.CopyDir(config.StaticPath, config.BuildPath)

			// convert content and apply templates
			if cCtx.Bool("watch") {

				wg := sync.WaitGroup{}
				wg.Add(1) // Add the server to wait group

				// Run the file watcher in a separate goroutine
				go func() {
					err := application.WatchInputDirectory(inputDir, outputDir, processFiles)
					if err != nil {
						if errors.Is(err, os.ErrPermission) {
							log.Fatal().Err(err).Msg("Insufficient permissions")
						} else {
							log.Fatal().Err(err).Msg("Error while watching input directory")
						}
					}
				}()

				addr := cCtx.String("addr")

				err := application.ServeOutputDirectory(outputDir, addr, &wg)
				if err != nil {
					if errors.Is(err, os.ErrPermission) {
						log.Fatal().Err(err).Msg("Insufficient permissions")
					} else {
						log.Fatal().Err(err).Msg("Failed to serve output directory")
					}
				}

				wg.Wait()
			} else {
				err := processFiles(inputDir, outputDir)
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
