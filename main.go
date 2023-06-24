package main

import (
	"errors"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

func serveOutputDirectory(outputDir string, addr string, wg *sync.WaitGroup) error {
	// Set up file server to serve the output directory
	fs := http.FileServer(http.Dir(outputDir))
	http.Handle("/", fs)

	server := &http.Server{Addr: addr}

	log.Info().Str("addr", addr).Msg("Serving files...")

	// Create a channel to listen for signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a goroutine to handle server shutdown
	go func() {
		<-sigChan // Wait for the signal
		log.Info().Msg("Shutting down HTTP server...")

		// Shutdown the server gracefully with a timeout of 5 seconds
		err := server.Shutdown(nil)
		if err != nil {
			log.Error().Err(err).Msg("Error during server shutdown")
		}

		wg.Done() // Notify wait group that shutdown is complete
	}()

	// Start the HTTP server
	err := server.ListenAndServe()
	if err != nil {
		log.Error().Err(err).Msg("Failed to start HTTP server")
		return err
	}

	return nil
}

func watchInputDirectory(inputDir, outputDir string) error {
	// Set up signal handling to stop the watcher gracefully
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Create a new file system watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create file watcher")
		return err
	}
	defer watcher.Close()

	// Add the input directory to the watcher
	err = watcher.Add(inputDir)
	if err != nil {
		log.Error().Err(err).Msg("Failed to watch input directory")
		return err
	}

	// Initial conversion of files
	err = convertFilesToHTML(inputDir, outputDir)
	if err != nil {
		log.Error().Err(err).Msg("Failed to convert initial files to HTML")
		return err
	}

	log.Info().Msg("Watching input directory for changes...")

	// Start watching for file events
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return errors.New("file watcher closed unexpectedly")
			}

			// Only trigger conversion on file modifications or creations
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				log.Info().Str("file", event.Name).Msg("File change detected")

				err = convertFilesToHTML(inputDir, outputDir)
				if err != nil {
					log.Error().Err(err).Msg("Failed to convert files to HTML")
					return err
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return errors.New("file watcher closed unexpectedly")
			}

			log.Error().Err(err).Msg("File watcher error")
		case <-stop:
			log.Info().Msg("Stopping file watcher")
			return nil
		}
	}
}

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
			&cli.StringFlag{
				Name:     "addr",
				Usage:    "Address to serve, defaults to `:8080`",
				Value:    ":8080",
				Required: false,
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

			inputDir := cCtx.Path("input")
			outputDir := cCtx.Path("output")

			wg := sync.WaitGroup{}
			wg.Add(1) // Add the server to wait group

			if cCtx.Bool("watch") {

				// Run the file watcher in a separate goroutine
				go func() {
					err := watchInputDirectory(inputDir, outputDir)
					if err != nil {
						if errors.Is(err, os.ErrPermission) {
							log.Fatal().Err(err).Msg("Insufficient permissions")
						} else {
							log.Fatal().Err(err).Msg("Error while watching input directory")
						}
					}
				}()

				addr := cCtx.String("addr")

				err := serveOutputDirectory(outputDir, addr, &wg)
				if err != nil {
					if errors.Is(err, os.ErrPermission) {
						log.Fatal().Err(err).Msg("Insufficient permissions")
					} else {
						log.Fatal().Err(err).Msg("Failed to serve output directory")
					}
				}
			} else {
				err := convertFilesToHTML(inputDir, outputDir)
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

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err)
	}
}
