package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

func convertFilesToHTML(inputDir, outputDir string) error {
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

		fileName := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))

		// Construct the output file path
		outputFile := filepath.Join(outputPath, fileName+".html")

		// Run the pandoc command to convert the file to HTML
		cmd := exec.Command("pandoc", filePath, "-o", outputFile)
		err = cmd.Run()
		if err != nil {
			log.Error().Err(err).Str("file", filePath).Msg("Failed to convert file to HTML")
			return err
		}

		log.Info().Str("file", filePath).Msg("Converted file to HTML")

		return nil
	})

	if err != nil {
		log.Error().Err(err).Msg("Error walking through directory")
		return err
	}

	return nil
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	app := &cli.App{
		Name:  "easy-static",
		Usage: "build static websites from a bunch of unstructured docs",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:     "input",
				Usage:    "input directory",
				Required: true,
			},
			&cli.PathFlag{
				Name:     "config",
				Usage:    "config directory",
				Required: false,
			},
			&cli.PathFlag{
				Name:     "output",
				Usage:    "output directory",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "watch",
				Value: false,
			},
			&cli.BoolFlag{
				Name:   "debug",
				Value:  false,
				Hidden: true,
			},
		},
		Action: func(cCtx *cli.Context) error {
			if cCtx.Bool("debug") {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
				log.Debug().Msg("Debug logging enabled.")
			}

			err := convertFilesToHTML(cCtx.Path("input"), cCtx.Path("output"))
			if err != nil {
				if errors.Is(err, os.ErrPermission) {
					log.Fatal().Err(err).Msg("Insufficient permissions")
				} else {
					log.Fatal().Err(err).Msg("Conversion failed")
				}
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err)
	}
}
