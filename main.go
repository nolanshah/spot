package main

import (
	"errors"
	"os"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"main/internal/application"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	app := &cli.App{
		Name:  "spot",
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
				zerolog.SetGlobalLevel(zerolog.TraceLevel)
				log.Debug().Msg("Debug logging enabled")
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
				log.Fatal().Err(err).Msg("Failed to get config.")
				return err
			}

			if cCtx.Bool("watch") {

				wg := sync.WaitGroup{}
				wg.Add(1) // Add the server to wait group

				// Run the file watcher in a separate goroutine
				go func() {
					err := application.WatchInputDirectory(config)
					if err != nil {
						if errors.Is(err, os.ErrPermission) {
							log.Fatal().Err(err).Msg("Insufficient permissions.")
						} else {
							log.Fatal().Err(err).Msg("Error while watching input directory.")
						}
					}
				}()

				addr := cCtx.String("addr")

				err := application.ServeOutputDirectory(config.BuildPath, addr, &wg)
				if err != nil {
					if errors.Is(err, os.ErrPermission) {
						log.Fatal().Err(err).Msg("Insufficient permissions.")
					} else {
						log.Fatal().Err(err).Msg("Failed to serve output directory.")
					}
				}

				wg.Wait()
			} else {
				err := application.ProcessFiles(config)
				if err != nil {
					if errors.Is(err, os.ErrPermission) {
						log.Fatal().Err(err).Msg("Insufficient permissions.")
					} else {
						log.Fatal().Err(err).Msg("Conversion failed.")
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
